package main

import (
	"errors"
	"fmt"
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
}

func CreateNewTCPServerInstance(listeningPort int, protocolReceiveve protocol.Implementation,
	proxy ProxyType, maxConnections int, timingConfig TimingConfiguration) ConnectionInstance {
	return &tcpServerInstance{
		listeningPort:    listeningPort,
		LowLevelProtocol: protocolReceiveve,
		maxConnections:   maxConnections,
		proxy:            proxy,
		timingConfig:     timingConfig,
		sessionCount:     0,
		handler:          nil,
		mainLoopActive:   &sync.WaitGroup{},
	}
}

func (s *tcpServerInstance) Stop() {
	s.isRunning = false
	s.listener.Close()
	s.mainLoopActive.Wait()
}

func (s *tcpServerInstance) Send(data []byte) (int, error) {
	return 0, errors.New("server instance can not send data. What are you looking for, a broadcast to all clients that are connected ? ")
}

func (s *tcpServerInstance) Receive() ([]byte, error) {
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

		session, err := createTcpServerSession(connection, handler, instance.LowLevelProtocol, instance.timingConfig)
		if err != nil {
			fmt.Errorf("error creating a new TCP session: %w", err)
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
	conn                net.Conn
	isRunning           bool
	sessionActive       *sync.WaitGroup
	timingConfig        TimingConfiguration
	remoteAddr          *net.TCPAddr
	protocolReceive     protocol.Implementation
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
		protocolReceive:     protocolReceive,
		timingConfig:        timingConfiguration,
		handler:             handler,
		remoteAddr:          conn.RemoteAddr().(*net.TCPAddr),
		blockedForSending:   &sync.Mutex{},
		blockedForReceiving: &sync.Mutex{},
		hasDataToSend:       false,
		dataToSend:          nil,
	}
	return session, nil
}

func (instance *tcpServerInstance) tcpSession(session *tcpServerSession) error {

	session.sessionActive.Add(1)

	go session.handler.Connected(session)

	defer session.Close()
	defer session.sessionActive.Done()

	for {

		if !session.isRunning || !instance.isRunning {
			break
		}

		data, err := session.protocolReceive.Receive(session.conn)

		if err != nil {
			session.handler.Error(session, ErrorReceive, err)
		} else {
			// Imporant detail : the read loop is over when DataReceived event occurs. This means
			// that at this point we can also send data
			session.handler.DataReceived(session, data, time.Now())
		}

	}

	return nil
}

func (s *tcpServerSession) IsAlive() bool {
	return s.isRunning
}

func (s *tcpServerSession) Send(data []byte) (int, error) {
	return s.protocolReceive.Send(s.conn, data)
}

func (s *tcpServerSession) Receive() ([]byte, error) {
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
	host, _, err := net.SplitHostPort(session.conn.RemoteAddr().String())
	if err != nil {
		return host, err
	}
	return host, nil
}
