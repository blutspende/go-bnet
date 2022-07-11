package protocol

import (
	"fmt"
	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
	"io"
	"net"
)

type Lis1A1ProtocolSettings struct {
	expectFrameNumbers       bool
	strictChecksumValidation bool
}

func (s Lis1A1ProtocolSettings) EnableStrictChecksum() Lis1A1ProtocolSettings {
	s.strictChecksumValidation = true
	return s
}

func (s Lis1A1ProtocolSettings) DisableStrictChecksum() Lis1A1ProtocolSettings {
	s.strictChecksumValidation = false
	return s
}

func (s Lis1A1ProtocolSettings) EnableFrameNumber() Lis1A1ProtocolSettings {
	s.expectFrameNumbers = true
	return s
}

func (s Lis1A1ProtocolSettings) DisableFrameNumber() Lis1A1ProtocolSettings {
	s.expectFrameNumbers = false
	return s
}

type ProcessState struct {
	State           int
	LastMessage     string
	LastChecksum    string
	MessageLog      []string
	ProtocolMessage protocolMessage
}

const (
	LineReceived utilities.ActionCode = "LineReceived"
	JustAck      utilities.ActionCode = "JustAck"
)

var Rules []utilities.Rule = []utilities.Rule{
	{FromState: utilities.Init, Symbols: []byte{utilities.ENQ}, ToState: 1, Scan: false, ActionCode: JustAck},
	{FromState: 1, Symbols: []byte{utilities.STX}, ToState: 2, Scan: false},

	{FromState: 2, Symbols: []byte{utilities.ETB}, ToState: 5, Scan: false},
	{FromState: 5, Symbols: []byte{utilities.STX}, ToState: 2, Scan: false},

	{FromState: 2, Symbols: utilities.PrintableChars8Bit, ToState: 2, Scan: true},
	{FromState: 2, Symbols: []byte{utilities.CR}, ToState: 3, Scan: false, ActionCode: LineReceived},
	{FromState: 3, Symbols: []byte{utilities.ETX}, ToState: 7, Scan: false},
	{FromState: 3, Symbols: []byte{utilities.ETB}, ToState: utilities.Init, Scan: false},

	{FromState: 7, Symbols: []byte("01234567890ABCDEFabcdef"), ToState: 7, Scan: true},
	{FromState: 7, Symbols: []byte{utilities.CR}, ToState: 9, Scan: false, ActionCode: utilities.CheckSum},
	{FromState: 9, Symbols: []byte{utilities.LF}, ToState: 10, Scan: false, ActionCode: JustAck},

	{FromState: 10, Symbols: []byte{utilities.STX}, ToState: 2, Scan: false},
	{FromState: 10, Symbols: []byte{utilities.ETB}, ToState: utilities.Init, Scan: false},
	{FromState: 10, Symbols: []byte{utilities.EOT}, ToState: utilities.Init, Scan: false, ActionCode: utilities.Finish},
}

type lis1A1 struct {
	settings               *Lis1A1ProtocolSettings
	receiveQ               chan protocolMessage
	receiveThreadIsRunning bool
	state                  ProcessState
}

func DefaultLis1A1ProtocolSettings() *Lis1A1ProtocolSettings {
	var settings Lis1A1ProtocolSettings
	settings.expectFrameNumbers = true
	settings.strictChecksumValidation = true
	return &settings
}

func Lis1A1Protocol(settings ...*Lis1A1ProtocolSettings) Implementation {

	var theSettings *Lis1A1ProtocolSettings
	if len(settings) >= 1 {
		theSettings = settings[0]
	} else {
		theSettings = DefaultLis1A1ProtocolSettings()
	}

	return &lis1A1{
		settings:               theSettings,
		receiveQ:               make(chan protocolMessage),
		receiveThreadIsRunning: false,
	}
}

func (proto *lis1A1) Receive(conn net.Conn) ([]byte, error) {

	proto.ensureReceiveThreadRunning(conn)

	message := <-proto.receiveQ

	switch message.Status {
	case DATA:
		return message.Data, nil
	case EOF:
		return []byte{}, io.EOF
	case ERROR:
		return []byte{}, fmt.Errorf("error while reading - abort receiving data: %s", string(message.Data))
	default:
		return []byte{}, fmt.Errorf("internal error: Invalid status of communication (%d) - abort", message.Status)
	}
}

