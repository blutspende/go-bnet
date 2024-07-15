package bloodlabnet

import (
	"fmt"
	"io"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/blutspende/go-bnet/protocol/utilities"

	"net"

	"github.com/blutspende/go-bnet/protocol"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type testSessionMock struct {
	errorThatWillbeReturnedOnConnect error
	receiveQ                         chan []byte
	lastConnectedIp                  string
	signalReady                      chan bool
	didReceiveDisconnectMessage      bool
	occuredErrorTypes                []ErrorType
}

func (s *testSessionMock) Connected(session Session) error {

	if s.errorThatWillbeReturnedOnConnect != nil { // This is for the "error on connect test"
		return s.errorThatWillbeReturnedOnConnect
	}

	s.lastConnectedIp, _ = session.RemoteAddress()
	s.signalReady <- true
	return nil
}

func (s *testSessionMock) Disconnected(session Session) {
	s.didReceiveDisconnectMessage = true
}

func (s *testSessionMock) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) error {
	s.lastConnectedIp, _ = session.RemoteAddress()
	s.receiveQ <- fileData

	anResponse := make([][]byte, 0)
	anResponse = append(anResponse, []byte("An adequate response"))
	session.Send(anResponse)

	return nil
}

func (s *testSessionMock) Error(session Session, errorType ErrorType, err error) {
	s.occuredErrorTypes = append(s.occuredErrorTypes, errorType)
}

// --------------------------------------------------------------------------------------------
// The server is expected to buffer received data. When a timeout occurs and the connection is
// closed, that buffer needs to be flushed to its handler. With the Raw Protocol the change
// should be read instantly.
// --------------------------------------------------------------------------------------------
func TestRawDataProtocolWithTimeoutFlushMs(t *testing.T) {
	settings := DefaultTCPServerSettings
	settings.SessionAfterFirstByte = false
	tcpServer := CreateNewTCPServerInstance(4001,
		protocol.Raw(protocol.DefaultRawProtocolSettings()), NoLoadBalancer, 100, settings)

	var handler testSessionMock
	handler.receiveQ = make(chan []byte, 500)
	handler.signalReady = make(chan bool)

	isServerHalted := false

	// run the mainloop, observe that it closes later
	waitRunning := sync.Mutex{}
	waitRunning.Lock()
	go func() {
		waitRunning.Unlock()
		tcpServer.Run(&handler)
		isServerHalted = true
	}()
	waitRunning.Lock()
	// allow time for the server to startup. This could be better handled with a status (TODO)
	time.Sleep(1 * time.Second)

	// connect and expect connection handler to signal
	clientConn, err := net.Dial("tcp", "127.0.0.1:4001")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to dial (this is not an error, rather a problem of the unit test itself")
		t.Fail()
		os.Exit(1)
	}
	select {
	case isReady := <-handler.signalReady:
		if isReady {
			assert.Equal(t, "127.0.0.1", handler.lastConnectedIp)
		}
	case <-time.After(2 * time.Second):
		{
			t.Fatalf("Can not start tcp server")
		}
	}

	// send data to server, expect it to receive it
	const TESTSTRING = "Hello its me"
	_, err = clientConn.Write([]byte(TESTSTRING))
	assert.Nil(t, err)

	select { // expecting to receive the same string
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

	time.Sleep(time.Second * 1)

	assert.True(t, handler.didReceiveDisconnectMessage, "Disconnect message was send")

	tcpServer.Stop()
	assert.True(t, isServerHalted, "Server has been stopped")
}

// --------------------------------------------------------------------------------------------
// When writing large amounts of data into one connection, that data should be processed
// sequentially. Using the STX-ETX here to indicate start and end of the messaage
// --------------------------------------------------------------------------------------------
func TestSendingLargeAmount(t *testing.T) {

	tcpServer := CreateNewTCPServerInstance(4009,
		protocol.STXETX(protocol.DefaultSTXETXProtocolSettings()), NoLoadBalancer, 100, DefaultTCPServerSettings)

	handler := &testSessionMock{
		receiveQ:    make(chan []byte, 500),
		signalReady: make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
	}
	go tcpServer.Run(handler)
	tcpServer.WaitReady()
	clientConn, err := net.Dial("tcp", "127.0.0.1:4009")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to dial (this is not an error, rather a problem of the unit test itself)")
	}

	dataIsSendSignal := make(chan bool)
	go func() {
		clientConn.Write([]byte{utilities.STX})
		for i := 0; i < 20000; i++ {
			clientConn.Write([]byte("A lot of data is pushed into the server, lets see how it deals with it"))
		}
		clientConn.Write([]byte{utilities.ETX})
		dataIsSendSignal <- true // dignal that writing is done
	}()

	// wait for the signal that data is sent
	select {
	case <-dataIsSendSignal: //
	case <-time.After(3 * time.Second):
		t.Fail() // timeout
	}

	expectString := ""
	for j := 0; j < 20000; j++ {
		expectString = expectString + "A lot of data is pushed into the server, lets see how it deals with it"
	}
	in := <-handler.receiveQ

	// expecting EXACTLY the data out that we put in
	assert.Equal(t, expectString, string(in))

	clientConn.Close()
	tcpServer.Stop()
}

