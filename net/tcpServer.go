package net

import (
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/pires/go-proxyproto"

	intNet "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net"
)

type tcpServerInstance struct {
	listeningPort    int
	dataTransferType DataReviveType
	proxy            ProxyType
	maxConnections   int
	timingConfig     TimingConfiguration
	isRunning        bool
	connectionCount  int
	listener         net.Listener
}

func CreateNewTCPServerInstance(listeningPort int, dataTransferType DataReviveType, proxy ProxyType, maxConnections int, defaultTiming ...TimingConfiguration) intNet.ConnectionInstance {
	tcpServerInit := &tcpServerInstance{
		listeningPort:    listeningPort,
		dataTransferType: dataTransferType,
		maxConnections:   maxConnections,
		proxy:            proxy,
		timingConfig: TimingConfiguration{
			Timeout:  time.Second * 3,
			Deadline: time.Millisecond * 200,
		},
		connectionCount: 0,
	}

	for i := range defaultTiming {
		timingConfig := defaultTiming[i]
		tcpServerInit.timingConfig = timingConfig
	}

	return tcpServerInit
}

func (s *tcpServerInstance) Stop() {
	s.isRunning = false
	s.listener.Close()
}

func (s *tcpServerInstance) Receive() ([]byte, error) {
	return nil, errors.New("TCP server can't receive messages. Hint: Use another method")
}

func (s *tcpServerInstance) Run(handler intNet.Handler) {
	var err error
	s.listener, err = net.Listen(TCPProtocol, fmt.Sprintf(":%d", s.listeningPort))
	if err != nil {
		panic(fmt.Sprintf("Can not start TCP-Server: %s", err))
	}

	rand.Seed(time.Now().Unix())

	if s.proxy == NoProxy {
		// TODO: implement if no proxy take net.listen
	}

	proxyListener := &proxyproto.Listener{Listener: s.listener}
	defer proxyListener.Close()

	s.isRunning = true

	connections := make([]net.Conn, 0)

	for s.isRunning {
		connection, err := proxyListener.Accept()
		if err != nil {
			continue
		}

		if s.connectionCount >= s.maxConnections {
			connection.Close()
			println("max connection reached. Disconnected by TCP-Server")
			log.Println("max connection reached. Disconnected by TCP-Server")
			continue
		}

		go func() {
			connections = append(connections, connection)
			s.connectionCount++
			s.handleTCPConnection(connection, handler)
			s.connectionCount--
			connections = remove(connections, connection)
		}()

	}

	for _, x := range connections {
		x.Close()
	}

	s.listener.Close()
}

func remove(connections []net.Conn, which net.Conn) []net.Conn {

	ret := make([]net.Conn, 0)

	for _, x := range connections {
		if x != which {
			ret = append(ret, x)
		}
	}

	return ret
}

func (s *tcpServerInstance) handleTCPConnection(conn net.Conn, handler intNet.Handler) ([]byte, error) {
	isSessionActive := true

	session := tcpSession{
		isRunning:     true,
		Conn:          conn,
		sessionActive: &sync.WaitGroup{},
	}

	session.sessionActive.Add(1)

	go func(session intNet.Session, handler intNet.Handler) {
		remoteIPAndPort := session.RemoteAddress()
		handler.ClientConnected(remoteIPAndPort, session) // blocking
		isSessionActive = false
	}(session, handler)

	buff := make([]byte, 1500)
	receivedMsg := make([]byte, 0)
	remoteIPAndPort := conn.RemoteAddr().(*net.TCPAddr).IP.To4().String()

	defer conn.Close()
	defer session.sessionActive.Done()

	for {
		if !isSessionActive || !s.isRunning {
			break
		}

		err := conn.SetDeadline(time.Now().Add(time.Millisecond * 200))
		if err != nil {
			return nil, err
		}

		n, err := conn.Read(buff)

		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			} else if err == io.EOF {
				handler.DataReceived(remoteIPAndPort, receivedMsg, time.Now())
			}
			return nil, err
		}

		if n == 0 {
			handler.DataReceived(remoteIPAndPort, receivedMsg, time.Now())
			return receivedMsg, err
		}

		for _, x := range buff[:n] {
			switch s.dataTransferType {

			case PROTOCOL_RAW:
				receivedMsg = append(receivedMsg, x)
				// raw transmission ends only if the client disconnect
			case PROTOCOL_STXETX:
				if x == STX {
					continue
				}
				if x == ETX {
					handler.DataReceived(remoteIPAndPort, receivedMsg, time.Now())
					return receivedMsg, err
				}
				receivedMsg = append(receivedMsg, x)
			default:
				return nil, errors.New(fmt.Sprintf("This type is not implemented yet: %s", s.dataTransferType))
			}
		}
	}

	return receivedMsg, nil
}

func (s *tcpServerInstance) Send(data []byte) (int, error) {
	return 0, errors.New("Server can't send a message! ")
}
