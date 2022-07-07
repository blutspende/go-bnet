package main

import (
	"fmt"
	"time"

	bnet "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net"
	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol"
)

type MySessionHandler struct {
}

func (s *MySessionHandler) Connected(session bnet.Session) {
	fmt.Println("Connect Event")
}

func (s *MySessionHandler) Disconnected(session bnet.Session) {
	fmt.Println("Disconnected Event")
}

func (s *MySessionHandler) Error(session bnet.Session, errorType bnet.ErrorType, err error) {
	fmt.Println(err)
}

func (s *MySessionHandler) DataReceived(session bnet.Session, data []byte, receiveTimestamp time.Time) {
	rad, _ := session.RemoteAddress()
	fmt.Printf("From %s received '%s'", rad, string(data))

	session.Send([]byte(fmt.Sprintf("You are sending from %s", rad)))
}

func main() {

	server := bnet.CreateNewTCPServerInstance(4009,
		protocol.STXETX(),
		bnet.HAProxySendProxyV2,
		2) // Max Connections

	server.Run(&MySessionHandler{})
}
