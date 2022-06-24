package net

import (
	"bufio"
	"net"
	"time"
)

const (
	TCPProtocol = "tcp"
)

type tcpSession struct {
	r     *bufio.Reader
	alive bool
	net.Conn
}

func (b tcpSession) SetAlive(alive bool) {
	b.alive = alive
}

func (b tcpSession) IsAlive() bool {
	return b.alive
}
func (b tcpSession) WaitTermination() {

}

func (b tcpSession) Send(msg []byte) (int, error) {
	return b.Write(msg)
}

func (b tcpSession) Peek(n int) ([]byte, error) {
	return b.r.Peek(n)
}

func (b tcpSession) Read(p []byte) (int, error) {
	return b.r.Read(p)
}

type TimingConfiguration struct {
	Timeout            time.Duration
	Deadline           time.Duration
	HealthCheckSpammer time.Duration
}

type DataReviveType string

const (
	RawProtocol            DataReviveType = "RAW"
	ASTMWrappedSTXProtocol DataReviveType = "ASTM-WRAPPED-STX"
	LIS2A2Protocol         DataReviveType = "LIS2A2"
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
