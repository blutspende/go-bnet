package e2etests

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	intNet "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/net"
	"github.com/stretchr/testify/assert"
)

var TCPMockServerSendQ chan []byte = make(chan []byte)
var TCPMockServerReceiveQ chan []byte = make(chan []byte)

/* Run a Server for one connection,
reading from socket, writing to channel
reading from channel, writing to socket
Server stops when client disconnects
*/
var listener net.Listener

func runTCPMockServer() {
	if listener != nil { // In case previous session got stuck remove it
		listener.Close()
	}

	go func() {

		listener, err := net.Listen(intNet.TCPProtocol, ":4001")
		if err != nil {
			panic(err)
		}
		defer listener.Close()

		conn, e := listener.Accept()
		if e != nil {
			panic(err)
		}

		go func(conn net.Conn) {
			buf := make([]byte, 100)
			for {
				conn.SetDeadline(time.Now().Add(time.Millisecond * 200))
				_, err := conn.Read(buf)
				if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() { // timeout dont care
				} else if err == io.EOF {
					return
				} else {
					if err != nil {
						panic(err)
					} else {
						fmt.Println("Inqueue")
						TCPMockServerReceiveQ <- buf
					}
				}

				select {
				case x, ok := <-TCPMockServerSendQ:
					if ok {
						conn.SetDeadline(time.Now().Add(time.Millisecond * 200))
						fmt.Println("Will send now")
						if _, err = conn.Write(x); err != nil {
							panic(err)
						}
					}
				default:
				}
			}
		}(conn)
	}()
}

/*
	Connect to TCP-Server, Read Data and transmit data
*/
func TestClientConnectAndSend(t *testing.T) {
	runTCPMockServer()
	tcpClient := intNet.CreateNewTCPClient("127.0.0.1", 4001, intNet.PROTOCOL_RAW, intNet.NoProxy)

	const TESTSTRINGSEND = "Testdata"
	n, err := tcpClient.Send([]byte(TESTSTRINGSEND))
	if err != nil {
		panic(err)
	}
	sendMsg := <-TCPMockServerReceiveQ
	assert.Equal(t, TESTSTRINGSEND, string(sendMsg[:n]))

	// Testing Reading
	const TESTSTRINGRECEIVE = "This data is definateley beeing transmitted"
	TCPMockServerSendQ <- []byte(TESTSTRINGRECEIVE)
	receivedMsg, err := tcpClient.Receive()
	assert.Nil(t, err)
	assert.Equal(t, TESTSTRINGRECEIVE, string(receivedMsg))

	// Disconnect & stop
	tcpClient.Stop()
}

func TestClientProtocolSTXETX(t *testing.T) {
	runTCPMockServer()
	tcpClient := intNet.CreateNewTCPClient("127.0.0.1", 4001, intNet.PROTOCOL_STXETX, intNet.NoProxy)

	TESTSTRING := "H|\\^&|||bloodlab-net|e2etest||||||||20220614163728\nL|1|N"
	TCPMockServerSendQ <- []byte([]byte("\u0002" + TESTSTRING + "\u0003"))
	receivedMsg, err := tcpClient.Receive()
	assert.Nil(t, err)
	assert.Equal(t, TESTSTRING, string(receivedMsg))

	tcpClient.Stop()
}

/*
func TestClientConnectAndSend(dataThatIsExpectedToBeSubmitted []byte) {
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
			_, err = conn.Write(dataThatIsExpectedToBeSubmitted)
			if err != nil {
				panic(err)
			}
		}(conn)
		connections = append(connections, conn)
		if len(connections)%100 == 0 {
			log.Printf("total number of connections: %v", len(connections))
		}
	}
}



func Test_Simple_TCP_Client_SendData(t *testing.T) {
	TESTSTRING := "H|\\^&|||bloodlab-net|e2etest||||||||20220614163728\nL|1|N"
	go startTCPMockWrappedSTXProtocolServer([]byte(""))

	tcpClient := intNet.CreateNewTCPClient("127.0.0.1", 4001, intNet.PROTOCOL_RAW, intNet.NoProxy)
	_, err := tcpClient.Send([]byte(TESTSTRING))
	assert.Nil(t, err)
	assert.Equal(t, TESTSTRING, <-TCPMockServerReceiveQ)
	tcpClient.Stop()
}
*/
