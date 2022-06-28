package protocol

import (
	"fmt"
	"io"
	"net"
)

type STXETXProtocolSettings struct {
	flushTimeout_ms int
	readTimeout_ms  int
	maxBufferSize   int
}

type stxetx struct {
	settings *STXETXProtocolSettings
}

func DefaultSTXETXProtocolSettings() *STXETXProtocolSettings {
	var settings STXETXProtocolSettings
	settings.maxBufferSize = 512 * 1024 * 1024
	return &settings
}

func STXETX(settings *STXETXProtocolSettings) Implementation {
	return &stxetx{
		settings: settings,
	}
}

func (proto *stxetx) Receive(conn net.Conn) ([]byte, error) {

	tcpReceiveBuffer := make([]byte, 4096)
	receivedMsg := make([]byte, proto.settings.maxBufferSize)

	for {

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
				break
			}
			return []byte{}, err
		}

		if n == 0 {
			continue
		}

		// TODO: this ipmlementation potentially deletes everything after ETX if its transmitted too close
		for _, x := range tcpReceiveBuffer[:n] {
			if x == STX {
				continue
			}
			if x == ETX {
				fmt.Println("got ETX")
				break
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

	n, err := conn.Write(sendbytes)

	return n + 2, err
}
