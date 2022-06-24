package e2etests

import (
	"io"
	"os"
	"testing"
	"time"

	go_bloodlab_net "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net"
	intNet "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/net"

	"net"

	"github.com/stretchr/testify/assert"
)

var signalReady chan bool = make(chan bool)

type testSimpleTCPServerReceive struct {
}

var receiveQ = make(chan []byte, 500)
var lastConnected = ""

func (s *testSimpleTCPServerReceive) ClientConnected(source string, bloodLabSession go_bloodlab_net.Session) {
	lastConnected = source
	signalReady <- true
	bloodLabSession.WaitTermination()
}

func (s *testSimpleTCPServerReceive) DataReceived(source string, fileData []byte, receiveTimestamp time.Time) {
	lastConnected = source
	receiveQ <- fileData
}

func Test_Simple_TCP_Server_Receive(t *testing.T) {
	tcpServer := intNet.CreateNewTCPServerInstance(4001, intNet.PROTOCOL_RAW, intNet.NoProxy, 100)
	var handlerTcp testSimpleTCPServerReceive
	go tcpServer.Run(&handlerTcp)

	clientConn, err := net.Dial(intNet.TCPProtocol, "127.0.0.1:4001")
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

func (s *testSimpleTCPServerSend) ClientConnected(source string, conn go_bloodlab_net.Session) {
	lastConnected = source
	signalReady <- true
	for conn.IsAlive() {
		data := <-receiveQ
		_, err := conn.Send(data)
		if err != nil {
			return
		}
	}
}

func (s *testSimpleTCPServerSend) DataReceived(source string, fileData []byte, receiveTimestamp time.Time) {
	lastConnected = source
	receiveQ <- fileData
}

func Test_Simple_TCP_Server_Send(t *testing.T) {
	tcpServer := intNet.CreateNewTCPServerInstance(4001, intNet.PROTOCOL_RAW, intNet.NoProxy, 100)
	var handlerTcp testSimpleTCPServerSend
	go tcpServer.Run(&handlerTcp)

	conn, err := net.Dial(intNet.TCPProtocol, "127.0.0.1:4001")
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
	tcpServer := intNet.CreateNewTCPServerInstance(4001, intNet.PROTOCOL_RAW, intNet.NoProxy, 0)
	var handlerTcp testSimpleTCPServerSend
	go tcpServer.Run(&handlerTcp)

	conn, err := net.Dial(intNet.TCPProtocol, "127.0.0.1:4001")
	assert.Nil(t, err)
	assert.NotNil(t, conn)

	err = conn.SetReadDeadline(time.Now().Add(time.Second * 10))
	assert.Nil(t, err)

	buf := make([]byte, 2)
	_, err = conn.Read(buf)
	assert.Equal(t, io.EOF, err)
}
