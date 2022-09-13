package protocol

import (
	"fmt"
	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
	"io"
	"net"
	"time"
)

type BeckmanSpecialProtocolSettings struct {
	strictChecksumValidation bool
	sendTimeoutDuration      time.Duration
	startByte                byte
	endByte                  byte
	lineBreak                byte
}

func (s BeckmanSpecialProtocolSettings) SetLineBreakByte(lineBreak byte) *BeckmanSpecialProtocolSettings {
	s.lineBreak = lineBreak
	return &s
}

func (s BeckmanSpecialProtocolSettings) SetStartByte(start byte) *BeckmanSpecialProtocolSettings {
	s.startByte = start
	return &s
}

func (s BeckmanSpecialProtocolSettings) SetEndByte(end byte) *BeckmanSpecialProtocolSettings {
	s.endByte = end
	return &s
}

func (s BeckmanSpecialProtocolSettings) SetSendTimeoutDuration(duration time.Duration) *BeckmanSpecialProtocolSettings {
	s.sendTimeoutDuration = duration
	return &s
}

func (s BeckmanSpecialProtocolSettings) EnableChecksumValidation() *BeckmanSpecialProtocolSettings {
	s.strictChecksumValidation = true
	return &s
}

func (s BeckmanSpecialProtocolSettings) DisableChecksumValidation() *BeckmanSpecialProtocolSettings {
	s.strictChecksumValidation = false
	return &s
}

func DefaultBeckmanSpecialProtocolSettings() *BeckmanSpecialProtocolSettings {
	return &BeckmanSpecialProtocolSettings{
		strictChecksumValidation: false,
		sendTimeoutDuration:      time.Second * 60,
		startByte:                utilities.STX,
		endByte:                  utilities.ETX,
		lineBreak:                utilities.CR,
	}
}

type processState struct {
	State           int
	LastMessage     string
	LastChecksum    string
	MessageLog      []string
	ProtocolMessage protocolMessage
}

type beckmanSpecialProtocol struct {
	settings               *BeckmanSpecialProtocolSettings
	receiveThreadIsRunning bool
	receiveQ               chan protocolMessage
	state                  processState
}

func BeckmanSpecialProtocol(settings ...*BeckmanSpecialProtocolSettings) Implementation {
	var theSettings *BeckmanSpecialProtocolSettings
	if len(settings) >= 1 {
		theSettings = settings[0]
	} else {
		theSettings = DefaultBeckmanSpecialProtocolSettings()
	}

	return &beckmanSpecialProtocol{
		settings: theSettings,
		receiveQ: make(chan protocolMessage),
	}
}

func (p *beckmanSpecialProtocol) Interrupt() {
	//TODO implement me
	panic("implement me")
}

func (p *beckmanSpecialProtocol) generateRules() []utilities.Rule {
	var printableChars8BitWithoutE = []byte{10, 13, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, 62, 63, 64, 65, 66, 67, 68, 70, 71, 72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 88, 89, 90, 91, 92, 93, 94, 95, 96, 97, 98, 99, 100, 101, 102, 103, 104, 105, 106, 107, 108, 109, 110, 111, 112, 113, 114, 115, 116, 117, 118, 119, 120, 121, 122, 123, 124, 125, 126, 127, 128, 129, 130, 131, 132, 133, 134, 135, 136, 137, 138, 139, 140, 141, 142, 143, 144, 145, 146, 147, 148, 149, 150, 151, 152, 153, 154, 155, 156, 157, 158, 159, 160, 161, 162, 163, 164, 165, 166, 167, 168, 169, 170, 171, 172, 173, 174, 175, 176, 177, 178, 179, 180, 181, 182, 183, 184, 185, 186, 187, 188, 189, 190, 191, 192, 193, 194, 195, 196, 197, 198, 199, 200, 201, 202, 203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237, 238, 239, 240, 241, 242, 243, 244, 245, 246, 247, 248, 249, 250, 251, 252, 253, 254, 255}

	// CHECK For If CheckSumCheck is enabled
	return []utilities.Rule{
		utilities.Rule{FromState: 0, Symbols: []byte{p.settings.startByte}, ToState: 1, ActionCode: JustAck, Scan: false},
		utilities.Rule{FromState: 1, Symbols: []byte{'D', 'S', 'R', 'd'}, ToState: 2, Scan: true},
		utilities.Rule{FromState: 2, Symbols: printableChars8BitWithoutE, ToState: 3, Scan: true},
		utilities.Rule{FromState: 3, Symbols: []byte{p.settings.endByte}, ToState: 6, ActionCode: LineReceived, Scan: false},
		utilities.Rule{FromState: 3, Symbols: utilities.PrintableChars8Bit, ToState: 3, Scan: true},
		//utilities.Rule{FromState: 5, Symbols: utilities.PrintableChars8Bit, ToState: 6, ActionCode: utilities.CheckSum, Scan: true},
		utilities.Rule{FromState: 6, Symbols: []byte{p.settings.startByte}, ToState: 1, ActionCode: JustAck, Scan: false},

		utilities.Rule{FromState: 2, Symbols: []byte{'E'}, ToState: 7, Scan: true},
		utilities.Rule{FromState: 7, Symbols: []byte{p.settings.endByte}, ToState: 0, ActionCode: utilities.Finish, Scan: false},
		utilities.Rule{FromState: 7, Symbols: utilities.PrintableChars8Bit, ToState: 7, Scan: true},
		//utilities.Rule{FromState: 8, Symbols: utilities.PrintableChars8Bit, ToState: 9, ActionCode: utilities.CheckSum, Scan: true},
		utilities.Rule{FromState: 9, Symbols: utilities.PrintableChars8Bit, ToState: 0, ActionCode: utilities.Finish, Scan: false},
	}
}

