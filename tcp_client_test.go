package main

import (
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol"
	"github.com/stretchr/testify/assert"
)

/* Run a Server for one connection, reading from socket, writing to channel
reading from channel, writing to socket Server stops when client disconnects
*/

func runTCPMockServer(port int, tcpMockServerSendQ chan []byte, tcpMockServerReceiveQ chan []byte) {
	var listener net.Listener

	if listener != nil { // In case previous session got stuck remove it
		listener.Close()
	}
	go func() {

		listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			panic(err)
		}
		defer listener.Close()

		conn, e := listener.Accept()
		if e != nil {
			panic(err)
		}
		buf := make([]byte, 100)
		for {
			conn.SetDeadline(time.Now().Add(time.Millisecond * 200))
			n, err := conn.Read(buf)
			if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() { // timeout dont care
			} else if err == io.EOF {
				return
			} else {
				if err != nil {
					panic(err)
				} else {
					tcpMockServerReceiveQ <- buf[:n]
				}
			}

			select {
			case x, ok := <-tcpMockServerSendQ:
				if ok {
					conn.SetDeadline(time.Now().Add(time.Millisecond * 200))
					if _, err = conn.Write(x); err != nil {
						panic(err)
					}
				}
			default:
			}
		}

	}()
}

/*
	Connect to TCP-Server, Read Data and transmit data

	For parallel test-execution keep server-port unique throughout the suite
*/
func TestClientConnectReceiveAndSendRaw(t *testing.T) {

	var tcpMockServerSendQ chan []byte = make(chan []byte, 1)
	var tcpMockServerReceiveQ chan []byte = make(chan []byte, 1)
	runTCPMockServer(4001, tcpMockServerSendQ, tcpMockServerReceiveQ)

	tcpClient := CreateNewTCPClient("127.0.0.1", 4001,
		protocol.Raw(protocol.DefaultRawProtocolSettings()),
		NoLoadbalancer,
		DefaultTCPServerSettings)

	err := tcpClient.Connect()
	assert.Nil(t, err)

	const TESTSTRINGSEND = "Testdata that is beeing send"
	n, err := tcpClient.Send([]byte(TESTSTRINGSEND))
	if err != nil {
		panic(err)
	}
	sendMsg := <-tcpMockServerReceiveQ
	assert.Equal(t, TESTSTRINGSEND, string(sendMsg[:n]))

	// Read data the instrument sent
	const TESTSTRINGRECEIVE = "This data is definateley beeing transmitted"
	tcpMockServerSendQ <- []byte(TESTSTRINGRECEIVE)
	receivedMsg, err := tcpClient.Receive()
	assert.Nil(t, err)
	assert.Equal(t, TESTSTRINGRECEIVE, string(receivedMsg))

	tcpClient.Stop()
}

/****************************************************************
Protocol wrapped STX Send and Receive
****************************************************************/
func TestClientProtocolSTXETX(t *testing.T) {
	var tcpMockServerSendQ chan []byte = make(chan []byte, 1)
	var tcpMockServerReceiveQ chan []byte = make(chan []byte, 1)
	runTCPMockServer(4002, tcpMockServerSendQ, tcpMockServerReceiveQ)

	tcpClient := CreateNewTCPClient("127.0.0.1", 4002,
		protocol.STXETX(protocol.DefaultSTXETXProtocolSettings()),
		NoLoadbalancer, DefaultTCPServerSettings)

	// Receiving from instrument expecting STX and ETX removed
	TESTSTRING := "H|\\^&|||bloodlab-net|e2etest||||||||20220614163728\nL|1|N"
	tcpMockServerSendQ <- []byte("\u0002" + TESTSTRING + "\u0003")
	receivedMsg, err := tcpClient.Receive()
	assert.Nil(t, err)
	assert.Equal(t, TESTSTRING, string(receivedMsg)) // stripped stx and etx

	// Sending to instrument expecting STX and ETX added
	TESTSTRING = "Not so important what we send here"
	_, err = tcpClient.Send([]byte(TESTSTRING))
	assert.Nil(t, err)
	dataReceived := <-tcpMockServerReceiveQ
	assert.Equal(t, "\u0002"+TESTSTRING+"\u0003", string(dataReceived))

	tcpClient.Stop()
}

/****************************************************************
Test client remote address
****************************************************************/
func TestClientRemoteAddress(t *testing.T) {
	var tcpMockServerSendQ chan []byte = make(chan []byte, 1)
	var tcpMockServerReceiveQ chan []byte = make(chan []byte, 1)
	runTCPMockServer(4003, tcpMockServerSendQ, tcpMockServerReceiveQ)

	tcpClient := CreateNewTCPClient("127.0.0.1", 4003,
		protocol.STXETX(&protocol.STXETXProtocolSettings{}),
		NoLoadbalancer,
		DefaultTCPServerSettings)

	tcpClient.Connect()
	addr, _ := tcpClient.RemoteAddress()
	assert.Equal(t, "127.0.0.1", addr)
}

/****************************************************************
Test client with Run-Session to connect, handle async events
****************************************************************/
type ClientTestSession struct {
	receiveBuffer            string
	connectionEventOccured   bool
	disconnectedEventOccured bool
}

func (s *ClientTestSession) Connected(session Session) {
	s.connectionEventOccured = true
}

func (s *ClientTestSession) Disconnected(session Session) {
	s.disconnectedEventOccured = true
}

func (s *ClientTestSession) DataReceived(session Session, fileData []byte, receiveTimestamp time.Time) {
	s.receiveBuffer = s.receiveBuffer + string(fileData)
}

func (s *ClientTestSession) Error(session Session, typeOfError ErrorType, err error) {
}

func TestClientRun(t *testing.T) {
	var tcpMockServerSendQ chan []byte = make(chan []byte, 1)
	var tcpMockServerReceiveQ chan []byte = make(chan []byte, 1)
	runTCPMockServer(4004, tcpMockServerSendQ, tcpMockServerReceiveQ)

	tcpClient := CreateNewTCPClient("127.0.0.1", 4004,
		protocol.Raw(protocol.DefaultRawProtocolSettings()),
		NoLoadbalancer,
		DefaultTCPServerSettings)

	var session ClientTestSession
	session.connectionEventOccured = false
	session.receiveBuffer = ""

	var eventLoopIsActive = true
	go func() {
		tcpClient.Run(&session)
		eventLoopIsActive = false
	}()

	//TODO: Waiting is not a good solution, instead check the state of the loop with timeout
	time.Sleep(time.Second * 1)

	const TESTSTRING = "Some Testdata!"

	// sending data from instrument to this client
	tcpMockServerSendQ <- []byte(TESTSTRING)

	time.Sleep(time.Second * 1) // TODO: Wait data beeing sent

	assert.True(t, session.connectionEventOccured, "The event 'connected' was triggered")
	assert.Equal(t, session.receiveBuffer, TESTSTRING)

	// Does Stop really stop the eventloop ?
	tcpClient.Stop()

	time.Sleep(time.Second * 1) // TODO: Wait data beeing sent

	assert.False(t, eventLoopIsActive, "Eventloop did terminated")
	assert.True(t, session.disconnectedEventOccured, "The event 'Disconnected' was triggered")
}