// --------------------------------------------------------------------------------------------
// When reaching the connection limit, the server should decline further connections
// --------------------------------------------------------------------------------------------
func TestTCPServerMaxConnections(t *testing.T) {

	tcpServer := CreateNewTCPServerInstance(4002,
		protocol.Raw(protocol.DefaultRawProtocolSettings()),
		NoLoadBalancer,
		2,
		DefaultTCPServerSettings)

	handlerTcp := &testSessionMock{
		receiveQ:          make(chan []byte, 500),
		signalReady:       make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
		occuredErrorTypes: make([]ErrorType, 0),
	}

	go tcpServer.Run(handlerTcp)
	tcpServer.WaitReady()

	conn1, err1 := net.Dial("tcp", "127.0.0.1:4002")
	conn1.Write([]byte("1"))
	// connections only establish a session (therefore increase connection count) after sending at least one byte
	assert.Nil(t, err1)
	assert.NotNil(t, conn1)

	conn2, err2 := net.Dial("tcp", "127.0.0.1:4002")
	conn2.Write([]byte("1"))
	assert.Nil(t, err2)
	assert.NotNil(t, conn2)

	conn3, err3 := net.Dial("tcp", "127.0.0.1:4002")
	conn3.Write([]byte("1"))
	assert.Nil(t, err3)
	assert.NotNil(t, conn3)

	// sessions for the clients start asynchronous, we need to wait for the process to start in order to count the clients
	time.Sleep(time.Second * 1)

	assert.Equal(t, ErrorMaxConnections, handlerTcp.occuredErrorTypes[0])

	tcpServer.Stop()
}

func TestSessionLimitDoesNotAffectConnections(t *testing.T) {
	settings := DefaultTCPServerSettings
	settings.SessionInitiationTimeout = time.Duration(1)
	tcpServer := CreateNewTCPServerInstance(4070,
		protocol.Raw(protocol.DefaultRawProtocolSettings()),
		NoLoadBalancer,
		2,
		settings)

	handlerTcp := &testSessionMock{
		receiveQ:          make(chan []byte, 500),
		signalReady:       make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
		occuredErrorTypes: make([]ErrorType, 0),
	}

	go tcpServer.Run(handlerTcp)
	tcpServer.WaitReady()

	// first and second connections do not send data, therefore do not count towards max connections limit
	conn1, err1 := net.Dial("tcp", "127.0.0.1:4070")
	assert.Nil(t, err1)
	assert.NotNil(t, conn1)
	conn2, err2 := net.Dial("tcp", "127.0.0.1:4070")
	assert.Nil(t, err2)
	assert.NotNil(t, conn2)

	// third connection sends one byte to the TCP server, and a session is created, but since the first two did not create new sessions,
	// there is no error for exceeding maximum connections
	conn3, err3 := net.Dial("tcp", "127.0.0.1:4070")
	conn3.Write([]byte{0xFF})
	assert.Nil(t, err3)
	assert.NotNil(t, conn3)
	time.Sleep(time.Second * 1)
	assert.Equal(t, 0, len(handlerTcp.occuredErrorTypes))
	select { // expecting to receive the same byte
	case receivedMsg := <-handlerTcp.receiveQ:
		assert.Equal(t, []byte{0xFF}, receivedMsg)
	case <-time.After(2 * time.Second):
		t.Fatalf("Can not receive messages from the client")
	}

	tcpServer.Stop()
}

