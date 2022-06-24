package net

import (
	"errors"
	"fmt"
	"github.com/pires/go-proxyproto"
	"io"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

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
}

func (s *tcpServerInstance) Receive() ([]byte, error) {
	return nil, errors.New("TCP server can't receive messages. Hint: Use another method")
}

func (s *tcpServerInstance) Run(handler intNet.Handler) {
	listener, err := net.Listen(TCPProtocol, fmt.Sprintf(":%d", s.listeningPort))
	if err != nil {
		panic(fmt.Sprintf("Can not start TCP-Server: %s", err))
	}

	rand.Seed(time.Now().Unix())

	if s.proxy == NoProxy {
		// TODO: implement if no proxy take net.listen
	}

	proxyListener := &proxyproto.Listener{Listener: listener}
	defer proxyListener.Close()

	s.isRunning = true

	for s.isRunning {
		conn, err := proxyListener.Accept()
		if err != nil {
			continue
		}

		if s.connectionCount >= s.maxConnections {
			conn.Close()
			println("max connection reached. Disconnected by TCP-Server")
			log.Println("max connection reached. Disconnected by TCP-Server")
			continue
		}

		go func() {
			s.connectionCount++
			s.readingReceivingMsg(conn, handler)
			s.connectionCount--
		}()

	}
}

func (s *tcpServerInstance) readingReceivingMsg(conn net.Conn, handler intNet.Handler) ([]byte, error) {
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

ReadLoop:
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
				continue ReadLoop
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

			case RawProtocol:
				receivedMsg = append(receivedMsg, x)
				// raw transmission ends only if the client disconnect
			case ASTMWrappedSTXProtocol:
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
