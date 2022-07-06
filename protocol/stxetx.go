package protocol

import (
	"errors"
	"io"
	"net"
	"time"
)

type STXETXProtocolSettings struct {
	flushTimeout_ms int
	readTimeout_ms  int
	maxBufferSize   int
}

type stxetx struct {
	settings *STXETXProtocolSettings
	sendQ    chan []byte
}

func DefaultSTXETXProtocolSettings() *STXETXProtocolSettings {
	var settings STXETXProtocolSettings
	settings.maxBufferSize = 4096
	settings.flushTimeout_ms = -1
	settings.readTimeout_ms = 50
	return &settings
}

func STXETX(settings *STXETXProtocolSettings) Implementation {
	return &stxetx{
		settings: settings,
		sendQ:    make(chan []byte, 1024),
	}
}

func (proto *stxetx) Receive(conn net.Conn) ([]byte, error) {

	tcpReceiveBuffer := make([]byte, 4096)
	receivedMsg := make([]byte, proto.settings.maxBufferSize)

	for {

		if proto.settings.readTimeout_ms > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(proto.settings.readTimeout_ms))); err != nil {
				return []byte{}, err
			}
		}

		n, err := conn.Read(tcpReceiveBuffer)

		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				//millisceondsSinceLastRead = millisceondsSinceLastRead + proto.settings.readTimeout_ms
				continue // on timeout....
			} else if opErr, ok := err.(*net.OpError); ok && opErr.Op == "read" && len(receivedMsg)+n == 0 {
				receivedMsg = append(receivedMsg, tcpReceiveBuffer[:n]...)
				break // eof when no data was read at all = is not an error as such, rather an unwanted disconnect
			} else if err == io.EOF {
				receivedMsg = append(receivedMsg, tcpReceiveBuffer[:n]...)
				return receivedMsg, nil
			}
			if _, ok := err.(*net.OpError); ok {
				return []byte{}, errors.New("connnection closed by peer")
			}

			return []byte{}, err
		}

		if n == 0 {
			continue
		}

		// TODO: this ipmlementation potentially deletes everything after ETX if its transmitted too close
		for _, x := range tcpReceiveBuffer[:n] {
			if x == STX {
				receivedMsg = []byte{} // start of text obsoletes all prior
				continue
			}
			if x == ETX {
				return receivedMsg, nil
			}
			receivedMsg = append(receivedMsg, x)
		}
	}

	return receivedMsg, nil
}

func (proto *stxetx) Interrupt() {
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
