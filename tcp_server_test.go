package main

import (
	"io"
	"log"
	"os"
	"testing"
	"time"

	"net"

	"github.com/stretchr/testify/assert"
)

var signalReady chan bool = make(chan bool)

type testSimpleTCPServerReceive struct {
}

var receiveQ = make(chan []byte, 500)
var lastConnected = ""

func (s *testSimpleTCPServerReceive) Connected(session Session) {
	lastConnected, _ = session.RemoteAddress()
	signalReady <- true
	session.WaitTermination()
}

func (s *testSimpleTCPServerReceive) Disconnected(session Session) {
	log.Panic("Not implemented")
}

func (s *testSimpleTCPServerReceive) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
	lastConnected, _ = session.RemoteAddress()
	receiveQ <- fileData
}

func (s *testSimpleTCPServerReceive) Error(session Session, errorType ErrorType, err error) {

}

func Test_Simple_TCP_Server_Receive(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4001, PROTOCOL_RAW, NoLoadbalancer, 100)
	var handlerTcp testSimpleTCPServerReceive
	go tcpServer.Run(&handlerTcp)

	clientConn, err := net.Dial("tcp", "127.0.0.1:4001")
	if err != nil {
		t.Fail()
		os.Exit(1)
	}

	select {
	case isReady := <-signalReady:
		if isReady {
			assert.Equal(t, "127.0.0.1", lastConnected)
		}
	case <-time.After(2 * time.Second):
		{
			t.Fatalf("Can not start tcp server")
		}
	}

	const TESTSTRING = "Hello its me"
	_, err = clientConn.Write([]byte(TESTSTRING))
	assert.Nil(t, err)
	clientConn.Close()

	select {
	case receivedMsg := <-receiveQ:
		if receivedMsg != nil {
			assert.Equal(t, TESTSTRING, string(receivedMsg))
		}
	case <-time.After(2 * time.Second):
		{
			t.Fatalf("Can not receive messages from the client")
		}
	}

	tcpServer.Stop()

}

type testSimpleTCPServerSend struct {
}

func (s *testSimpleTCPServerSend) Connected(session Session) {
	lastConnected, _ = session.RemoteAddress()
	signalReady <- true
	for session.IsAlive() {
		data := <-receiveQ
		_, err := session.Send(data)
		if err != nil {
			return
		}
	}
}

func (s *testSimpleTCPServerSend) Disconnected(session Session) {
}

func (s *testSimpleTCPServerSend) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
	lastConnected, _ = session.RemoteAddress()
	receiveQ <- fileData
}

func (s *testSimpleTCPServerSend) Error(session Session, errtype ErrorType, err error) {

}

func Test_Simple_TCP_Server_Send(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4001, PROTOCOL_RAW, NoLoadbalancer, 100)

	var handlerTcp testSimpleTCPServerSend
	go tcpServer.Run(&handlerTcp)

	conn, err := net.Dial("tcp", "127.0.0.1:4001")
	if err != nil {
		t.Fail()
		os.Exit(1)
	}

	receiveQ <- []byte("ACK")

	select {
	case isReady := <-signalReady:
		if isReady {
			assert.Equal(t, "127.0.0.1", lastConnected)
		}
	case <-time.After(50 * time.Second):
		{
			t.Fatalf("Can not start tcp server")
		}
	}

	buff := make([]byte, 1500)
	n, err := conn.Read(buff)
	assert.Nil(t, err)
	assert.Equal(t, "ACK", string(buff[:n]))
	conn.Close()
	tcpServer.Stop()
}

func Test_Simple_TCP_Server_Max_Connections(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4001, PROTOCOL_RAW, NoLoadbalancer, 0)
	var handlerTcp testSimpleTCPServerSend
	go tcpServer.Run(&handlerTcp)

	conn, err := net.Dial("tcp", "127.0.0.1:4001")
	assert.Nil(t, err)
	assert.NotNil(t, conn)

	err = conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	assert.Nil(t, err)

	buf := make([]byte, 2)
	_, err = conn.Read(buf)
	assert.Equal(t, io.EOF, err)
}
