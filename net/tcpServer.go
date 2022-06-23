package net

import (
	"errors"
	"fmt"
	go_bloodlab_net "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net"
	"math/rand"
	"net"
	"time"

	"github.com/pires/go-proxyproto"
)

type tcpServerInstance struct {
	listeningPort    int
	dataTransferType DataReviveType
	proxy            ProxyType
	maxConnections   int
	timingConfig     TimingConfiguration
	go_bloodlab_net.Session
}

func CreateNewTCPServerInstance(listeningPort int, dataTransferType DataReviveType, proxy ProxyType, maxConnections int, defaultTiming ...TimingConfiguration) go_bloodlab_net.ConnectionInstance {
	tcpServerInit := &tcpServerInstance{
		listeningPort:    listeningPort,
		dataTransferType: dataTransferType,
		maxConnections:   maxConnections,
		proxy:            proxy,
		timingConfig: TimingConfiguration{
			Timeout:            time.Second * 3,
			Deadline:           time.Millisecond * 200,
			HealthCheckSpammer: time.Second * 5,
		},
	}

	for i := range defaultTiming {
		timingConfig := defaultTiming[i]
		tcpServerInit.timingConfig = timingConfig
	}

	return tcpServerInit
}

func (s *tcpServerInstance) Stop() {
	// TODO: Implement this function
	panic("Not implemented ATM")
}

func (s *tcpServerInstance) Receive() ([]byte, error) {
	return nil, errors.New("TCP server can't receive messages. Hint: Use another method")
}

func (s *tcpServerInstance) Run(handler go_bloodlab_net.Handler) {
	listener, err := net.Listen(TCPProtocol, fmt.Sprintf(":%d", s.listeningPort))
	if err != nil {
		panic(fmt.Sprintf("Can not start TCP-Server: %s", err))
	}

	rand.Seed(time.Now().Unix())

	proxyListener := &proxyproto.Listener{Listener: listener}
	defer proxyListener.Close()
	for {
		//conn, err := proxyListener.Accept()
		//if err != nil {
		//	continue
		//}

		// s.Session =
		//
		//if s.isHealthCheckOrScammerCall() {
		//	s.Session.Close()
		//	continue
		//}

		switch s.dataTransferType {
		case RawProtocol:
			go handler.ClientConnected(s.Session)
		case LIS2A2Protocol:
		// filter some data
		case ASTMWrappedSTXProtocol:
		// remove STX & ETX

		default:
			s.Session.Close()
		}

	}
}

func (s *tcpServerInstance) Send(data []byte) (int, error) {
	return 0, errors.New("Server can't send a message! ")
}

//
//func (s *tcpServerInstance) isHealthCheckOrScammerCall() bool {
//	err := s.Session.SetDeadline(time.Now().Add(s.timingConfig.HealthCheckSpammer))
//	if err != nil {
//		s.Session.Close()
//		return true
//	}
//
//	peakBuff, err := s.Session.Peek(1)
//	if err != nil {
//		s.Session.Close()
//		return true
//	}
//
//	if len(peakBuff) == 0 {
//		s.BloodLabConn.Close()
//		return true
//	}
//
//	return false
//}
