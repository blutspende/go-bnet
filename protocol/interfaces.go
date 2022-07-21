package protocol

import "net"

type Implementation interface {

	// Stop end end whatever you are doing
	Interrupt()

	// Receive - asynch.
	Receive(conn net.Conn) ([]byte, error)

	// Send - asynch
	Send(conn net.Conn, data [][]byte) (int, error)

	//  Create a new Instance of this class duplicating all settings
	NewInstance() Implementation
}

// The internal ProtocolMessage type helps to communicate within the protocol
type protocolMessageType int

const (
	DATA protocolMessageType = iota
	EOF
	ERROR
	DISCONNECT
)

type protocolMessage struct {
	Status protocolMessageType
	Data   []byte
}