func TestSessionTerminatesIfFirstByteNotSent(t *testing.T) {
	settings := DefaultTCPServerSettings
	settings.SessionInitiationTimeout = time.Second
	tcpServer := CreateNewTCPServerInstance(4003,
		protocol.Raw(protocol.DefaultRawProtocolSettings()),
		NoLoadBalancer,
		2,
		settings)

	handlerTcp := &testSessionMock{
		receiveQ:          make(chan []byte, 500),
		signalReady:       make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
		occuredErrorTypes: make([]ErrorType, 0),
	}

	go tcpServer.Run(handlerTcp)
	tcpServer.WaitReady()

	conn1, err := net.Dial("tcp", "127.0.0.1:4003")
	assert.Nil(t, err)
	// reading from a socket is the only reliable way to determine whether the server side had been closed, or not
	conn1.SetDeadline(time.Now().Add(settings.SessionInitiationTimeout * 2))
	b := make([]byte, 2)
	_, err = conn1.Read(b)
	assert.Equal(t, io.EOF, err)

	tcpServer.Stop()
}

// --------------------------------------------------------------------------------------------
// Server identifies the remote-Address on direct connections
// --------------------------------------------------------------------------------------------
func TestTCPServerIdentifyRemoteAddress(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4005,
		protocol.Raw(protocol.DefaultRawProtocolSettings()),
		NoLoadBalancer,
		2,
		DefaultTCPServerSettings)

	handlerTcp := &testSessionMock{
		receiveQ:    make(chan []byte, 500),
		signalReady: make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
		//		occuredErrorTypes: make([]ErrorType, 0),
	}

	go tcpServer.Run(handlerTcp)
	tcpServer.WaitReady()

	conn1, err1 := net.Dial("tcp", "127.0.0.1:4005")
	conn1.Write([]byte{0xFF})
	assert.Nil(t, err1)
	assert.NotNil(t, conn1)

	// sessions for the clients start asynchronous, we need to wait for the process to start in order to count the clients
	time.Sleep(time.Second * 1) // sessions start async, therefor a short waitign is required

	assert.Equal(t, "127.0.0.1", handlerTcp.lastConnectedIp)

	tcpServer.Stop()
}

// --------------------------------------------------------------------------------------------
// When reaching the connection limit, the server should decline further connections
// --------------------------------------------------------------------------------------------
func TestTCPServerConnectionInitiationTimeout(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4002,
		protocol.Raw(),
		NoLoadBalancer,
		2,
		TCPServerConfiguration{
			Timeout:                  time.Second * 3,
			Deadline:                 time.Millisecond * 200,
			FlushBufferTimoutMs:      500,
			PollInterval:             time.Second * 60,
			SessionAfterFirstByte:    true,        // Sessions are initiated after reading the first bytes (avoids disconnects)
			SessionInitiationTimeout: time.Second, // Waiting 30 sec by default
		})

	handlerTcp := &testSessionMock{
		receiveQ:          make(chan []byte, 500),
		signalReady:       make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
		occuredErrorTypes: make([]ErrorType, 0),
	}

	go tcpServer.Run(handlerTcp)
	tcpServer.WaitReady()
	defer tcpServer.Stop()
	conn1, err1 := net.Dial("tcp", "127.0.0.1:4002")
	assert.Nil(t, err1)
	assert.NotNil(t, conn1)
	time.Sleep(2 * time.Second)
	conn1.Write([]byte("connection should be already closed and this should never be received"))
	assert.Equal(t, 0, len(handlerTcp.receiveQ))
}

