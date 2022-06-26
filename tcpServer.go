package main

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
	dataTransferType HighLevelProtocol
	proxy            ProxyType
	maxConnections   int
	timingConfig     TimingConfiguration
	isRunning        bool
	connectionCount  int
	listener         net.Listener
}

func CreateNewTCPServerInstance(listeningPort int, dataTransferType HighLevelProtocol, proxy ProxyType, maxConnections int, defaultTiming ...TimingConfiguration) ConnectionInstance {
	return &tcpServerInstance{
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
}

func (s *tcpServerInstance) Stop() {
	s.isRunning = false
	s.listener.Close()
}

func (s *tcpServerInstance) Send(data []byte) (int, error) {
	return 0, errors.New("Server instance can not send data. What are you looking for, a broadcast to all clients that are connected ?")
}

func (s *tcpServerInstance) Receive() ([]byte, error) {
	return nil, errors.New("TCP server can't receive messages. Hint: Use another method")
}

func (s *tcpServerInstance) Run(handler Handler) {
	var err error
	s.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", s.listeningPort))
	if err != nil {
		panic(fmt.Sprintf("Can not start TCP-Server: %s", err))
	}

	rand.Seed(time.Now().Unix())

	if s.proxy == NoLoadbalancer {
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
			connections = removeConnection(connections, connection)
		}()

	}

	for _, x := range connections {
		x.Close()
	}

	s.listener.Close()
}

func removeConnection(connections []net.Conn, which net.Conn) []net.Conn {

	ret := make([]net.Conn, 0)

	for _, x := range connections {
		if x != which {
			ret = append(ret, x)
		}
	}

	return ret
}

type tcpServerSession struct {
	conn             net.Conn
	isRunning        bool
	connected        bool
	sessionActive    *sync.WaitGroup
	dataTransferType HighLevelProtocol
	timingConfig     TimingConfiguration
	remoteAddr       *net.TCPAddr
}

func (instance *tcpServerInstance) handleTCPConnection(conn net.Conn, handler Handler) ([]byte, error) {

	session := &tcpServerSession{
		conn:          conn,
		isRunning:     true,
		sessionActive: &sync.WaitGroup{},
	}

	session.sessionActive.Add(1)

	go func(session *tcpServerSession, handler Handler) {
		handler.Connected(session) // blocking
		session.Close()
	}(session, handler)

	buff := make([]byte, 1500)
	receivedMsg := make([]byte, 0)
	session.remoteAddr = conn.RemoteAddr().(*net.TCPAddr)

	defer conn.Close()
	defer session.sessionActive.Done()

	for {
		if !session.isRunning || !instance.isRunning {
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
				handler.DataReceived(session, receivedMsg, time.Now())
			}
			return nil, err
		}

		if n == 0 {
			handler.DataReceived(session, receivedMsg, time.Now())
			return receivedMsg, err
		}

		for _, x := range buff[:n] {
			switch instance.dataTransferType {

			case PROTOCOL_RAW:
				receivedMsg = append(receivedMsg, x)
				// raw transmission ends only if the client disconnect
			case PROTOCOL_STXETX:
				if x == protocol.STX {
					continue
				}
				if x == protocol.ETX {
					handler.DataReceived(session, receivedMsg, time.Now())
					return receivedMsg, err
				}
				receivedMsg = append(receivedMsg, x)
			default:
				return nil, errors.New(fmt.Sprintf("This type is not implemented yet: %+v", instance.dataTransferType))
			}
		}
	}

	return receivedMsg, nil
}

func (s *tcpServerSession) IsAlive() bool {
	return false
}

func (s *tcpServerSession) Send(data []byte) (int, error) {
	return 0, errors.New("server can't send a message! ")
}

func (s *tcpServerSession) Receive() ([]byte, error) {
	return []byte{}, errors.New("you can not receive messages through the serverinstance - use the session instead")
}

func (s *tcpServerSession) Close() error {
	return nil
}

func (s *tcpServerSession) WaitTermination() error {
	return errors.New("Not implemented yet")
}

func (s *tcpServerSession) RemoteAddress() (string, error) {
	return s.remoteAddr.IP.To4().String(), nil
}
