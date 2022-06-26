package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol"
)

/* One instance of client can only connect one
   session to a server, thus Instance = Session
   implements both interfaces
*/
type tcpClientConnectionAndSession struct {
	hostname        string
	port            int
	protocolReceive HighLevelProtocol
	protocolSend    HighLevelProtocol
	proxy           ProxyType
	timingConfig    TimingConfiguration
	conn            net.Conn
	connected       bool
	isStopped       bool
	handler         Handler
}

func CreateNewTCPClient(hostname string, port int,
	protocolReceiveve HighLevelProtocol,
	protocolSend HighLevelProtocol,
	proxy ProxyType, timing TimingConfiguration) ConnectionAndSessionInstance {
	return &tcpClientConnectionAndSession{
		hostname:        hostname,
		port:            port,
		protocolReceive: protocolReceiveve,
		protocolSend:    protocolSend,
		proxy:           proxy,
		timingConfig:    timing,
		connected:       false,
		isStopped:       false,
		handler:         nil, // is set by run
	}
}

/* Ensure the client stays connected and receives Data.
On disconnect from server the client will forever retry to connect.
Call Stop() will exit the loop  */
func (s *tcpClientConnectionAndSession) Run(handler Handler) {
	s.handler = handler
	s.isStopped = false

	s.Connect()
	for !s.isStopped {
		s.ensureConnected()
		data, err := s.Receive()
		if err != nil {
			log.Println(err)
			if err == io.EOF {
				s.Close()
				break
			} else {
				//TODO: better handling
				log.Println(err)
			}
		} else {
			handler.DataReceived(s, data, time.Now())
		}
	}

	s.handler = nil
}

func (s *tcpClientConnectionAndSession) Stop() {
	s.Close()
	s.isStopped = true
}

func (s *tcpClientConnectionAndSession) RemoteAddress() (string, error) {
	if s.conn != nil {
		if err := s.ensureConnected(); err != nil {
			return "", err
		}
	}

	if s.conn != nil {
		host, _, err := net.SplitHostPort(s.conn.RemoteAddr().String())
		return host, err
	} else {
		return "", nil
	}
}

func (s *tcpClientConnectionAndSession) Close() error {
	if s.conn != nil {
		if s.handler != nil {
			s.handler.Disconnected(s)
		}
		err := s.conn.Close()
		s.conn = nil
		if err != nil {
			if s.handler != nil {
				s.handler.Error(s, ErrorDisconnect, err)
			}
			return err
		}
	}
	return nil
}

func (s *tcpClientConnectionAndSession) IsAlive() bool {
	if s.conn != nil && s.connected {
		return true
	}
	return false
}

func (s *tcpClientConnectionAndSession) WaitTermination() error {
	return errors.New("Client connections can not be passively observed for disconnect.")
}

func (s *tcpClientConnectionAndSession) Connect() error {
	if err := s.ensureConnected(); err != nil {
		if s.handler != nil {
			go s.handler.Error(s, ErrorConnect,
				errors.New(fmt.Sprintf("failed to connect - %s", err)),
			)
		}
		return err
	}
	return nil
}

func (s *tcpClientConnectionAndSession) Receive() ([]byte, error) {

	if err := s.ensureConnected(); err != nil {
		if s.handler != nil {
			s.handler.Error(s, ErrorReceive,
				errors.New(fmt.Sprintf("failed to reconnect - %s", err)))
		}
		return nil, err
	}

	switch s.protocolReceive {
	case PROTOCOL_RAW:
		buff := make([]byte, 500)
		n, err := s.conn.Read(buff)
		return buff[:n], err
	case PROTOCLOL_LIS1A1:
		return nil, errors.New("not implemented")
	case PROTOCOL_STXETX:
		return protocol.ReceiveWrappedStxProtocol(s.conn)
	default:
		return nil, errors.New("invalid data transfer type")
	}

}

func (s *tcpClientConnectionAndSession) Send(data []byte) (int, error) {

	if err := s.ensureConnected(); err != nil {
		return 0, err
	}

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

func (s *tcpClientConnectionAndSession) ensureConnected() error {
	if s.connected || s.isStopped {
		return nil
	}

	conn, err := net.Dial("tcp", s.hostname+fmt.Sprintf(":%d", s.port))
	if err != nil {
		if s.handler != nil {
			s.handler.Error(s, ErrorConnect, err)
		}
		return err
	}
	s.conn = conn
	s.connected = true

	if s.handler != nil {
		s.handler.Connected(s)
	}

	return nil
}
