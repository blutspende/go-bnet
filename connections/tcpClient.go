package connections

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type tcpClient struct {
	hostname         string
	port             int
	dataTransferType DataReviveType
	proxy            string
	timingConfig     TimingConfiguration
	BloodLabConn
}

func CreateNewTCPClient(hostname string, port int, dataTransferType DataReviveType, proxy string, defaultTiming ...TimingConfiguration) Connections {
	tcpClientConfig := &tcpClient{
		hostname:         hostname,
		port:             port,
		dataTransferType: dataTransferType,
		proxy:            proxy,
		timingConfig: TimingConfiguration{
			Timeout:  time.Second * 3,
			Deadline: time.Millisecond * 200,
		},
	}

	for i := range defaultTiming {
		timingConfig := defaultTiming[i]
		tcpClientConfig.timingConfig = timingConfig
	}

	return tcpClientConfig
}

func (s *tcpClient) Run(handler Handlers) {
	// TODO: Check maybe other configuration
	for {
		conn, err := net.Dial(TCPProtocol, s.hostname+fmt.Sprintf(":%d", s.port))
		if err != nil {
			fmt.Println(fmt.Sprintf("Can not connect to client: %v", err))
			continue
		}

		s.BloodLabConn = BloodLabConn{
			r:    bufio.NewReader(conn),
			Conn: conn,
		}

		go handler.HandleServerSession(s.BloodLabConn)
	}
}

func (s *tcpClient) Send(data interface{}) {

}
