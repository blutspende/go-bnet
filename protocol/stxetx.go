package protocol

import (
	"io"
	"net"
)

type STXETXProtocolSettings struct {
}

type stxetx struct {
	settings               *STXETXProtocolSettings
	receiveQ               chan []byte
	receiveThreadIsRunning bool
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
		receiveQ:               make(chan []byte, 1024),
		receiveThreadIsRunning: false,
	}
}

func (proto *stxetx) Receive(conn net.Conn) ([]byte, error) {

	proto.ensureReceiveThreadRunning(conn)

	return <-proto.receiveQ, nil
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
				} else if opErr, ok := err.(*net.OpError); ok && opErr.Op == "read" && len(receivedMsg)+n == 0 {
					proto.receiveThreadIsRunning = false
					return
				} else if err == io.EOF { // EOF = silent exit
					proto.receiveThreadIsRunning = false
					return
				}
				proto.receiveThreadIsRunning = false
				return
			}

			for _, x := range tcpReceiveBuffer[:n] {
				if x == STX {
					receivedMsg = []byte{} // start of text obsoletes all prior
					continue
				}
				if x == ETX {
					proto.receiveQ <- receivedMsg
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

func (proto *stxetx) Send(conn net.Conn, data []byte) (int, error) {
	sendbytes := make([]byte, len(data)+2)
	sendbytes[0] = STX
	for i := 0; i < len(data); i++ {
		sendbytes[i+1] = data[i]
	}
	sendbytes[len(data)+1] = ETX
	return conn.Write(sendbytes)
}
