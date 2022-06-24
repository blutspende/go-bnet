package net

import (
	"net"
	"sync"
	"time"
)

const (
	TCPProtocol = "tcp"
)

type tcpSession struct {
	isRunning bool
	net.Conn
	sessionActive *sync.WaitGroup
}

func (b tcpSession) SetAlive(alive bool) {
	b.isRunning = alive
}

func (b tcpSession) IsAlive() bool {
	return b.isRunning
}

func (b tcpSession) WaitTermination() {
	b.sessionActive.Wait()
}

func (b tcpSession) Send(msg []byte) (int, error) {
	return b.Write(msg)
}

func (b tcpSession) Read(p []byte) (int, error) {
	return b.Conn.Read(p)
}

func (b tcpSession) Close() {
	b.Conn.Close()
}

func (b tcpSession) RemoteAddress() string {
	return b.Conn.RemoteAddr().(*net.TCPAddr).IP.To4().String()
}

type TimingConfiguration struct {
	Timeout            time.Duration
	Deadline           time.Duration
	HealthCheckSpammer time.Duration
}

type DataReviveType int

const (
	PROTOCOL_RAW     DataReviveType = 1
	PROTOCOL_STXETX  DataReviveType = 2
	PROTOCLOL_LIS1A1 DataReviveType = 3
)

type ProxyType int

const (
	NoProxy            ProxyType = 1
	HaProxySendProxyV2 ProxyType = 2
)

type FileNameGeneration int

const (
	Default   FileNameGeneration = 1
	TimeStamp FileNameGeneration = 1
)

type ReadFilePolicy int

const (
	Nothing ReadFilePolicy = 1
	Delete  ReadFilePolicy = 2
	Rename  ReadFilePolicy = 3
)

const (
	STX = 0x02
	ETX = 0x03
)

func wrappedStxProtocol(conn net.Conn) ([]byte, error) {
	buff := make([]byte, 100)
	receivedMsg := make([]byte, 0)
ReadLoop:
	for {

		n, err := conn.Read(buff)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue ReadLoop
			}
			return nil, err
		}
		if n == 0 {
			return receivedMsg, err
		}

		for _, x := range buff[:n] {
			if x == STX {
				continue
			}
			if x == ETX {
				return receivedMsg, err
			}
			receivedMsg = append(receivedMsg, x)
		}
	}
}
