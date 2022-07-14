package protocol

import (
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"time"

	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
)

var (
	ReceiverDoesNotRespond   = errors.New("receiver does not react on sent message")
	ReceivedMessageIsInvalid = errors.New("received message is invalid")
)

type Lis1A1ProtocolSettings struct {
	expectFrameNumbers       bool
	strictChecksumValidation bool
	sendTimeoutDuration      time.Duration
	strictFrameOrder         bool
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

func (s Lis1A1ProtocolSettings) EnableStrictFrameOrder() Lis1A1ProtocolSettings {
	s.strictFrameOrder = true
	return s
}

func (s Lis1A1ProtocolSettings) DisableStrictFrameOrder() Lis1A1ProtocolSettings {
	s.strictFrameOrder = false
	return s
}

func (s Lis1A1ProtocolSettings) SetSendTimeOutDuration(timeout time.Duration) Lis1A1ProtocolSettings {
	s.sendTimeoutDuration = timeout
	return s
}

type ProcessState struct {
	State              int
	LastMessage        string
	LastChecksum       string
	MessageLog         []string
	ProtocolMessage    protocolMessage
	currentFrameNumber string
}

const (
	LineReceived utilities.ActionCode = "LineReceived"
	JustAck      utilities.ActionCode = "JustAck"
	FrameNumber  utilities.ActionCode = "FrameNumber"
)

var Rules []utilities.Rule = []utilities.Rule{
	{FromState: utilities.Init, Symbols: []byte{utilities.EOT}, ToState: utilities.Init, Scan: false},
	{FromState: utilities.Init, Symbols: []byte{utilities.ENQ}, ToState: 1, Scan: false, ActionCode: JustAck},

	{FromState: 1, Symbols: []byte{utilities.STX}, ToState: 99, Scan: false},

	{FromState: 99, Symbols: []byte("01234567"), ToState: 100, Scan: true, ActionCode: FrameNumber},
	{FromState: 99, Symbols: utilities.PrintableChars8Bit, ToState: 2, Scan: true, ActionCode: FrameNumber}, //  TODO: exclude 0...7
	{FromState: 100, Symbols: utilities.PrintableChars8Bit, ToState: 2, Scan: true},

	{FromState: 2, Symbols: utilities.PrintableChars8Bit, ToState: 2, Scan: true},
	{FromState: 2, Symbols: []byte{utilities.ETX}, ToState: 10, Scan: false, ActionCode: LineReceived},
	{FromState: 2, Symbols: []byte{utilities.ETB}, ToState: 20, Scan: false, ActionCode: LineReceived},

	{FromState: 20, Symbols: []byte("0123456789ABCDEFabcdef"), ToState: 20, Scan: true},
	{FromState: 20, Symbols: []byte{utilities.CR}, ToState: 21, Scan: false, ActionCode: utilities.CheckSum},
	{FromState: 21, Symbols: []byte{utilities.LF}, ToState: 22, Scan: false, ActionCode: JustAck},
	{FromState: 22, Symbols: []byte{utilities.STX}, ToState: 99, Scan: false},

	{FromState: 10, Symbols: []byte("0123456789ABCDEFabcdef"), ToState: 10, Scan: true},
	{FromState: 10, Symbols: []byte{utilities.CR}, ToState: 11, Scan: false, ActionCode: utilities.CheckSum},
	{FromState: 11, Symbols: []byte{utilities.LF}, ToState: 12, Scan: false, ActionCode: JustAck},
	{FromState: 12, Symbols: []byte{utilities.STX}, ToState: 99, Scan: false},
	{FromState: 12, Symbols: []byte{utilities.EOT}, ToState: utilities.Init, Scan: false, ActionCode: utilities.Finish},
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
	settings.sendTimeoutDuration = 30
	settings.strictFrameOrder = false
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
		fullMsg = append(fullMsg, []byte(messageLine)...)
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
		nextExpectedFrameNumber := 1

		// init state machine
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
					protocolMsg := protocolMessage{
						Status: ERROR,
						Data:   []byte("Internal error"),
					}

					if err != nil {
						protocolMsg.Data = []byte(err.Error())
					}

					_, err = conn.Write([]byte{utilities.NAK})
					if err != nil {
						protocolMsg.Data = append(protocolMsg.Data, []byte(err.Error())...)
					}

					proto.receiveQ <- protocolMsg
					proto.receiveThreadIsRunning = false
					return

				case LineReceived:
					// append Data
					lastMessage = messageBuffer
					fileBuffer = append(fileBuffer, lastMessage)
					fsm.ResetBuffer()

				case utilities.CheckSum:

					if proto.settings.strictChecksumValidation {
						currentChecksum := computeChecksum([]byte(proto.state.currentFrameNumber), lastMessage, []byte{utilities.ETX})

						if string(currentChecksum) != string(messageBuffer) {
							protocolMsg := protocolMessage{
								Status: ERROR,
								Data:   []byte(fmt.Sprintf("Invalid Checksum. want: %s given: %s ", string(currentChecksum), string(messageBuffer))),
							}

							_, err = conn.Write([]byte{utilities.NAK})
							if err != nil {
								protocolMsg.Data = append(protocolMsg.Data, []byte(err.Error())...)
							}

							proto.receiveQ <- protocolMsg
							proto.receiveThreadIsRunning = false
							return
						}
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
				case FrameNumber:
					if proto.settings.strictFrameOrder && string(ascii) != strconv.Itoa(nextExpectedFrameNumber) {
						// Check valid frame number
						protocolMsg := protocolMessage{
							Status: ERROR,
							Data:   []byte(fmt.Sprintf("invalid Frame number. currentFrameNumber: %s expectedFrameNumber: %s", string(ascii), strconv.Itoa(nextExpectedFrameNumber))),
						}
						_, err = conn.Write([]byte{utilities.NAK})
						if err != nil {
							protocolMsg.Data = append(protocolMsg.Data, []byte(err.Error())...)
						}
						proto.receiveQ <- protocolMsg
						proto.receiveThreadIsRunning = false
						return
					}

					matched, err := regexp.MatchString("[0-7]", string(ascii))
					if err != nil {
						println(fmt.Sprintf("Can not use search for number in incoming byte: %s", err.Error()))
					}

					if matched {
						nextExpectedFrameNumber = (nextExpectedFrameNumber + 1) % 8
						proto.state.currentFrameNumber = string(messageBuffer[0])
						fsm.ResetBuffer()
					}
				default:
					protocolMsg := protocolMessage{
						Status: ERROR,
						Data:   []byte("Invalid action code "),
					}

					_, err = conn.Write([]byte{utilities.NAK})
					if err != nil {
						protocolMsg.Data = append(protocolMsg.Data, []byte(err.Error())...)
					}
					proto.receiveQ <- protocolMsg
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

//  https://wiki.bloodlab.org/lib/exe/fetch.php?media=listnode:lis1-a.pdf
func (proto *lis1A1) send(conn net.Conn, data [][]byte, recursionDepth int) (int, error) {
	if recursionDepth > 10 {
		return -1, fmt.Errorf("the receiver does not accept any data")
	}

	_, err := conn.Write([]byte{utilities.ENQ}) // 8.2.4
	if err != nil {
		return -1, err
	}

	for {
		err = conn.SetDeadline(time.Now().Add(time.Second * proto.settings.sendTimeoutDuration))
		if err != nil {
			return -1, ReceiverDoesNotRespond
		}

		receivingMsg := make([]byte, 1)
		n, err := conn.Read(receivingMsg)
		if err != nil {
			return n, err
		}

		if n == 1 {
			switch receivingMsg[0] {
			case utilities.ACK: // 8.2.5
				break // continue operation
			case utilities.NAK: // 8.2.6
				return -1, fmt.Errorf("instrument(lis1a1) did not accept any data")
			case utilities.ENQ: // 8.2.7.1, 2
				time.Sleep(time.Second)
				return proto.send(conn, data, recursionDepth+1)
			default:
				continue // ignore all characters until ACK / 8.2.5
			}
		}

		// 8.3 - Transfer Phase
		maxLengthOfFrame := 63993
		frameNumber := 1
		bytesTransferred := 0
		var checksum []byte
		for _, frame := range data {
			if len(frame) > maxLengthOfFrame {

				iterations := len(frame) / maxLengthOfFrame
				if (iterations * maxLengthOfFrame) < len(frame) {
					iterations++
				}

				// take first 63993 bytes for each run
				for i := 0; i < iterations; i++ {
					copiedFrame := make([]byte, len(frame)+1)
					copy(copiedFrame, []byte(strconv.Itoa(frameNumber)))
					copy(copiedFrame[1:], frame)

					subPartOfFrame := copiedFrame[iterations*maxLengthOfFrame:]
					// If the last iteration of this frame send ETX
					usedEndByte := []byte{utilities.ETB}
					if i >= iterations {
						usedEndByte = []byte{utilities.ETX}
					}

					checksum = computeChecksum([]byte{}, subPartOfFrame, usedEndByte) //  frameNumber is an empty []of bytes because it's already set to
					_, err = conn.Write([]byte{utilities.STX})
					if err != nil {
						return -1, err
					}

					_, err = conn.Write(subPartOfFrame)
					if err != nil {
						return -1, err
					}

					_, err = conn.Write(usedEndByte)
					if err != nil {
						return -1, err
					}
					_, err = conn.Write(checksum)
					if err != nil {
						return -1, err
					}

					_, err = conn.Write([]byte{utilities.CR, utilities.LF})
					if err != nil {
						return -1, err
					}

					receivedMsg, err := proto.receiveSendAnswer(conn)
					if err != nil {
						return 0, err
					}

					switch receivedMsg {
					case utilities.ACK:
						// was successfully do next
						bytesTransferred += len(subPartOfFrame)
						bytesTransferred += len(checksum)
						bytesTransferred += 3 // cr, lf stx and endByte
					case utilities.NAK:
						// Last was not successfully do next
					case utilities.EOT:
						// Cancel after that one
						bytesTransferred += len(subPartOfFrame)
						bytesTransferred += len(checksum)
						bytesTransferred += 3 // cr, lf stx and endByte
						return bytesTransferred, nil
					default:
						return 0, ReceivedMessageIsInvalid
					}

					frameNumber = updateFrameNumber(frameNumber)
				}
			} else {
				checksum = computeChecksum([]byte{}, frame, []byte{utilities.ETX}) //  frameNumber is an empty [] of bytes because it's already set to
				_, err = conn.Write([]byte{utilities.STX})
				if err != nil {
					return -1, err
				}

				_, err = conn.Write(frame)
				if err != nil {
					return -1, err
				}

				_, err = conn.Write([]byte{utilities.ETX})
				if err != nil {
					return -1, err
				}
				_, err = conn.Write(checksum)
				if err != nil {
					return -1, err
				}

				_, err = conn.Write([]byte{utilities.CR, utilities.LF})
				if err != nil {
					return -1, err
				}
			}

			receivedMsg, err := proto.receiveSendAnswer(conn)
			if err != nil {
				return 0, err
			}

			switch receivedMsg {
			case utilities.ACK:
				bytesTransferred += len(frame)
				bytesTransferred += len(checksum)
				bytesTransferred += 3 // cr, lf stx and endByte
				continue              // was successfully do next
			case utilities.NAK:
				continue // Last was not successfully do next
			case utilities.EOT:
				bytesTransferred += len(frame)
				bytesTransferred += len(checksum)
				bytesTransferred += 3        // cr, lf stx and endByte
				return bytesTransferred, nil // Cancel after that one
			default:
				return 0, ReceivedMessageIsInvalid
			}
			frameNumber = updateFrameNumber(frameNumber)
		}
		return bytesTransferred, nil
	}
}

func updateFrameNumber(frameNumber int) int {
	if (frameNumber % 7) == 0 {
		return 0
	}
	frameNumber++
	return frameNumber
}

func (proto *lis1A1) receiveSendAnswer(conn net.Conn) (byte, error) {
	err := conn.SetDeadline(time.Now().Add(time.Second * proto.settings.sendTimeoutDuration))
	if err != nil {
		return 0, ReceiverDoesNotRespond
	}

	receivingMsg := make([]byte, 1)
	_, err = conn.Read(receivingMsg)
	if err != nil {
		return 0, err
	}

	switch receivingMsg[0] {
	case utilities.ACK:
		// was successfully do next
		return utilities.ACK, nil
	case utilities.NAK:
		// Last was not successfully do next
		return utilities.NAK, nil
	case utilities.EOT:
		// Cancel after that one
		return utilities.EOT, nil
	default:
		return 0, ReceivedMessageIsInvalid
	}

}

func (proto *lis1A1) Send(conn net.Conn, data [][]byte) (int, error) {
	return proto.send(conn, data, 1)
}

// ComputeChecksum Helper to compute the ASTM-Checksum
func computeChecksum(frameNumber, record, specialChars []byte) []byte {
	sum := int(0)
	for _, b := range frameNumber {
		sum += int(b)
	}
	for _, b := range record {
		sum += int(b)
	}
	for _, b := range specialChars {
		sum += int(b)
	}

	sum = sum % 256
	return []byte(fmt.Sprintf("%02X", sum))
}