func (p *beckmanSpecialProtocol) ensureReceiveThreadRunning(conn net.Conn) {

	if p.receiveThreadIsRunning {
		return
	}

	go func() {
		p.receiveThreadIsRunning = true
		p.state.State = 0

		lastMessage := make([]byte, 0)
		tcpReceiveBuffer := make([]byte, 4096)
		fileBuffer := make([][]byte, 0)

		fsm := utilities.CreateFSM(p.generateRules())
		for {
			n, err := conn.Read(tcpReceiveBuffer)
			// enabled FSM
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					fsm.Init()
					continue // on timeout....
				} else if opErr, ok := err.(*net.OpError); ok && opErr.Op == "read" {
					p.receiveThreadIsRunning = false
					p.receiveQ <- protocolMessage{
						Status: DISCONNECT,
						Data:   []byte(err.Error()),
					}
					return
				} else if err == io.EOF { // EOF = silent exit
					p.receiveQ <- protocolMessage{
						Status: DISCONNECT,
						Data:   []byte(err.Error()),
					}
					p.receiveThreadIsRunning = false
					return
				}

				p.receiveQ <- protocolMessage{
					Status: DISCONNECT,
					Data:   []byte(err.Error()),
				}
				p.receiveThreadIsRunning = false
				return
			}

			for _, ascii := range tcpReceiveBuffer[:n] {
				messageBuffer, action, err := fsm.Push(ascii)
				if err != nil {
					p.receiveQ <- protocolMessage{
						Status: ERROR,
						Data:   []byte(err.Error()),
					}
					p.receiveThreadIsRunning = false
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

					p.receiveQ <- protocolMsg
					p.receiveThreadIsRunning = false
					return
				case LineReceived:
					// append Data
					lastMessage = messageBuffer
					fileBuffer = append(fileBuffer, lastMessage)
					fsm.ResetBuffer()
				case utilities.Finish:
					// send fileData
					fullMsg := make([]byte, 0)
					for i, messageLine := range fileBuffer {
						// skip first element because this is only RecordType + unitNo not needed in instrumentAPI
						if i == 0 || len(messageLine) == 4 {
							continue
						}
						fullMsg = append(fullMsg, messageLine...)
						fullMsg = append(fullMsg, p.settings.lineBreak)
					}

					p.receiveQ <- protocolMessage{
						Status: DATA,
						Data:   fullMsg,
					}
					fsm.ResetBuffer()
					fsm.Init()
				case JustAck:
					conn.Write([]byte{utilities.ACK})
				default:
					protocolMsg := protocolMessage{
						Status: ERROR,
						Data:   []byte("Invalid action code "),
					}

					p.receiveQ <- protocolMsg
					p.receiveThreadIsRunning = false
					fmt.Println("Disconnect due to unexpected, unkown and unlikley error")
					return
				}
			}
		}
	}()
}

func (p *beckmanSpecialProtocol) Receive(conn net.Conn) ([]byte, error) {
	p.ensureReceiveThreadRunning(conn)

	message := <-p.receiveQ

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

func (p *beckmanSpecialProtocol) Send(conn net.Conn, data [][]byte) (int, error) {
	msgBuff := make([]byte, 0)

	// Maybe need to wait until the answer of the instrument
	for _, buff := range data {
		msgBuff = append(msgBuff, p.settings.startByte)
		msgBuff = append(msgBuff, buff...)
		msgBuff = append(msgBuff, p.settings.endByte)
	}

	return conn.Write(msgBuff)
}

func (p *beckmanSpecialProtocol) NewInstance() Implementation {
	return &beckmanSpecialProtocol{
		settings: p.settings,
		receiveQ: make(chan protocolMessage),
	}
}
