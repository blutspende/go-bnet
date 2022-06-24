package net

import (
	"errors"
	"fmt"
	go_bloodlab_net "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net"
	"net"
	"time"
)

type tcpClient struct {
	hostname         string
	port             int
	dataTransferType DataReviveType
	proxy            ProxyType
	timingConfig     TimingConfiguration
	net.Conn
	connected bool
}

func CreateNewTCPClient(hostname string, port int, dataTransferType DataReviveType, proxy ProxyType, defaultTiming ...TimingConfiguration) go_bloodlab_net.ConnectionInstance {
	tcpClientConfig := &tcpClient{
		hostname:         hostname,
		port:             port,
		dataTransferType: dataTransferType,
		proxy:            proxy,
		timingConfig: TimingConfiguration{
			Timeout:  time.Second * 3,
			Deadline: time.Millisecond * 200,
		},
		connected: false,
	}

	for i := range defaultTiming {
		timingConfig := defaultTiming[i]
		tcpClientConfig.timingConfig = timingConfig
	}

	return tcpClientConfig
}

func (s *tcpClient) Stop() {
	s.Close()
}

func (s *tcpClient) Receive() ([]byte, error) {
	err := s.reconnect()
	if err != nil {
		return nil, err
	}

	switch s.dataTransferType {
	case RawProtocol:
		buff := make([]byte, 500)
		n, err := s.Conn.Read(buff)
		return buff[:n], err
	case LIS2A2Protocol:
		return nil, errors.New("not implemented")
	case ASTMWrappedSTXProtocol:
		return wrappedStxProtocol(s.Conn)
	default:
		return nil, errors.New("invalid data transfer type")
	}
}

func (s *tcpClient) Run(handler go_bloodlab_net.Handler) {
	panic("TCP Client cant run as Server!")
}

func (s *tcpClient) Send(data []byte) (int, error) {
	err := s.reconnect()
	if err != nil {
		return 0, err
	}

	return s.Conn.Write(data)
}

func (s *tcpClient) reconnect() error {
	if s.connected {
		return nil
	}
	return s.connect()
}

func (s *tcpClient) connect() error {
	conn, err := net.Dial(TCPProtocol, s.hostname+fmt.Sprintf(":%d", s.port))
	if err != nil {
		return err
	}

	s.Conn = conn
	s.connected = true
	return nil
}
