package protocol

import "net"

type Implementation interface {
	Receive(conn net.Conn) ([]byte, error)
	Interrupt()
	Send(conn net.Conn, data [][]byte) (int, error)
}

// The internal ProtocolMessage type helps to communicate within the protocol
type protocolMessageType int

const (
	DATA protocolMessageType = iota
	EOF
	ERROR
)

type protocolMessage struct {
	Status protocolMessageType
	Data   []byte
}
