package protocol

import (
	"fmt"
	"io"
	"net"
)

type Lis1A1ProtocolSettings struct {
}

type ProcessState struct {
	State        int
	Lastmessage  string
	LastChecksum string
	Messagelog   []string
}

type FSMConditionHandler func(state *ProcessState, rule FSM, scanbuffer []byte, filebuffer []byte, conn net.Conn) bool

// FSM struct for FSM
type FSM struct {
	FromState int
	Symbols   []byte
	ToState   int
	Handler   FSMConditionHandler
	Scan      bool
	Finish    bool
}

var lis1a1protocol = []FSM{
	{FromState: 0, Symbols: []byte{ENQ}, ToState: 1, Scan: false, Finish: false, Handler: justAck},
	{FromState: 1, Symbols: []byte{STX}, ToState: 2, Scan: false, Finish: false, Handler: nil},

	{FromState: 2, Symbols: []byte{ETB}, ToState: 5, Scan: false, Finish: false, Handler: nil},
	{FromState: 5, Symbols: []byte{STX}, ToState: 2, Scan: false, Finish: false, Handler: nil},

	{FromState: 2, Symbols: []byte{CR}, ToState: 3, Scan: false, Finish: false, Handler: nil},
	{FromState: 3, Symbols: []byte{ETX}, ToState: 7, Scan: false, Finish: false, Handler: nil},

	{FromState: 7, Symbols: []byte("01234567890ABCDEFabcdef"), ToState: 7, Scan: true, Finish: false, Handler: checksum},
	{FromState: 7, Symbols: []byte{CR}, ToState: 9, Scan: false, Finish: false, Handler: processMessage},
	{FromState: 9, Symbols: []byte{LF}, ToState: 10, Scan: false, Finish: false, Handler: nil},

	{FromState: 10, Symbols: []byte{STX}, ToState: 2, Scan: false, Finish: false, Handler: nil},
	{FromState: 10, Symbols: []byte{ETB}, ToState: 0, Scan: false, Finish: false, Handler: finishTransmission},
	{FromState: 10, Symbols: []byte{EOT}, ToState: 0, Scan: false, Finish: false, Handler: finishTransmission},
	// Any rule must  be last
	{FromState: 2, Symbols: []byte{0x18}, ToState: 2, Scan: true, Finish: false, Handler: astmMessage},
}

type lis1A1 struct {
	settings               *Lis1A1ProtocolSettings
	receiveQ               chan []byte
	receiveThreadIsRunning bool
	state                  ProcessState
}

func DefaultLis1A1ProtocolSettings() *Lis1A1ProtocolSettings {
	var settings Lis1A1ProtocolSettings
	return &settings
}

func Lis1A1Protocol(settings ...*Lis1A1ProtocolSettings) Implementation {

	var thesettings *Lis1A1ProtocolSettings
	if len(settings) >= 1 {
		thesettings = settings[0]
	} else {
		thesettings = DefaultLis1A1ProtocolSettings()
	}

	return &lis1A1{
		settings:               thesettings,
		receiveQ:               make(chan []byte, 1024),
		receiveThreadIsRunning: false,
	}
}

func (proto *lis1A1) Receive(conn net.Conn) ([]byte, error) {

	proto.ensureReceiveThreadRunning(conn)

	return <-proto.receiveQ, nil
}

func astmMessage(state *ProcessState, rule FSM, token []byte, filebuffer []byte, conn net.Conn) bool {
	//state.Lastmessage = token

	fmt.Println("ASTMMEssage:", string(token))
	filebuffer = append(filebuffer, token...)
	return true
}

func checksum(istate *ProcessState, rule FSM, token []byte, filebuffer []byte, conn net.Conn) bool {
	//state.LastChecksum = token
	return true
}

func justAck(state *ProcessState, rule FSM, token []byte, filebuffer []byte, conn net.Conn) bool {

	conn.Write([]byte{ACK})

	return true
}

func processMessage(state *ProcessState, rule FSM, token []byte, filebuffer []byte, conn net.Conn) bool {

	conn.Write([]byte{ACK})

	fmt.Println("RECOCNIG", string(token))
	/*
		if !h.validateChecksum(state.Lastmessage, state.LastChecksum) {
			h.instrumentMessageService.AddInstrumentMessage(*instrument.IPAddress, repositories.INSTRUMENTERROR,
				[]byte(fmt.Sprintf("Warning: Invalid checksm for Message, but will process anyhow. %s", token)), 0, instrument.ID)
		} else {
			state.Messagelog = append(state.Messagelog, state.Lastmessage)
		}*/

	return true
}

func finishTransmission(state *ProcessState, rule FSM, token []byte, filebuffer []byte, conn net.Conn) bool {
	fmt.Println("finished:", string(token))
	return true
}

// asynchronous receiveloop
func (proto *lis1A1) ensureReceiveThreadRunnfiring(conn net.Conn) {

	if proto.receiveThreadIsRunning {
		return
	}

	go func() {
		proto.receiveThreadIsRunning = true

		proto.state.State = 0 // initial state for FSM
		scanbuffer := []byte{}
		filebuffer := []byte{}

		tcpReceiveBuffer := make([]byte, 4096)

		for {
			fmt.Print("[", proto.state.State, "]")

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

				ruleapplied := false

				for _, trans := range lis1a1protocol {
					if trans.FromState == proto.state.State && containsSymbol(ascii, trans.Symbols) {
						ruleapplied = true
						//fmt.Println("[", trans.ToState, "]")
						if trans.Scan {
							scanbuffer = append(scanbuffer, ascii)
							// fmt.Println(fmt.Sprintf("scanbuffer: %s", scanbuffer))
						}
						// fmt.Println(fmt.Sprintf("scanbuffer: %s (FromState %d ToState %d)", scanbuffer, trans.FromState, trans.ToState))
						if trans.Handler != nil {
							trans.Handler(&proto.state, trans, []byte{ascii}, filebuffer, conn)
						}
						if trans.ToState != trans.FromState {
							scanbuffer = []byte{}
							proto.state.State = trans.ToState
						}
						// fmt.Println(fmt.Sprintf("scanbuffer: %s", scanbuffer))
						break
					}
				}

				if !ruleapplied {
					fmt.Println("**** INVALID Character in input-stream: ", string(ascii), "(", ascii, ")")
				}
			}
		}
	}()
}

func (proto *lis1A1) Interrupt() {
	// not implemented (not required neither)
}

func (proto *lis1A1) Send(conn net.Conn, data []byte) (int, error) {
	sendbytes := make([]byte, len(data)+2)
	sendbytes[0] = STX
	for i := 0; i < len(data); i++ {
		sendbytes[i+1] = data[i]
	}
	sendbytes[len(data)+1] = ETX
	return conn.Write(sendbytes)
}

func containsSymbol(symbol byte, symbols []byte) bool {
	for _, x := range symbols {
		if x == 0x18 { // 0x18 = ANY
			return true
		}
		if x == symbol {
			return true
		}
	}
	return false
}

// ComputeChecksum Helper to compute the ASTM-Checksum
func computeChecksum(record string) []byte {
	sum := int(0)

	for _, b := range []byte(record) {
		sum = sum + int(b)
	}
	sum += int(CR)
	sum += int(ETX)
	sum = sum % 256
	//fmt.Println("Checksum:", fmt.Sprintf("hex:%x dec:%d", sum, sum))
	return []byte(fmt.Sprintf("%02x", sum))
}
