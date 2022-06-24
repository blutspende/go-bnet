package e2etests

import (
	intNet "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/net"
	"github.com/stretchr/testify/assert"
	"log"
	"net"
	"os"
	"testing"
)

func startTCPMockServer() {
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

func Test_Simple_TCP_Client(t *testing.T) {
	go startTCPMockServer()
	tcpClient := intNet.CreateNewTCPClient("127.0.0.1", 4001, intNet.RawProtocol, intNet.NoProxy)

	n, err := tcpClient.Send([]byte("Hallo hier ist client!"))
	sendMsg := <-receiveQ
	assert.Equal(t, "Hallo hier ist client!", string(sendMsg[:n]))
	receivedMsg, err := tcpClient.Receive()
	assert.Nil(t, err)
	assert.Equal(t, "Hallo hier ist Server!", string(receivedMsg))
	tcpClient.Stop()

}

func startTCPMockWrappedSTXProtocolServer() {
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
			_, err = conn.Write([]byte("\u0002H|\\^&|||Bio-Rad|IH v5.2||||||||20220614163728\nP|1||1010448658||NEUWIRTH^Werner||19410807|M||||||||||||||||||||||||^\nO|1|1122206473|1122206473^^^\\1122206473^^^|^^^CH03^^28319^|R|20220309153033|20220309153033|||||||||||11||||20220310155151|||F\nR|1|^^^AntiIgG^CH03^Direct Antiglobulin Test: (IgG) (5054)^|0^^|C||||R||lalina^|20220310130712|20220310155151|11|IH-1000|0300768|lalina\nC|1|ID-Diluent 2^^05761.03.01^20231231\\^^^|CAS^5054027022302331866^50540.27.02^20230228^4||\nR|2|^^^Result^CH03^Direct Antiglobulin Test: (IgG) (5054)^|0^POS^NEG^Ccee^K-^NEG^^^|C||||R||lalina^lalina|20220310130712|20220310155151|11|IH-1000|0300768|lalina\nC|1|^^^||C|1|^^^||\nL|1|N\u0003"))
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

func Test_Simple_TCP_Client_With_Wrapped_STX_Protocol(t *testing.T) {
	go startTCPMockWrappedSTXProtocolServer()
	tcpClient := intNet.CreateNewTCPClient("127.0.0.1", 4001, intNet.ASTMWrappedSTXProtocol, intNet.NoProxy)

	receivedMsg, err := tcpClient.Receive()
	assert.Nil(t, err)
	assert.Equal(t, "H|\\^&|||Bio-Rad|IH v5.2||||||||20220614163728\nP|1||1010448658||NEUWIRTH^Werner||19410807|M||||||||||||||||||||||||^\nO|1|1122206473|1122206473^^^\\1122206473^^^|^^^CH03^^28319^|R|20220309153033|20220309153033|||||||||||11||||20220310155151|||F\nR|1|^^^AntiIgG^CH03^Direct Antiglobulin Test: (IgG) (5054)^|0^^|C||||R||lalina^|20220310130712|20220310155151|11|IH-1000|0300768|lalina\nC|1|ID-Diluent 2^^05761.03.01^20231231\\^^^|CAS^5054027022302331866^50540.27.02^20230228^4||\nR|2|^^^Result^CH03^Direct Antiglobulin Test: (IgG) (5054)^|0^POS^NEG^Ccee^K-^NEG^^^|C||||R||lalina^lalina|20220310130712|20220310155151|11|IH-1000|0300768|lalina\nC|1|^^^||C|1|^^^||\nL|1|N", string(receivedMsg))
	tcpClient.Stop()
}
