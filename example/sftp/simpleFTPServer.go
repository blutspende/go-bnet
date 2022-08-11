package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	bnet "github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net"
)

type testFtpHandler struct {
	hadConnected bool
	hadError     bool
}

func (th *testFtpHandler) Connected(session bnet.Session) error {
	fmt.Println("Connected")
	th.hadConnected = true
	return nil
}

func (th *testFtpHandler) DataReceived(session bnet.Session, data []byte, receiveTimestamp time.Time) error {
	fmt.Println("Data received ", string(data))
	session.Send()
	return errors.New("Dont delete pleeea")
}

func (th *testFtpHandler) Disconnected(session bnet.Session) {
	fmt.Println("Disconnected")
}

func (th *testFtpHandler) Error(session bnet.Session, typeOfError bnet.ErrorType, err error) {
	fmt.Println("error : ", typeOfError)
	th.hadError = true
}

func main() {
	server, err := bnet.CreateSFTPClient(bnet.SFTP, "172.23.114.30", 22, "/tests", "*.dat",
		bnet.DefaultFTPConfig().UserPass("test", "testpaul").PollInterval(5*time.Second),
	)

	if err != nil {
		log.Fatal(err)
	}

	var handler testFtpHandler

	go server.Run(&handler)

	server.Send([][]byte{})
}
