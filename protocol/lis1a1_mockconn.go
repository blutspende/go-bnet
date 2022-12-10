package protocol

import (
	"fmt"
	"net"
	"time"
)

type scriptedProtocol struct {
	receiveOrSend string
	bytes         []byte
}

type mockConnection struct {
	scriptedProtocol []scriptedProtocol
	currentRecord    int
}

func (m *mockConnection) Read(b []byte) (n int, err error) {

	if m.currentRecord >= len(m.scriptedProtocol) {
		return 0, fmt.Errorf("Script is at end, but was expecting to receive data from instrument")
	}

	if m.scriptedProtocol[m.currentRecord].receiveOrSend == "tx" {
		copy(b, m.scriptedProtocol[m.currentRecord].bytes)

		theLength := len(m.scriptedProtocol[m.currentRecord].bytes)
		m.currentRecord = m.currentRecord + 1
		return theLength, nil
	}

	return 0, fmt.Errorf("Testing: Expected to write data, but no data was ready. Current record: '%v' ", m.scriptedProtocol[m.currentRecord])
}

func (m *mockConnection) Write(b []byte) (n int, err error) {

	if m.currentRecord >= len(m.scriptedProtocol) {
		return 0, fmt.Errorf("Script is at end, but received unexpected data '%v'", b)
	}

	if m.scriptedProtocol[m.currentRecord].receiveOrSend == "rx" {

		if len(m.scriptedProtocol[m.currentRecord].bytes) != len(b) {
			return 0, fmt.Errorf("Testing: Expected to recieve '%v' but got '%v' (different lengths)", m.scriptedProtocol[m.currentRecord], b)
		}

		for i := 0; i < len(m.scriptedProtocol[m.currentRecord].bytes); i++ {
			if m.scriptedProtocol[m.currentRecord].bytes[i] != b[i] {
				return 0, fmt.Errorf("Testing: Expected to recieve '%v' but got '%v' (different bytes)", m.scriptedProtocol[m.currentRecord], b)
			}
		}

		// at this point we know we recieved the expected, time to move on
		m.currentRecord = m.currentRecord + 1

		// fmt.Println("Recieving:", string(b), b)
	} else {
		return 0, fmt.Errorf("Testing: Expected to recieve '%v' but got '%v'", m.scriptedProtocol[m.currentRecord], b)
	}
	return 0, nil
}

func (m *mockConnection) Close() error {
	return nil
}

func (m *mockConnection) LocalAddr() net.Addr {
	return &net.IPAddr{}
}

func (m *mockConnection) RemoteAddr() net.Addr {
	return &net.IPAddr{}
}

func (m *mockConnection) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConnection) SetWriteDeadline(t time.Time) error {
	return nil
}
