package e2etests

import (
	go_bloodlab_net "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net"
	intNet "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"net"
)

type testSimpleTCPServerReceive struct {
}

var receiveQ = make(chan []byte, 500)

func (s *testSimpleTCPServerReceive) ClientConnected(bloodLabSession go_bloodlab_net.Session) {
	bloodLabSession.WaitTermination()
}

func (s *testSimpleTCPServerReceive) DataReceived(source string, fileData []byte, receiveTimestamp time.Time) {
	receiveQ <- fileData
}

func Test_Simple_TCP_Server_Receive(t *testing.T) {
	tcpServer := intNet.CreateNewTCPServerInstance(4001, intNet.RawProtocol, intNet.NoProxy, 100)
	//	tcpClient := net.CreateNewTCPClient("meingeraet.netzwerk", 4001, net.ASTMWrappedSTXProtocol, net.NoProxy)
	//sftp = CreateSFTPClient('host.hostus.net', '/wodiedateniegen', 'user', 'pass', '1938123083key', DefaultTimings)
	var handlerTcp testSimpleTCPServerReceive

	//var handler_ftp SessionHandlerFTP
	go tcpServer.Run(&handlerTcp)

	time.Sleep(time.Second * 2)
	conn, err := net.Dial(intNet.TCPProtocol, "127.0.0.1:4001")
	if err != nil {
		t.Fail()
		os.Exit(1)
	}

	conn.Write([]byte("Hallo Hier bin ich!"))
	assert.Equal(t, "Hallo Hier bin ich!", string(<-receiveQ))
	conn.Close()
	tcpServer.Stop()
}

type testSimpleTCPServerSend struct {
}

func (s *testSimpleTCPServerSend) ClientConnected(bloodLabSession go_bloodlab_net.Session) {
	for bloodLabSession.IsAlive() {
		data := <-receiveQ
		_, err := bloodLabSession.Send(data)
		if err != nil {
			return
		}
	}
}

func (s *testSimpleTCPServerSend) DataReceived(source string, fileData []byte, receiveTimestamp time.Time) {
	receiveQ <- fileData
}

func Test_Simple_TCP_Server_Send(t *testing.T) {
	tcpServer := intNet.CreateNewTCPServerInstance(4001, intNet.RawProtocol, intNet.NoProxy, 100)
	var handlerTcp testSimpleTCPServerSend

	go tcpServer.Run(&handlerTcp)

	conn, err := net.Dial(intNet.TCPProtocol, "127.0.0.1:4001")
	if err != nil {
		t.Fail()
		os.Exit(1)
	}

	receiveQ <- []byte("ACK")

	buff := make([]byte, 1500)
	_, err = conn.Read(buff)
	assert.Nil(t, err)
	assert.Equal(t, "ACK", string(buff))
	conn.Close()
	tcpServer.Stop()
}
