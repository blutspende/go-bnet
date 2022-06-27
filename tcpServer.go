package bloodlabnet

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol"
	"github.com/pires/go-proxyproto"
)

type tcpServerInstance struct {
	listeningPort   int
	protocolReceive HighLevelProtocol
	protocolSend    HighLevelProtocol
	proxy           ProxyType
	maxConnections  int
	timingConfig    TimingConfiguration
	isRunning       bool
	sessionCount    int
	listener        net.Listener
	handler         Handler
	mainLoopActive  *sync.WaitGroup
}

func CreateNewTCPServerInstance(listeningPort int, protocolReceiveve HighLevelProtocol,
	protocolSend HighLevelProtocol, proxy ProxyType, maxConnections int, timingConfig TimingConfiguration) ConnectionInstance {
	return &tcpServerInstance{
		listeningPort:   listeningPort,
		protocolReceive: protocolReceiveve,
		protocolSend:    protocolSend,
		maxConnections:  maxConnections,
		proxy:           proxy,
		timingConfig:    timingConfig,
		sessionCount:    0,
		handler:         nil,
		mainLoopActive:  &sync.WaitGroup{},
	}
}

func (instance *tcpServerInstance) Stop() {
	instance.isRunning = false
	instance.listener.Close()
	instance.mainLoopActive.Wait()
}

func (instance *tcpServerInstance) Send(data []byte) (int, error) {
	return 0, errors.New("server instance can not send data. What are you looking for, a broadcast to all clients that are connected ? ")
}

func (instance *tcpServerInstance) Receive() ([]byte, error) {
	return nil, errors.New("TCP server can't receive messages. Hint: Use another method")
}

func (instance *tcpServerInstance) Run(handler Handler) {
	var err error
	instance.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", instance.listeningPort))
	if err != nil {
		panic(fmt.Errorf("can not start TCP-Server: %w", err))
	}
	instance.handler = handler

	rand.Seed(time.Now().Unix())

	proxyListener := &proxyproto.Listener{Listener: instance.listener}
	defer proxyListener.Close()

	instance.isRunning = true
	instance.mainLoopActive.Add(1)

	sessions := make([]*tcpServerSession, 0)

	for instance.isRunning {

		connection, err := proxyListener.Accept()
		if err != nil {
			if instance.handler != nil && instance.isRunning {
				go instance.handler.Error(nil, ErrorAccept, err)
			}
			continue
		}

		if instance.sessionCount >= instance.maxConnections {
			connection.Close()
			if instance.handler != nil {
				go instance.handler.Error(nil, ErrorMaxConnections, nil)
			}
			log.Println("max connection reached, forcing disconnect")
			continue
		}

		session, err := createTcpServerSession(connection, handler, instance.protocolReceive, instance.protocolSend, instance.timingConfig)
		if err != nil {
			log.Println(err)
			instance.handler.Error(session, ErrorCreateSession, err)
		} else {
			waitStartup := &sync.Mutex{}
			waitStartup.Lock()
			go func() {
				sessions = append(sessions, session)
				instance.sessionCount++
				waitStartup.Unlock()
				instance.tcpSession(session)
				instance.sessionCount--
				sessions = removeSessionFromList(sessions, session)
			}()
			waitStartup.Lock() // wait for the startup to update the sessioncounter
		}

	}

	for _, x := range sessions {
		x.Close()
	}

	instance.listener.Close()

	instance.handler = nil
	instance.mainLoopActive.Done()
}

func removeSessionFromList(connections []*tcpServerSession, which *tcpServerSession) []*tcpServerSession {

	ret := make([]*tcpServerSession, 0)

	for _, x := range connections {
		if x != which {
			ret = append(ret, x)
		}
	}

	return ret
}

type tcpServerSession struct {
	conn            net.Conn
	isRunning       bool
	sessionActive   *sync.WaitGroup
	timingConfig    TimingConfiguration
	remoteAddr      *net.TCPAddr
	protocolReceive HighLevelProtocol
	protocolSend    HighLevelProtocol
	handler         Handler
}

