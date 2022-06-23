package connections

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/pires/go-proxyproto"
)

type tcpServer struct {
	listeningPort    int
	dataTransferType DataReviveType
	proxy            string
	maxConnections   int
	timingConfig     TimingConfiguration
	BloodLabConn
}

func CreateNewTCPServer(listeningPort int, dataTransferType DataReviveType, proxy string, maxConnections int, defaultTiming ...TimingConfiguration) Connections {
	tcpServerInit := &tcpServer{
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

func (s *tcpServer) Run(handler Handlers) {
	listener, err := net.Listen(TCPProtocol, fmt.Sprintf(":%d", s.listeningPort))
	if err != nil {
		panic(fmt.Sprintf("Can not start TCP-Server: %s", err))
	}

	rand.Seed(time.Now().Unix())

	proxyListener := &proxyproto.Listener{Listener: listener}
	defer proxyListener.Close()
	for {
		conn, err := proxyListener.Accept()
		if err != nil {
			continue
		}

		s.BloodLabConn = BloodLabConn{
			r:    bufio.NewReader(conn),
			Conn: conn,
		}

		if s.isHealthCheckOrScammerCall() {
			s.BloodLabConn.Close()
			continue
		}

		switch s.dataTransferType {
		case RawProtocol:
			go handler.HandleServerSession(s.BloodLabConn)
		case LIS2A2Protocol:
		// filter some data
		case ASTMWrappedSTXProtocol:
		// remove STX & ETX

		default:
			s.BloodLabConn.Close()
		}

	}
}

func (s *tcpServer) Send(data interface{}) {
	return
}

func (s *tcpServer) isHealthCheckOrScammerCall() bool {
	err := s.BloodLabConn.SetDeadline(time.Now().Add(s.timingConfig.HealthCheckSpammer))
	if err != nil {
		s.BloodLabConn.Close()
		return true
	}

	peakBuff, err := s.BloodLabConn.Peek(1)
	if err != nil {
		s.BloodLabConn.Close()
		return true
	}

	if len(peakBuff) == 0 {
		s.BloodLabConn.Close()
		return true
	}

	return false
}