// --------------------------------------------------------------------------------------------
// Sending "out of frame" extra data, classical used for buffer-overflows, because protocols
// often turn a blind eye to unexpected data. The expected behaviour here is that that data
// plainly gets ignored
// --------------------------------------------------------------------------------------------
func TestSTXETXOutOfFrameData(t *testing.T) {

	const TESTSTRING = "Submitting data test"
	const TESTSTRING2 = "This is the second datapackage within the same datastream"

	tcpServer := CreateNewTCPServerInstance(4010,
		protocol.STXETX(protocol.DefaultSTXETXProtocolSettings()),
		NoLoadBalancer,
		100,
		DefaultTCPServerSettings)

	handler := &testSessionMock{
		receiveQ:          make(chan []byte, 500),
		signalReady:       make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
		occuredErrorTypes: make([]ErrorType, 0),
	}

	go tcpServer.Run(handler)
	tcpServer.WaitReady()

	clientConn, err := net.Dial("tcp", "127.0.0.1:4010")
	assert.Nil(t, err)

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

// --------------------------------------------------------------------------------------------
// When connections are dropped, their memory needs to be freed and the slot needs to be
// made available again. This test opens 2 connections, closes both, then opens a third
// one and verifies that the communication still works as expected by sending data.
// --------------------------------------------------------------------------------------------
func TestDropConnectionsAfterError(t *testing.T) {
	tcpServer := CreateNewTCPServerInstance(4013,
		protocol.STXETX(),
		NoLoadBalancer,
		2 /* MAX Connection */)

	handler := &testSessionMock{
		receiveQ:          make(chan []byte, 500),
		signalReady:       make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
		occuredErrorTypes: make([]ErrorType, 0),
	}

	go tcpServer.Run(handler)
	tcpServer.WaitReady()

	conn, err := net.Dial("tcp", "127.0.0.1:4013")
	assert.Nil(t, err)
	assert.NotNil(t, conn)
	conn.Close()

	conn2, err := net.Dial("tcp", "127.0.0.1:4013")
	assert.Nil(t, err)
	assert.NotNil(t, conn2)
	conn2.Close()

	clientConn, err := net.Dial("tcp", "127.0.0.1:4013")
	assert.Nil(t, err)
	assert.NotNil(t, clientConn)

	_, err = clientConn.Write([]byte("\u0002Test connection\u0003"))
	assert.Nil(t, err)

	select {
	case receivedMsg := <-handler.receiveQ:
		assert.NotNil(t, receivedMsg, "Received a valid response")
		assert.Equal(t, "Test connection", string(receivedMsg))
	case <-time.After(2 * time.Second):
		t.Fatalf("Timout waiting on valid response. This means the Server was unable to receive this message ")
	}

	tcpServer.Stop()
}

// ----------------------------------------------------------------------------------------
// Run a full protocol of an everyday laboratoy instrument. This is a typical use case
// for bnet
// ----------------------------------------------------------------------------------------
func TestLis1A1Protocol(t *testing.T) {

	tcpServer := CreateNewTCPServerInstance(4011,
		protocol.Lis1A1Protocol(protocol.DefaultLis1A1ProtocolSettings()),
		NoLoadBalancer,
		100,
		DefaultTCPServerSettings)

	handler := &testSessionMock{
		receiveQ:          make(chan []byte, 500),
		signalReady:       make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
		occuredErrorTypes: make([]ErrorType, 0),
	}
	go tcpServer.Run(handler)
	tcpServer.WaitReady()

	clientConn, err := net.Dial("tcp", "127.0.0.1:4011")
	assert.Nil(t, err)

	// An inline Script for the expected communication-flow
	communicationFlow := []struct {
		Receive bool
		Data    []byte
	}{
		{Data: []byte{utilities.ENQ}, Receive: false},
		{Data: []byte{utilities.ACK}, Receive: true},
		{Data: []byte{utilities.STX}, Receive: false},
		{Data: []byte("1H|\\^&|||"), Receive: false},
		{Data: []byte{utilities.CR}, Receive: false},
		{Data: []byte{utilities.ETX}, Receive: false},
		{Data: []byte{'5', '9'}, Receive: false},
		{Data: []byte{utilities.CR, utilities.LF}, Receive: false},
		{Data: []byte{utilities.ACK}, Receive: true},

		{Data: []byte{utilities.STX}, Receive: false},
		{Data: []byte("2P|1||777025164810"), Receive: false},
		{Data: []byte{utilities.CR}, Receive: false},
		{Data: []byte{utilities.ETX}, Receive: false},
		{Data: []byte{'A', '7'}, Receive: false},
		{Data: []byte{utilities.CR, utilities.LF}, Receive: false},
		{Data: []byte{utilities.ACK}, Receive: true},

		{Data: []byte{utilities.STX}, Receive: false},
		{Data: []byte("3O|1|||^^^SARSCOV2IGG||20200811095913"), Receive: false},
		{Data: []byte{utilities.CR}, Receive: false},
		{Data: []byte{utilities.ETX}, Receive: false},
		{Data: []byte{'B', '8'}, Receive: false},
		{Data: []byte{utilities.CR, utilities.LF}, Receive: false},
		{Data: []byte{utilities.ACK}, Receive: true},

		{Data: []byte{utilities.STX}, Receive: false},
		{Data: []byte("4R|1|^^^SARSCOV2IGG|0,18|Ratio|"), Receive: false},
		{Data: []byte{utilities.CR}, Receive: false},
		{Data: []byte{utilities.ETX}, Receive: false},
		{Data: []byte{'3', 'B'}, Receive: false},
		{Data: []byte{utilities.CR, utilities.LF}, Receive: false},
		{Data: []byte{utilities.ACK}, Receive: true},

		{Data: []byte{utilities.STX}, Receive: false},
		{Data: []byte("5P|2||777642348910"), Receive: false},
		{Data: []byte{utilities.CR}, Receive: false},
		{Data: []byte{utilities.ETX}, Receive: false},
		{Data: []byte{'B', '5'}, Receive: false},
		{Data: []byte{utilities.CR, utilities.LF}, Receive: false},
		{Data: []byte{utilities.ACK}, Receive: true},

		{Data: []byte{utilities.STX}, Receive: false},
		{Data: []byte("6O|1|||^^^SARSCOV2IGG||20200811095913"), Receive: false},
		{Data: []byte{utilities.CR}, Receive: false},
		{Data: []byte{utilities.ETX}, Receive: false},
		{Data: []byte{'B', 'B'}, Receive: false},
		{Data: []byte{utilities.CR, utilities.LF}, Receive: false},
		{Data: []byte{utilities.ACK}, Receive: true},

		{Data: []byte{utilities.STX}, Receive: false},
		{Data: []byte("7R|1|^^^SARSCOV2IGG|0,18|Ratio|"), Receive: false},
		{Data: []byte{utilities.CR}, Receive: false},
		{Data: []byte{utilities.ETX}, Receive: false},
		{Data: []byte{'3', 'E'}, Receive: false},
		{Data: []byte{utilities.CR, utilities.LF}, Receive: false},
		{Data: []byte{utilities.ACK}, Receive: true},

		{Data: []byte{utilities.STX}, Receive: false},
		{Data: []byte("0L|1|N"), Receive: false},
		{Data: []byte{utilities.CR}, Receive: false},
		{Data: []byte{utilities.ETX}, Receive: false},
		{Data: []byte{'0', '3'}, Receive: false},
		{Data: []byte{utilities.CR, utilities.LF}, Receive: false},
		{Data: []byte{utilities.EOT}, Receive: false},
	}

	for _, rec := range communicationFlow {
		if !rec.Receive {

			clientConn.Write(rec.Data)

		} else {

			data := make([]byte, 500)
			n, err := clientConn.Read(data)

			assert.Nil(t, err)

			if n == len(rec.Data) {
				for i, s := range data {
					if s != data[i] {
						t.Error(fmt.Sprint("Invalid response. Expected:", rec.Data, "(", string(rec.Data), ") but got ", data, "(", string(data), ")"))
					}
				}
			} else {
				t.Error(fmt.Sprint("Invalid response. Expected:", rec.Data, "(", string(rec.Data), ") but got ", data, "(", string(data), ")"))
			}
		}
	}

	tcpServer.Stop()
}

// ----------------------------------------------------------------------------------------
// When a connection is made and the Session does return an error, the Server is
// expected to instantly close and drop that connection.
// ----------------------------------------------------------------------------------------
func TestTCPServerDeclineConnection(t *testing.T) {

	tcpServer := CreateNewTCPServerInstance(40015,
		protocol.Raw(protocol.DefaultRawProtocolSettings()),
		NoLoadBalancer,
		2,
		DefaultTCPServerSettings)

	handler := &testSessionMock{
		errorThatWillbeReturnedOnConnect: fmt.Errorf("Invalid connection"),
		//		receiveQ:                         make(chan []byte, 500),
		//		signalReady:                      make(chan bool, 100), // buffered so that we can ignore this signal without blocking process
		//		occuredErrorTypes:                make([]ErrorType, 0),
	}

	go tcpServer.Run(handler)
	tcpServer.WaitReady()

	conn1, err1 := net.Dial("tcp", "127.0.0.1:4015")
	assert.NotNil(t, err1) // the server instantly declines this connection
	assert.Nil(t, conn1)

	tcpServer.Stop()
}