func (proto *lis1A1) transferMessageToHandler(messageLog [][]byte) {
	// looks like message is successfully transferred
	fullMsg := make([]byte, 0)
	for _, messageLine := range messageLog {
		if proto.settings.expectFrameNumbers && len(messageLine) > 0 {
			fullMsg = append(fullMsg, []byte(messageLine[1:])...)
		} else {
			fullMsg = append(fullMsg, []byte(messageLine)...)
		}

		fullMsg = append(fullMsg, utilities.CR)
	}

	proto.receiveQ <- protocolMessage{
		Status: DATA,
		Data:   fullMsg,
	}
}

// asynchronous receive loop
func (proto *lis1A1) ensureReceiveThreadRunning(conn net.Conn) {

	if proto.receiveThreadIsRunning {
		return
	}

	go func() {
		proto.receiveThreadIsRunning = true

		proto.state.State = 0 // initial state for FSM
		lastMessage := make([]byte, 0)
		fileBuffer := make([][]byte, 0)

		tcpReceiveBuffer := make([]byte, 4096)

		// init automate
		fsm := utilities.CreateFSM(Rules)
		for {
			n, err := conn.Read(tcpReceiveBuffer)
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue // on timeout....
				} else if opErr, ok := err.(*net.OpError); ok && opErr.Op == "read" {
					proto.receiveThreadIsRunning = false
					return
				} else if err == io.EOF { // EOF = silent exit
					proto.receiveThreadIsRunning = false
					return
				}
				proto.receiveThreadIsRunning = false
				return
			}

			for _, ascii := range tcpReceiveBuffer[:n] {
				messageBuffer, action, err := fsm.Push(ascii)
				if err != nil {
					proto.receiveQ <- protocolMessage{
						Status: ERROR,
						Data:   []byte(err.Error()),
					}
					proto.receiveThreadIsRunning = false
					return
				}
				switch action {
				case utilities.Ok:
				case utilities.Error:
					// error
					proto.receiveQ <- protocolMessage{
						Status: ERROR,
						Data:   []byte("Internal error"),
					}
					proto.receiveThreadIsRunning = false
					return

				case LineReceived:
					// append Data
					lastMessage = messageBuffer
					fileBuffer = append(fileBuffer, lastMessage)
					fsm.ResetBuffer()
				case utilities.CheckSum:
					currentChecksum := computeChecksum(string(lastMessage))
					if string(currentChecksum) != string(messageBuffer) {
						proto.receiveQ <- protocolMessage{
							Status: ERROR,
							Data:   []byte("Invalid Checksum"),
						}
						proto.receiveThreadIsRunning = false
						return
					}

					lastMessage = make([]byte, 0)
					fsm.ResetBuffer()
				case utilities.Finish:
					// send fileData
					proto.transferMessageToHandler(fileBuffer)
					proto.receiveThreadIsRunning = false
					return
				case JustAck:
					conn.Write([]byte{utilities.ACK})

				default:
					proto.receiveQ <- protocolMessage{
						Status: ERROR,
						Data:   []byte("Invalid action code"),
					}
					proto.receiveThreadIsRunning = false
					return
				}
			}
		}
	}()
}

func (proto *lis1A1) Interrupt() {
	// not implemented (not required neither)
}

func (proto *lis1A1) Send(conn net.Conn, data []byte) (int, error) {
	sendBytes := make([]byte, len(data)+2)
	sendBytes[0] = utilities.STX
	for i := 0; i < len(data); i++ {
		sendBytes[i+1] = data[i]
	}
	sendBytes[len(data)+1] = utilities.ETX
	return conn.Write(sendBytes)
}

// ComputeChecksum Helper to compute the ASTM-Checksum
func computeChecksum(record string) []byte {
	sum := int(0)

	for _, b := range []byte(record) {
		sum = sum + int(b)
	}
	sum += int(utilities.CR)
	sum += int(utilities.ETX)
	sum = sum % 256
	//fmt.Println("Checksum:", fmt.Sprintf("hex:%x dec:%d", sum, sum))
	return []byte(fmt.Sprintf("%02X", sum))
}
