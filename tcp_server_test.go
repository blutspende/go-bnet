package bloodlabnet

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"net"

	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol"
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
	log.Fatal("Fatal error:", err)
}

func TestRawDataProtocolWithTimeoutFlushMs(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4001,
		protocol.Raw(protocol.DefaultRawProtocolSettings()), NoLoadBalancer, 100, DefaultTCPServerSettings)

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
		log.Fatalf("Failed to dial (this is not an error, rather a problem of the unit test itself) : %s", err)
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

	clientConn.Close()
	tcpServer.Stop()

	time.Sleep(time.Second * 1)

	assert.True(t, handler.didReceiveDisconnectMessage, "Disconnect message was send")
	assert.True(t, isServerHalted, "Server has been stopped")
}

// Create some stress by pushing a lot of transmissions
/*
func TestRawDataProtocolSendingStress(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4003,
		protocol.Raw(protocol.DefaultRawProtocolSettings()), NoLoadBalancer, 100, DefaultTCPServerSettings)
	var handler testRawDataProtocolSession
	handler.receiveQ = make(chan []byte, 500)
	go tcpServer.Run(&handler)

	clientConn, err := net.Dial("tcp", "127.0.0.1:4003")
	if err != nil {
		log.Fatalf("Failed to dial (this is not an error, rather a problem of the unit test itself) : %s", err)
	}

	go func() {
		for i := 0; i < 20000; i++ {
			clientConn.Write([]byte("A lot of data is pushed into the server, lets see how it deals with it"))
		}
	}()
	time.Sleep(time.Second * 1)
	clientConn.Close()

	time.Sleep(time.Second * 1)
	expectString := ""
	for j := 0; j < 20000; j++ {
		expectString = expectString + "A lot of data is pushed into the server, lets see how it deals with it"
	}
	in := <-handler.receiveQ
	assert.Equal(t, expectString, string(in))
}
*/
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
		protocol.Raw(protocol.DefaultRawProtocolSettings()),
		NoLoadBalancer,
		2,
		DefaultTCPServerSettings)

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

	tcpServer.Stop()
}

//------------------------------------------------------
// Server identifies the remote-Address
//------------------------------------------------------
type testTCPServerRemoteAddress struct {
	lastConnectionSource string
}

func (s *testTCPServerRemoteAddress) Connected(session Session) {
	s.lastConnectionSource, _ = session.RemoteAddress()
}

func (s *testTCPServerRemoteAddress) Disconnected(session Session) {
}

func (s *testTCPServerRemoteAddress) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
}

func (s *testTCPServerRemoteAddress) Error(session Session, errorType ErrorType, err error) {
}

func TestTCPServerIdentifyRemoteAddress(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4005,
		protocol.Raw(protocol.DefaultRawProtocolSettings()),
		NoLoadBalancer,
		2,
		DefaultTCPServerSettings)

	var handlerTcp testTCPServerRemoteAddress

	go tcpServer.Run(&handlerTcp)

	conn1, err1 := net.Dial("tcp", "127.0.0.1:4005")
	assert.Nil(t, err1)
	assert.NotNil(t, conn1)

	time.Sleep(time.Second * 1) // sessions start async, therefor a short waitign is required

	assert.Equal(t, "127.0.0.1", handlerTcp.lastConnectionSource)
}

//--------------------------------------------------------------
// Test STX Protocol
//--------------------------------------------------------------
type testSTXETXProtocolSession struct {
	receiveQ                    chan []byte
	didReceiveDisconnectMessage bool
}

func (s *testSTXETXProtocolSession) Connected(session Session) {
}

func (s *testSTXETXProtocolSession) Disconnected(session Session) {
	s.didReceiveDisconnectMessage = true
}

func (s *testSTXETXProtocolSession) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
	s.receiveQ <- fileData

	// build a response that exceeds the MTU to ensure stx-etx reads start and stop codes
	largeDataPackage := ""
	for i := 0; i < 80000; i++ {
		largeDataPackage = largeDataPackage + "X"
	}
	session.Send([]byte(largeDataPackage))
}

func (s *testSTXETXProtocolSession) Error(session Session, errorType ErrorType, err error) {
	log.Fatal("Fatal error:", err)
}

