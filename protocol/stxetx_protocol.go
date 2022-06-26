package protocol

import "net"

func ReceiveWrappedStxProtocol(conn net.Conn) ([]byte, error) {
	buff := make([]byte, 100)
	receivedMsg := make([]byte, 0)

	for {

		n, err := conn.Read(buff)
		if err != nil {
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
				continue
			}
			return nil, err
		}
		if n == 0 {
			return receivedMsg, err
		}

		for _, x := range buff[:n] {
			if x == STX {
				continue
			}
			if x == ETX {
				return receivedMsg, err
			}
			receivedMsg = append(receivedMsg, x)
		}
	}
}

func SendWrappedStxProtocol(conn net.Conn, data []byte) (int, error) {

	sendbytes := make([]byte, len(data)+2)
	sendbytes[0] = STX
	for i := 0; i < len(data); i++ {
		sendbytes[i+1] = data[i]
	}
	sendbytes[len(data)+1] = ETX

	n, err := conn.Write(sendbytes)

	return n + 2, err
}
