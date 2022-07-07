package protocol

import (
	"io"
	"net"
	"sync"
	"time"
)

type RawProtocolSettings struct {
	flushTimeout_ms int
	readTimeout_ms  int
	maxBufferSize   int
}

type rawprotocol struct {
	settings               *RawProtocolSettings
	blockReceivingMainloop *sync.Mutex
}

func DefaultRawProtocolSettings() *RawProtocolSettings {
	var rp RawProtocolSettings
	rp.readTimeout_ms = 100
	rp.flushTimeout_ms = 200
	rp.maxBufferSize = 512 * 1024 * 1024 // half a meg
	return &rp
}

/* Rawreceiver - no changes to incoming data
   maxBuffersize - bytes to store (prevent buffer overflow with this)
   readTimeout_ms required to enable the flush timeout
   flushTimeout_ms >1 for a timeout when the receive buffer is beeing forwared, 0 to disable
*/
func Raw(settings ...*RawProtocolSettings) Implementation {

	var thesettings *RawProtocolSettings
	if len(settings) >= 1 {
		thesettings = settings[0]
	} else {
		thesettings = DefaultRawProtocolSettings()
	}

	return &rawprotocol{
		settings:               thesettings,
		blockReceivingMainloop: &sync.Mutex{},
	}
}

func (proto *rawprotocol) Receive(conn net.Conn) ([]byte, error) {

	tcpReceiveBuffer := make([]byte, 4096)
	receivedMsg := make([]byte, 0)

	millisceondsSinceLastRead := 0

	for {

		// flush timeout for protocol RAW.
		if proto.settings.flushTimeout_ms > 0 && // disabled timout ?
			millisceondsSinceLastRead > proto.settings.flushTimeout_ms &&
			len(receivedMsg) > 0 { // buffer full, time up = flush it
			break
		}

		if proto.settings.readTimeout_ms > 0 {
			if err := conn.SetReadDeadline(time.Now().Add(time.Millisecond * time.Duration(proto.settings.readTimeout_ms))); err != nil {
				return []byte{}, err
			}
		}

		n, err := conn.Read(tcpReceiveBuffer)

		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				millisceondsSinceLastRead = millisceondsSinceLastRead + proto.settings.readTimeout_ms
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
		millisceondsSinceLastRead = 0

		receivedMsg = append(receivedMsg, tcpReceiveBuffer[:n]...)

	}

	return receivedMsg, nil
}

func (proto *rawprotocol) Interrupt() {
	// Not necessary for raw
}

func (proto *rawprotocol) Send(conn net.Conn, data []byte) (int, error) {
	//proto.blockReceivingMainloop.Lock()
	n, err := conn.Write(data)
	//proto.blockReceivingMainloop.Unlock()
	return n, err
}
