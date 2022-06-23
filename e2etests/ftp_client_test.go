package e2etests

import (
	intNet "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/net"
	"github.com/stretchr/testify/assert"
	"log"
	"net"
	"os"
	"testing"
)

func startFTPMockServer() {
	ln, err := net.Listen(intNet.TCPProtocol, ":4001")
	if err != nil {
		os.Exit(1)
	}

	var connections []net.Conn
	defer func() {
		for _, conn := range connections {
			conn.Close()
		}
	}()

	defer ln.Close()
	for {
		conn, e := ln.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				log.Printf("accept temp err: %v", ne)
				continue
			}

			log.Printf("accept err: %v", e)
			return
		}

		go func(conn net.Conn) {
			buf := make([]byte, 100)
			_, err := conn.Read(buf)
			if err != nil {
				panic("Read error")
			}
			receiveQ <- buf

			_, err = conn.Write([]byte("Hallo hier ist Server!"))
			if err != nil {
				panic("Write error")
			}
		}(conn)
		connections = append(connections, conn)
		if len(connections)%100 == 0 {
			log.Printf("total number of connections: %v", len(connections))
		}
	}
}

func Test_Simple_FTP_Client(t *testing.T) {
// 	go startFTPMockServer()
	tcpClient := intNet.CreateNewTCPClient("127.0.0.1", 4001, intNet.RawProtocol, intNet.NoProxy)

	tcpClient.Send([]byte("Hallo hier ist client!"))
	assert.Equal(t, "Hallo hier ist client!", string(<-receiveQ))
	receivedMsg, err := tcpClient.Receive()
	assert.Nil(t, err)
	assert.Equal(t, "Hallo hier ist Server!", receivedMsg)
	tcpClient.Stop()

}
