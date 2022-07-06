package protocol

import "net"

type Implementation interface {
	Receive(conn net.Conn) ([]byte, error)
	Interrupt()
	Send(conn net.Conn, data []byte) (int, error)
}
