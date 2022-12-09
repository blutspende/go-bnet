package protocol

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
	"github.com/stretchr/testify/assert"
)

func TestComputeChecksum(t *testing.T) {
	message := "This is transmission text for which we need a checksum"
	frameNumber := "1"
	specialChars := []byte{utilities.ETX}

	expectedChecksum := []byte(fmt.Sprintf("%02X", 61))
	checksum := computeChecksum([]byte(frameNumber), []byte(message), specialChars)

	assert.Equal(t, expectedChecksum, checksum)

	frameNumber = "2"
	checksum = computeChecksum([]byte(frameNumber), []byte(message), specialChars)
	assert.NotEqual(t, expectedChecksum, checksum)
}

type scriptedProtocol struct {
	receiveOrSend string
	bytes         []byte
}

type mockConnection struct {
	scriptedProtocol []scriptedProtocol
	currentRecord    int
}

func (m *mockConnection) Read(b []byte) (n int, err error) {
	// fmt.Println("Trying to read...")

	if m.currentRecord >= len(m.scriptedProtocol) {
		return 0, fmt.Errorf("Script is at end, but was expecting to receive data from instrument")
	}

	if m.scriptedProtocol[m.currentRecord].receiveOrSend == "tx" {
		for i := 0; i < len(m.scriptedProtocol[m.currentRecord].bytes); i++ {
			b[i] = m.scriptedProtocol[m.currentRecord].bytes[i]
		}

		// fmt.Printf("Sending: %+v\n", m.scriptedProtocol[m.currentRecord].bytes)

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

func TestSendData(t *testing.T) {

	// this is like the instrument would behave
	var mc mockConnection
	mc.scriptedProtocol = make([]scriptedProtocol, 0)
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.ENQ}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "tx", bytes: []byte{utilities.ACK}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.STX}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte([]byte("1H||||"))})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.ETX}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{54, 67}}) // checksum
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{13, 10}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "tx", bytes: []byte{utilities.ACK}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.STX}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte([]byte("2O|1|||||"))})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.ETX}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{57, 68}}) // checksum
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{13, 10}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "tx", bytes: []byte{utilities.ACK}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.EOT}})

	mc.currentRecord = 0

	// this is "our" sid of the protocol
	message := [][]byte{}
	message = append(message, []byte("H||||"))
	message = append(message, []byte("O|1|||||"))

	os.Setenv("PROTOLOG_ENABLE", "true") // enable logging
	instance := Logger(Lis1A1Protocol(DefaultLis1A1ProtocolSettings()))

	_, err := instance.Send(&mc, message)

	assert.Nil(t, err)
}
