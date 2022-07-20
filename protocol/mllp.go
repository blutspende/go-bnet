//
//Implementation of the minimal Low Level Protocol.
//
//In order to introduce message orientation to a stream-oriented TCP/IP protocol, a
//Minimal Low-Level Protocol (MLLP) was proposed. This subchapter contains a very
//brief overview of MLLP. HL7 messages are enclosed by special characters to form a
//block.
//
//The format is as follows:
//<SB>dddd<EB><CR>
//
//The characters used for begin and end of the message is configurable
//
//By default, the values are <VT> for <SB> and <FS> for <EB>

package protocol

import (
	"fmt"
	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
	"io"
	"net"
)

type MLLPProtocolSettings struct {
	startByte     byte
	endByte       byte
	lineBreakByte byte
}

type mllp struct {
	settings               *MLLPProtocolSettings
	receiveQ               chan protocolMessage
	receiveThreadIsRunning bool
}

func DefaultMLLPProtocolSettings() *MLLPProtocolSettings {
	return &MLLPProtocolSettings{
		startByte:     utilities.VT,
		endByte:       utilities.FS,
		lineBreakByte: utilities.CR,
	}
}

func (set *MLLPProtocolSettings) SetStartByte(startByte byte) *MLLPProtocolSettings {
	set.startByte = startByte
	return set
}

func (set *MLLPProtocolSettings) SetEndByte(endByte byte) *MLLPProtocolSettings {
	set.endByte = endByte
	return set
}

func MLLP(settings ...*MLLPProtocolSettings) Implementation {

	var thesettings *MLLPProtocolSettings
	if len(settings) >= 1 {
		thesettings = settings[0]
	} else {
		thesettings = DefaultMLLPProtocolSettings()
	}

	return &mllp{
		settings:               thesettings,
		receiveQ:               make(chan protocolMessage, 1024),
		receiveThreadIsRunning: false,
	}
}

func (proto *mllp) Receive(conn net.Conn) ([]byte, error) {

	proto.ensureReceiveThreadRunning(conn)

	// TODO Timeout
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

// asynchronous receiveloop
func (proto *mllp) ensureReceiveThreadRunning(conn net.Conn) {

	if proto.receiveThreadIsRunning {
		return
	}

	go func() {
		proto.receiveThreadIsRunning = true

		tcpReceiveBuffer := make([]byte, 4096)
		receivedMsg := make([]byte, 0)

		for {

			n, err := conn.Read(tcpReceiveBuffer)

			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					continue // on timeout....
				} else if opErr, ok := err.(*net.OpError); ok && opErr.Op == "read" {

					if len(receivedMsg)+n > 0 { // Process the remainder of the cache

						for _, x := range tcpReceiveBuffer[:n] {
							if x == utilities.STX {
								receivedMsg = []byte{} // start of text obsoletes all prior
								continue
							}
							if x == utilities.ETX {
								messageDATA := protocolMessage{Status: DATA, Data: receivedMsg}
								proto.receiveQ <- messageDATA
								continue
							}
							receivedMsg = append(receivedMsg, x)
						}
					}

					messageEOF := protocolMessage{Status: EOF}
					proto.receiveQ <- messageEOF
					proto.receiveThreadIsRunning = false
					return
				} else if err == io.EOF { // EOF = silent exit

					messageEOF := protocolMessage{Status: EOF}
					proto.receiveQ <- messageEOF
					proto.receiveThreadIsRunning = false
					return
				}

				messageERROR := protocolMessage{Status: ERROR, Data: []byte(err.Error())}
				proto.receiveQ <- messageERROR
				proto.receiveThreadIsRunning = false
				return
			}

			for _, x := range tcpReceiveBuffer[:n] {
				if x == proto.settings.startByte {
					receivedMsg = []byte{} // start of text obsoletes all prior
					continue
				}
				if x == proto.settings.endByte {
					messageDATA := protocolMessage{Status: DATA, Data: receivedMsg}
					proto.receiveQ <- messageDATA
					continue
				}
				receivedMsg = append(receivedMsg, x)
			}
		}
	}()
}

func (proto *mllp) Interrupt() {
	// not implemented (not required neither)
}

func (proto *mllp) Send(conn net.Conn, data [][]byte) (int, error) {

	msgBuff := make([]byte, 0)
	msgBuff = append(msgBuff, proto.settings.startByte)
	for _, line := range data {
		msgBuff = append(msgBuff, line...)
		msgBuff = append(msgBuff, proto.settings.lineBreakByte)
	}
	msgBuff = append(msgBuff, proto.settings.endByte)

	return conn.Write(msgBuff)
}