func createTcpServerSession(conn net.Conn, handler Handler,
	protocolReceive HighLevelProtocol, protocolSend HighLevelProtocol,
	timingConfiguration TimingConfiguration) (*tcpServerSession, error) {

	session := &tcpServerSession{
		conn:            conn,
		isRunning:       true,
		sessionActive:   &sync.WaitGroup{},
		protocolReceive: protocolReceive,
		protocolSend:    protocolSend,
		timingConfig:    timingConfiguration,
		handler:         handler,
		remoteAddr:      conn.RemoteAddr().(*net.TCPAddr),
	}
	return session, nil
}

func (instance *tcpServerInstance) tcpSession(session *tcpServerSession) error {

	session.sessionActive.Add(1)

	tcpReceiveBuffer := make([]byte, 4096)
	receivedMsg := make([]byte, 0)

	go session.handler.Connected(session)

	defer session.Close()
	defer session.sessionActive.Done()

	milliSecondsSinceLastRead := 0

	for {

		if !session.isRunning || !instance.isRunning {
			break
		}

		// flush timeout for protocol RAW.
		if instance.protocolReceive == PROTOCOL_RAW &&
			instance.timingConfig.FlushBufferTimoutMs > 0 && // disabled timout ?
			milliSecondsSinceLastRead > instance.timingConfig.FlushBufferTimoutMs &&
			len(receivedMsg) > 0 { // buffer full, time up = flush it
			session.handler.DataReceived(session, receivedMsg, time.Now())
			receivedMsg = []byte{}
		}

		if err := session.conn.SetReadDeadline(time.Now().Add(time.Millisecond * 200)); err != nil {
			return err
		}
		n, err := session.conn.Read(tcpReceiveBuffer)

		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				milliSecondsSinceLastRead = milliSecondsSinceLastRead + 200
				continue // on timeout....
			} else if err == io.EOF {
				// end of input (client disconnected)
				if instance.protocolReceive == PROTOCOL_RAW {
					// for raw protocol flush the cache on exit
					session.handler.DataReceived(session, receivedMsg, time.Now())
				}
			} else {
				if instance.isRunning {
					session.handler.Error(session, ErrorReceive, err)
				}
			}
			return err
		}

		if n == 0 {
			// handler.DataReceived(session, receivedMsg, time.Now())
			// return receivedMsg, err
			continue
		}

		milliSecondsSinceLastRead = 0

		for _, x := range tcpReceiveBuffer[:n] {
			switch instance.protocolReceive {

			case PROTOCOL_RAW:
				// raw transmission ends only if the client disconnect
				receivedMsg = append(receivedMsg, x)
			case PROTOCOL_STXETX:
				if x == protocol.STX {
					continue
				}
				if x == protocol.ETX {
					session.handler.DataReceived(session, receivedMsg, time.Now())
					receivedMsg = []byte{}
				} else {
					receivedMsg = append(receivedMsg, x)
				}
			default:
				return fmt.Errorf("this type is not implemented yet: %+v", instance.protocolReceive)
			}
		}

	}

	return nil
}

func (s *tcpServerSession) IsAlive() bool {
	return s.isRunning
}

func (s *tcpServerSession) Send(data []byte) (int, error) {
	switch s.protocolSend {
	case PROTOCOL_RAW:
		return s.conn.Write(data)
	case PROTOCLOL_LIS1A1:
		return 0, errors.New("not implemented")
	case PROTOCOL_STXETX:
		return protocol.SendWrappedStxProtocol(s.conn, data)
	default:
		return 0, errors.New("invalid data transfer type")
	}
}

func (s *tcpServerSession) Receive() ([]byte, error) {
	return []byte{}, errors.New("you can not receive messages directly, use the event-handler instead")
}

func (s *tcpServerSession) Close() error {
	if s.IsAlive() && s.conn != nil {
		if s.handler != nil {
			s.handler.Disconnected(s)
		}
		s.conn.Close()
		s.isRunning = false
		return nil
	}
	return nil
}

func (s *tcpServerSession) WaitTermination() error {
	return errors.New("not implemented yet")
}

func (s *tcpServerSession) RemoteAddress() (string, error) {
	return s.remoteAddr.IP.To4().String(), nil
}
