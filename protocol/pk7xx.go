package protocol

import (
	"fmt"
	"io"
	"net"

	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
)

type pk7xxProtocol struct {
	//settings               *AU6XXProtocolSettings
	receiveThreadIsRunning bool
	receiveQ               chan protocolMessage
	state                  processState
	fsm                    []utilities.Rule
}

func PK7xxProtocol() Implementation {

	numbers := []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}

	normalCharacters := []byte{}
	for i := 0x21; i < 0xF7; i++ { // Character specification according to manual
		normalCharacters = append(normalCharacters, byte(i))
	}

	anySymbol := []byte{}
	for i := 0x0; i < 0xFF; i++ {
		anySymbol = append(anySymbol, byte(i))
	}

	fsm := []utilities.Rule{
		// Always starting a Message with STX SBxx ETX
		{FromState: 0, Symbols: []byte{utilities.STX}, ToState: 10, Scan: false},

		// Transmission Control
		{FromState: 10, Symbols: []byte{'S'}, ToState: 11, Scan: true},
		{FromState: 11, Symbols: []byte{'B'}, ToState: 200, Scan: true, ActionCode: "SB-Record"},

		// Datablock start
		{FromState: 10, Symbols: []byte{'D'}, ToState: 21, Scan: true},
		{FromState: 21, Symbols: []byte{'B'}, ToState: 1000, Scan: true, ActionCode: "DB-Record"},

		// Reagent information
		{FromState: 10, Symbols: []byte{'Q'}, ToState: 31, Scan: true},
		{FromState: 31, Symbols: []byte{'R'}, ToState: 200, Scan: true, ActionCode: "QR-Record"},

		// Diluent information
		{FromState: 10, Symbols: []byte{'Q'}, ToState: 21, Scan: true},
		{FromState: 41, Symbols: []byte{'D'}, ToState: 200, Scan: true, ActionCode: "QD-Record"},

		// Machine info
		{FromState: 10, Symbols: []byte{'M'}, ToState: 21, Scan: true},
		{FromState: 41, Symbols: []byte{' '}, ToState: 200, Scan: true, ActionCode: "M-Record"},

		// Device number All blocks
		{FromState: 200, Symbols: numbers, ToState: 201, Scan: true, ActionCode: "BCC"},

		{FromState: 201, Symbols: []byte{utilities.ETX}, ToState: 202, Scan: false},
		{FromState: 202, Symbols: anySymbol, ToState: 210, Scan: true},

		/*
			// Device number for DB-Block
			{FromState: 1000, Symbols: numbers, ToState: 1001, Scan: true},
			{FromState: 1001, Symbols: numbers, ToState: 1010, Scan: true},


			{FromState: 30, Symbols: numbers, ToState: 30, Scan: true},
			{FromState: 30, Symbols: []byte{utilities.ETX}, ToState: 40, Scan: false, ActionCode: "StartMessage"},
			{FromState: 40, Symbols: anySymbol, ToState: 50, Scan: true, ActionCode: "CheckSum"},

			// Normal Characters
			{FromState: 50, Symbols: []byte{utilities.STX}, ToState: 60, Scan: false},
			{FromState: 60, Symbols: []byte{'D'}, ToState: 61, Scan: true},
			{FromState: 61, Symbols: []byte{'E'}, ToState: 100, Scan: true},
			{FromState: 60, Symbols: normalCharacters, ToState: 60, Scan: true},
			{FromState: 60, Symbols: []byte{utilities.ETX}, ToState: 70, Scan: false, ActionCode: "Data"},
			{FromState: 70, Symbols: anySymbol, ToState: 50, Scan: true},

			// Leaving the show with DE ETX checksum
			{FromState: 100, Symbols: []byte{utilities.ETX}, ToState: 101, Scan: false},
			{FromState: 101, Symbols: anySymbol, ToState: 0, Scan: true}, */
	}

	return &pk7xxProtocol{
		fsm: fsm,
	}
}

func (p *pk7xxProtocol) Interrupt() {}

func (p *pk7xxProtocol) ensureReceiveThreadRunning(conn net.Conn) {

	if p.receiveThreadIsRunning {
		return
	}

	go func() {
		p.receiveThreadIsRunning = true
		p.state.State = 0

		//lastMessage := make([]byte, 0)
		tcpReceiveBuffer := make([]byte, 4096)
		//fileBuffer := make([][]byte, 0)

		fsm := utilities.CreateFSM(p.fsm)
		for p.receiveThreadIsRunning {
			n, err := conn.Read(tcpReceiveBuffer)
			// enabled FSM
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					fmt.Printf("read timeout reached. reset fsm")
					tcpReceiveBuffer = make([]byte, 4096)
					fsm.ResetBuffer()
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
				case "SB-Record":
					fmt.Printf("Messagebuffer %s\n", messageBuffer)
				default:
					protocolMsg := protocolMessage{
						Status: ERROR,
						Data:   []byte("Invalid action code "),
					}

					p.receiveQ <- protocolMsg
					p.receiveThreadIsRunning = false
					fmt.Println("Disconnect due to unexpected, unknown and unlikely error")
					return
				}
			}
		}
	}()
}

// Receive - asynch.
func (p *pk7xxProtocol) Receive(conn net.Conn) ([]byte, error) {
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

// Send - asynch
func (p *pk7xxProtocol) Send(conn net.Conn, data [][]byte) (int, error) {
	// Were not sending to the instrument
	return 0, fmt.Errorf("p7xxProtcol.Send is not implemented")
}

// Create a new Instance of this class duplicating all settings
func (p *pk7xxProtocol) NewInstance() Implementation {
	return &pk7xxProtocol{
		state: p.state,
	}
}
