package connections

import (
	"bufio"
	"net"
	"time"
)

const (
	TCPProtocol = "tcp"
)

type Connections interface {
	Run(handler Handlers)
	Send(data interface{})
}

type BloodLabConn struct {
	r     *bufio.Reader
	alive bool
	net.Conn
}

func (b BloodLabConn) SetAlive(alive bool) {
	b.alive = alive
}

func (b BloodLabConn) IsAlive() bool {
	return b.alive
}

func (b BloodLabConn) Peek(n int) ([]byte, error) {
	return b.r.Peek(n)
}

func (b BloodLabConn) Read(p []byte) (int, error) {
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
