package bloodlabnet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testFtpHandler struct {
	hadConnected bool
	hadError     bool
}

func (th *testFtpHandler) Connected(session Session) error {
	th.hadConnected = true
	return nil
}

func (th *testFtpHandler) DataReceived(session Session, data []byte, receiveTimestamp time.Time) error {
	return nil
}

func (th *testFtpHandler) Disconnected(session Session) {

}

func (th *testFtpHandler) Error(session Session, typeOfError ErrorType, err error) {
	th.hadError = true
}

func TestSFTPServerConnect(t *testing.T) {

	// staart a ftp server
	// place a file istvan.dat here

	var myHandler testFtpHandler

	server, err := CreateFTP(SFTP, "localhost", 5000, "/", "*.dat",
		DefaultFTPConfig().UserPass("Istvan", "Pass"))

	assert.Nil(t, err)
	go server.Run(&myHandler)

	// wait for data or timeout

	assert.True(t, myHandler.hadConnected)
	assert.False(t, myHandler.hadError)
}
