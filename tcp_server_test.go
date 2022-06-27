package bloodlabnet

import (
	"log"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testRawDataProtocolSession struct {
	receiveQ                    chan []byte
	lastConnected               string
	signalReady                 chan bool
	didReceiveDisconnectMessage bool
}

func (s *testRawDataProtocolSession) Connected(session Session) {
	s.lastConnected, _ = session.RemoteAddress()
	s.signalReady <- true
}

func (s *testRawDataProtocolSession) Disconnected(session Session) {
	s.didReceiveDisconnectMessage = true
}

func (s *testRawDataProtocolSession) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
	s.lastConnected, _ = session.RemoteAddress()
	s.receiveQ <- fileData
	session.Send([]byte("An adequate response"))
}

func (s *testRawDataProtocolSession) Error(session Session, errorType ErrorType, err error) {
	log.Fatal(err)
}

func TestRawDataProtocol(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4001, PROTOCOL_RAW, PROTOCOL_RAW, NoLoadBalancer, 100, DefaultTCPServerTimings)

	var handler testRawDataProtocolSession
	handler.receiveQ = make(chan []byte, 500)
	handler.signalReady = make(chan bool)

	isServerHalted := false

	// run the mainloop, observe that it closes later
	go func() {
		tcpServer.Run(&handler)
		isServerHalted = true
	}()

	// connect and expect connection handler to signal
	clientConn, err := net.Dial("tcp", "127.0.0.1:4001")
	if err != nil {
		t.Fail()
		os.Exit(1)
	}

	select {
	case isReady := <-handler.signalReady:
		if isReady {
			assert.Equal(t, "127.0.0.1", handler.lastConnected)
		}
	case <-time.After(2 * time.Second):
		{
			t.Fatalf("Can not start tcp server")
		}
	}

	// send data to server, expect it to receive
	const TESTSTRING = "Hello its me"
	_, err = clientConn.Write([]byte(TESTSTRING))
	assert.Nil(t, err)

	select {
	case receivedMsg := <-handler.receiveQ:
		assert.NotNil(t, receivedMsg, "Received a valid response")
		assert.Equal(t, TESTSTRING, string(receivedMsg))
	case <-time.After(2 * time.Second):
		t.Fatalf("Can not receive messages from the client")
	}

	// expect "an adeqate response", that is the string the server sends back
	var buffer []byte = make([]byte, 100)

	_ = clientConn.SetDeadline(time.Now().Add(time.Second * 2))
	n, err := clientConn.Read(buffer)
	assert.Nil(t, err, "Reading from client")
	assert.Equal(t, "An adequate response", string(buffer[:n]))

	//	clientConn.Close()

	tcpServer.Stop()

	time.Sleep(time.Second * 1)

	assert.True(t, handler.didReceiveDisconnectMessage, "Disconnect message was send")
	assert.True(t, isServerHalted, "Server has been stopped")
}

//------------------------------------------------------
// Test that the server declines too many connections
//------------------------------------------------------
type testTCPServerMaxConnections struct {
	maxConnectionErrorDidOccur bool
}

func (s *testTCPServerMaxConnections) Connected(session Session) {
}

func (s *testTCPServerMaxConnections) Disconnected(session Session) {
}

func (s *testTCPServerMaxConnections) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
}

func (s *testTCPServerMaxConnections) Error(session Session, errorType ErrorType, err error) {
	if errorType == ErrorMaxConnections {
		s.maxConnectionErrorDidOccur = true
	}
}

func TestTCPServerMaxConnections(t *testing.T) {

	tcpServer := CreateNewTCPServerInstance(4002,
		PROTOCOL_RAW,
		PROTOCOL_RAW,
		NoLoadBalancer,
		2,
		DefaultTCPServerTimings)

	var handlerTcp testTCPServerMaxConnections
	handlerTcp.maxConnectionErrorDidOccur = false

	go tcpServer.Run(&handlerTcp)

	conn1, err1 := net.Dial("tcp", "127.0.0.1:4002")
	assert.Nil(t, err1)
	assert.NotNil(t, conn1)

	conn2, err2 := net.Dial("tcp", "127.0.0.1:4002")
	assert.Nil(t, err2)
	assert.NotNil(t, conn2)

	conn3, err3 := net.Dial("tcp", "127.0.0.1:4002")
	assert.Nil(t, err3)
	assert.NotNil(t, conn3)

	time.Sleep(time.Second * 1) // sessions start async, therefor a short waitign is required

	assert.True(t, handlerTcp.maxConnectionErrorDidOccur, "Expected error: MaxConnections did occur")
}

//------------------------------------------------------
// Server identifies the remote-Address
//------------------------------------------------------
type testTCPServerRemoteAddress struct {
	lastConnectionSource string
	wasConnectedCalled   *sync.WaitGroup
}

func (s *testTCPServerRemoteAddress) Connected(session Session) {
	s.lastConnectionSource, _ = session.RemoteAddress()
	s.wasConnectedCalled.Done()
}

func (s *testTCPServerRemoteAddress) Disconnected(session Session) {
}

func (s *testTCPServerRemoteAddress) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
}

func (s *testTCPServerRemoteAddress) Error(session Session, errorType ErrorType, err error) {
	log.Println(err)
}

func TestTCPServerIdentifyRemoteAddress(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4005,
		PROTOCOL_RAW,
		PROTOCOL_RAW,
		NoLoadBalancer,
		2,
		DefaultTCPServerTimings)

	var handlerTcp testTCPServerRemoteAddress

	go tcpServer.Run(&handlerTcp)

	handlerTcp.wasConnectedCalled = &sync.WaitGroup{}
	handlerTcp.wasConnectedCalled.Add(1)

	conn1, err1 := net.Dial("tcp", "127.0.0.1:4005")
	assert.Nil(t, err1, "Connecting to server")
	assert.NotNil(t, conn1)

	handlerTcp.wasConnectedCalled.Wait() // ToDo: This potentially freezes the test, add timeout impl

	assert.Equal(t, "127.0.0.1", handlerTcp.lastConnectionSource)
}
