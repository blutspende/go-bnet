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
	"bytes"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net"
	"time"

	"github.com/blutspende/go-bnet/protocol/utilities"
)

type MLLPProtocolSettings struct {
	startBytes         []byte
	endBytes           []byte
	lineBreakByte      byte
	readTimeoutSeconds int
}

type mllp struct {
	settings               *MLLPProtocolSettings
	receiveQ               chan protocolMessage
	receiveThreadIsRunning bool
}

func DefaultMLLPProtocolSettings() *MLLPProtocolSettings {
	return &MLLPProtocolSettings{
		startBytes:         []byte{utilities.VT},
		endBytes:           []byte{utilities.FS, utilities.CR},
		lineBreakByte:      utilities.CR,
		readTimeoutSeconds: 60,
	}
}

func (set *MLLPProtocolSettings) SetStartBytes(startBytes []byte) *MLLPProtocolSettings {
	set.startBytes = startBytes
	return set
}

func (set *MLLPProtocolSettings) SetEndBytes(endBytes []byte) *MLLPProtocolSettings {
	set.endBytes = endBytes
	return set
}

func (set *MLLPProtocolSettings) SetReadTimeoutSeconds(readTimeoutSeconds int) *MLLPProtocolSettings {
	set.readTimeoutSeconds = readTimeoutSeconds
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

func (proto *mllp) NewInstance() Implementation {
	return &mllp{
		settings:               proto.settings,
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
		remoteAddress := conn.RemoteAddr().String()
		for {
			if proto.settings.readTimeoutSeconds > 0 {
				_ = conn.SetReadDeadline(time.Now().Add(time.Second * time.Duration(proto.settings.readTimeoutSeconds)))
			}
			n, err := conn.Read(tcpReceiveBuffer)
			log.Debug().Str("remoteAddress", remoteAddress).Bytes("receivedBytes", tcpReceiveBuffer[:n]).Msg("mllp: bytes received")
			if err != nil {
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
					log.Debug().Str("remoteAddress", remoteAddress).Err(err).Msg("mllp: timeout - continue")
					continue // on timeout....
				} else if opErr, ok := err.(*net.OpError); ok && opErr.Op == "read" {
					log.Debug().Str("remoteAddress", remoteAddress).Err(err).Msg("mllp: operror - read")
					if len(receivedMsg)+n > 0 { // Process the remainder of the cache
						log.Debug().Str("remoteAddress", remoteAddress).Msg("mllp: operror - read - process remainder")
						for _, x := range tcpReceiveBuffer[:n] {
							if x == utilities.STX {
								log.Debug().Str("remoteAddress", remoteAddress).Msg("mllp: operror - read - process remainder - stx found")
								receivedMsg = []byte{} // start of text obsoletes all prior
								continue
							}
							if x == utilities.ETX {
								log.Debug().Str("remoteAddress", remoteAddress).Msg("mllp: operror - read - process remainder - etx found")
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

			log.Debug().Str("remoteAddress", remoteAddress).Msg("mllp: process received bytes")
			for i := 0; i < n; i++ {
				if i <= n-len(proto.settings.startBytes) && bytes.Equal(tcpReceiveBuffer[i:i+len(proto.settings.startBytes)], proto.settings.startBytes) {
					log.Debug().Str("remoteAddress", remoteAddress).Msg("mllp: startBytes found")
					receivedMsg = []byte{} // start of text obsoletes all prior
					i = i + len(proto.settings.startBytes) - 1
					continue
				}

				//log only for message start end frame bytes
				if n <= 3 {
					log.Debug().
						Str("remoteAddress", remoteAddress).
						Bytes("tcpBufferPart", tcpReceiveBuffer[i:n]).
						Int("i", i).
						Bytes("endBytes", proto.settings.endBytes).
						Msg("mllp: compare bytes with endBytes")
				}

				if bytes.Equal(tcpReceiveBuffer[i:n], proto.settings.endBytes) {
					log.Debug().Str("remoteAddress", remoteAddress).Msg("mllp: endBytes found")
					messageDATA := protocolMessage{Status: DATA, Data: receivedMsg}
					i += len(proto.settings.endBytes)
					proto.receiveQ <- messageDATA
					continue
				}
				receivedMsg = append(receivedMsg, tcpReceiveBuffer[i])
			}
		}
	}()
}

func (proto *mllp) Interrupt() {
	// not implemented (not required neither)
}

func (proto *mllp) Send(conn net.Conn, data [][]byte) (int, error) {

	msgBuff := make([]byte, 0)
	msgBuff = append(msgBuff, proto.settings.startBytes...)
	for _, line := range data {
		msgBuff = append(msgBuff, line...)
		msgBuff = append(msgBuff, proto.settings.lineBreakByte)
	}
	msgBuff = append(msgBuff, proto.settings.endBytes...)

	return conn.Write(msgBuff)
}
