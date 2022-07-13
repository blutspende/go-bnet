package protocol

import (
	"fmt"
	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
	"io"
	"net"
)

type STXETXProtocolSettings struct {
}

type stxetx struct {
	settings               *STXETXProtocolSettings
	receiveQ               chan protocolMessage
	receiveThreadIsRunning bool
	connectionIsValid      bool
}

func DefaultSTXETXProtocolSettings() *STXETXProtocolSettings {
	var settings STXETXProtocolSettings
	return &settings
}

func STXETX(settings ...*STXETXProtocolSettings) Implementation {

	var thesettings *STXETXProtocolSettings
	if len(settings) >= 1 {
		thesettings = settings[0]
	} else {
		thesettings = DefaultSTXETXProtocolSettings()
	}

	return &stxetx{
		settings:               thesettings,
		receiveQ:               make(chan protocolMessage, 1024),
		receiveThreadIsRunning: false,
		connectionIsValid:      false,
	}
}

func (proto *stxetx) Receive(conn net.Conn) ([]byte, error) {

	proto.connectionIsValid = true
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
func (proto *stxetx) ensureReceiveThreadRunning(conn net.Conn) {

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

					proto.connectionIsValid = false // invalid connections can not be written to anymore

					messageEOF := protocolMessage{Status: EOF, Data: []byte{}}
					proto.receiveQ <- messageEOF
					proto.receiveThreadIsRunning = false
					return
				} else if err == io.EOF { // EOF = silent exit

					proto.connectionIsValid = false // invalid connections can not be written to anymore

					messageEOF := protocolMessage{Status: EOF}
					proto.receiveQ <- messageEOF

					proto.receiveThreadIsRunning = false
					return
				}

				proto.connectionIsValid = false // invalid connections can not be written to anymore

				messageERROR := protocolMessage{Status: ERROR, Data: []byte(err.Error())}
				proto.receiveQ <- messageERROR
				proto.receiveThreadIsRunning = false
				return
			}

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
	}()
}

func (proto *stxetx) Interrupt() {
	// not implemented (not required neither)
}

func (proto *stxetx) Send(conn net.Conn, data [][]byte) (int, error) {

	if proto.connectionIsValid {
		msgBuff := make([]byte, 0)
		for _, line := range data {
			msgBuff = append(msgBuff, line...)
		}

		sendBytes := make([]byte, len(msgBuff)+2)
		sendBytes[0] = utilities.STX
		for i := 0; i < len(msgBuff); i++ {
			sendBytes[i+1] = msgBuff[i]
		}
		sendBytes[len(msgBuff)+1] = utilities.ETX
		return conn.Write(sendBytes)
	}

	return 0, io.EOF
}