func TestSTXETXProtocol(t *testing.T) {

	const TESTSTRING = "Submitting data test"

	tcpServer := CreateNewTCPServerInstance(4009,
		protocol.STXETX(protocol.DefaultSTXETXProtocolSettings()),
		NoLoadBalancer,
		100,
		DefaultTCPServerSettings)

	var handler testSTXETXProtocolSession
	handler.receiveQ = make(chan []byte, 500)

	go tcpServer.Run(&handler)

	clientConn, err := net.Dial("tcp", "127.0.0.1:4009")
	if err != nil {
		log.Fatalf("Failed to dial (this is not an error, rather a problem of the unit test itself) : %s", err)
	}

	_, err = clientConn.Write([]byte("\u0002" + TESTSTRING + "\u0003"))
	assert.Nil(t, err)

	select {
	case receivedMsg := <-handler.receiveQ:
		assert.NotNil(t, receivedMsg, "Received a valid response")
		assert.Equal(t, TESTSTRING, string(receivedMsg))
	case <-time.After(2 * time.Second):
		t.Fatalf("Timout waiting on valid response. This means the Server was unable to receive this message ")
	}

	// At this point the server should respond to the received data with a few XXXes
	_ = clientConn.SetDeadline(time.Now().Add(time.Second * 2))
	buffer := make([]byte, 90000)
	n, err := clientConn.Read(buffer)

	assert.Nil(t, err, "Reading from client.")

	largeDataPackage := ""
	for i := 0; i < 80000; i++ {
		largeDataPackage = largeDataPackage + "X"
	}

	assert.Equal(t, "\u0002"+largeDataPackage+"\u0003", string(buffer[:n]))

	tcpServer.Stop()
}

//----------------------------------------------------------------------------------------
// Test STX Protocol with Packages larger than the buffer and 2 messages in the stream
// STX first string ETX data to be ignored STX second string ETX
// expecting this to create two data-received events
//----------------------------------------------------------------------------------------
type testSTXETXBufferOverflowProtocolSession struct {
	receiveQ                    chan []byte
	didReceiveDisconnectMessage bool
}

func (s *testSTXETXBufferOverflowProtocolSession) Connected(session Session) {
}

func (s *testSTXETXBufferOverflowProtocolSession) Disconnected(session Session) {
	s.didReceiveDisconnectMessage = true
}

func (s *testSTXETXBufferOverflowProtocolSession) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
	fmt.Println("Eventhandler : ", string(fileData))
	s.receiveQ <- fileData
}

func (s *testSTXETXBufferOverflowProtocolSession) Error(session Session, errorType ErrorType, err error) {
	log.Fatal("Fatal error:", err)
}

func TestSTXETXBufferOverflowProtocol(t *testing.T) {

	const TESTSTRING = "Submitting data test"
	const TESTSTRING2 = "This is the second datapackage within the same datastream"

	tcpServer := CreateNewTCPServerInstance(4010,
		protocol.STXETX(protocol.DefaultSTXETXProtocolSettings()),
		NoLoadBalancer,
		100,
		DefaultTCPServerSettings)

	var handler testSTXETXBufferOverflowProtocolSession
	handler.receiveQ = make(chan []byte, 500)

	go tcpServer.Run(&handler)

	clientConn, err := net.Dial("tcp", "127.0.0.1:4010")
	if err != nil {
		log.Fatalf("Failed to dial (this is not an error, rather a problem of the unit test itself) : %s", err)
	}

	_, err = clientConn.Write([]byte("\u0002" + TESTSTRING + "\u0003IngoredData}\u0002" + TESTSTRING2 + "\u0003"))
	assert.Nil(t, err)

	select {
	case receivedMsg := <-handler.receiveQ:
		assert.NotNil(t, receivedMsg, "Received a valid response")
		assert.Equal(t, TESTSTRING, string(receivedMsg))
	case <-time.After(2 * time.Second):
		t.Fatalf("Timout waiting on valid response. This means the Server was unable to receive this message ")
	}

	select {
	case receivedMsg := <-handler.receiveQ:
		assert.NotNil(t, receivedMsg, "Received a valid response")
		assert.Equal(t, TESTSTRING2, string(receivedMsg))
	case <-time.After(2 * time.Second):
		t.Fatalf("Timout waiting on valid response. This means the Server was unable to receive this message ")
	}

	tcpServer.Stop()
}
