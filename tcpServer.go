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
	listeningPort    int
	LowLevelProtocol protocol.Implementation
	proxy            ProxyType
	maxConnections   int
	timingConfig     TimingConfiguration
	isRunning        bool
	sessionCount     int
	listener         net.Listener
	handler          Handler
	mainLoopActive   *sync.WaitGroup
	sessions         []*tcpServerSession
}

func CreateNewTCPServerInstance(listeningPort int, protocolReceiveve protocol.Implementation,
	proxy ProxyType, maxConnections int, timingConfig ...TimingConfiguration) ConnectionInstance {

	var thetimingConfig TimingConfiguration
	if len(timingConfig) == 0 {
		thetimingConfig = DefaultTCPServerSettings
	}

	return &tcpServerInstance{
		listeningPort:    listeningPort,
		LowLevelProtocol: protocolReceiveve,
		maxConnections:   maxConnections,
		proxy:            proxy,
		timingConfig:     thetimingConfig,
		sessionCount:     0,
		handler:          nil,
		mainLoopActive:   &sync.WaitGroup{},
		sessions:         make([]*tcpServerSession, 0),
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

		session, err := createTcpServerSession(connection, handler, instance.LowLevelProtocol, instance.timingConfig)
		if err != nil {
			fmt.Errorf("error creating a new TCP session: %w", err)
			instance.handler.Error(session, ErrorCreateSession, err)
		} else {
			waitStartup := &sync.Mutex{}
			waitStartup.Lock()
			go func() {
				instance.sessions = append(instance.sessions, session)
				instance.sessionCount++
				waitStartup.Unlock()
				instance.tcpSession(session)
				instance.sessionCount--
				instance.sessions = removeSessionFromList(instance.sessions, session)
			}()
			waitStartup.Lock() // wait for the startup to update the sessioncounter
		}

	}

	for _, x := range instance.sessions {
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

func (instance *tcpServerInstance) FindSessionsByIp(ip string) []Session {
	sessions := make([]Session, 0)

	for _, x := range instance.sessions {
		if x.remoteAddr == ip {
			sessions = append(sessions, x)
		}
	}

	return sessions
}

type tcpServerSession struct {
	conn                net.Conn
	isRunning           bool
	sessionActive       *sync.WaitGroup
	timingConfig        TimingConfiguration
	remoteAddr          string
	lowLevelProtocol    protocol.Implementation
	handler             Handler
	blockedForSending   *sync.Mutex
	blockedForReceiving *sync.Mutex
	hasDataToSend       bool
	dataToSend          *[]byte
}

func createTcpServerSession(conn net.Conn, handler Handler,
	protocolReceive protocol.Implementation,
	timingConfiguration TimingConfiguration) (*tcpServerSession, error) {

	session := &tcpServerSession{
		conn:                conn,
		isRunning:           true,
		sessionActive:       &sync.WaitGroup{},
		lowLevelProtocol:    protocolReceive,
		timingConfig:        timingConfiguration,
		handler:             handler,
		remoteAddr:          "",
		blockedForSending:   &sync.Mutex{},
		blockedForReceiving: &sync.Mutex{},
		hasDataToSend:       false,
		dataToSend:          nil,
	}
	return session, nil
}

func (instance *tcpServerInstance) tcpSession(session *tcpServerSession) error {

	session.sessionActive.Add(1)

	defer session.Close()
	defer session.sessionActive.Done()

	host, _, _ := net.SplitHostPort(session.conn.RemoteAddr().String())
	session.remoteAddr = host

	go session.handler.Connected(session)

	for {

		if !session.isRunning || !instance.isRunning {
			break
		}

		data, err := session.lowLevelProtocol.Receive(session.conn)
		if err != nil {
			if err == io.EOF {
				// EOF is not an error, its a disconnect in TCP-terms: clean exit
				break
			}
			session.handler.Error(session, ErrorReceive, err)
			session.isRunning = false
		} else {
			// Important detail : the read loop is over when DataReceived event occurs. This means
			// that at this point we can also send data
			session.handler.DataReceived(session, data, time.Now())
		}

	}

	return nil
}

func (session *tcpServerSession) IsAlive() bool {
	return session.isRunning
}

func (session *tcpServerSession) Send(data []byte) (int, error) {
	return session.lowLevelProtocol.Send(session.conn, data)
}

func (session *tcpServerSession) Receive() ([]byte, error) {
	return []byte{}, errors.New("you can not receive messages directly, use the event-handler instead")
}

func (session *tcpServerSession) Close() error {
	if session.IsAlive() && session.conn != nil {
		if session.handler != nil {
			session.handler.Disconnected(session)
		}
		session.conn.Close()
		session.conn = nil
		session.isRunning = false
		return nil
	}
	return nil
}

func (session *tcpServerSession) WaitTermination() error {
	return errors.New("not implemented yet")
}

func (session *tcpServerSession) RemoteAddress() (string, error) {
	return session.remoteAddr, nil
}
