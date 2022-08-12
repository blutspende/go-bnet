package bloodlabnet

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"os"
	"path"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testFtpHandler struct {
	hadConnected bool
	dataReceived bool
	hadError     bool
	receiveQ     chan []byte
}

func (th *testFtpHandler) Connected(session Session) error {
	th.hadConnected = true
	return nil
}

func (th *testFtpHandler) DataReceived(session Session, data []byte, receiveTimestamp time.Time) error {
	th.dataReceived = true
	th.receiveQ <- data
	fmt.Println("Got something")
	return nil
}

func (th *testFtpHandler) Disconnected(session Session) {

}

func (th *testFtpHandler) Error(session Session, typeOfError ErrorType, err error) {
	th.hadError = true
}

func TestSFTPServerReceive(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(t, err)

	myPath, err := os.Executable()
	assert.Nil(t, err)

	myPath = path.Dir(myPath)
	testFTPdirname := myPath + "/" + "ftptest"
	err = os.Mkdir(testFTPdirname, 0777)
	assert.Nil(t, err)

	sftpserver := CreateSFTPServer(testFTPdirname)
	go sftpserver.RunSimpleSFTPServer(privateKey)

	var myHandler testFtpHandler
	myHandler.receiveQ = make(chan []byte)

	client, err := CreateSFTPClient(SFTP, "localhost", 2022, testFTPdirname, "*.dat",
		DefaultFTPConfig().UserPass("testuser", "tiger").PollInterval(1).DontDeleteAfterRead())

	assert.Nil(t, err)
	go client.Run(&myHandler)

	sftpserver.MakeFile("test.dat", 0544)

	select {
	case receivedMsg := <-myHandler.receiveQ:
		assert.True(t, myHandler.hadConnected)
		assert.True(t, myHandler.dataReceived)
		assert.NotNil(t, receivedMsg, "Received a valid response")
		assert.Equal(t, TESTSTRING, string(receivedMsg))
	case <-time.After(2 * time.Second):
		t.Fatalf("Can not receive messages from the client")
	}

	// wait for data or timeout
	assert.False(t, myHandler.hadError)
	err = os.RemoveAll(testFTPdirname)
	assert.Nil(t, err, "Please delete ftp test directory:", testFTPdirname)
}

func TestSFTPServerSend(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.Nil(t, err)

	myPath, err := os.Executable()
	assert.Nil(t, err)

	myPath = path.Dir(myPath)
	testFTPdirname := myPath + "/" + "ftptest"
	err = os.Mkdir(testFTPdirname, 0777)
	assert.Nil(t, err)

	sftpserver := CreateSFTPServer(testFTPdirname)
	go sftpserver.RunSimpleSFTPServer(privateKey)

	var myHandler testFtpHandler
	myHandler.receiveQ = make(chan []byte)

	testFTPConfig := DefaultFTPConfig().UserPass("testuser", "tiger").PollInterval(1).DontDeleteAfterRead()
	testFTPConfig.linebreak = LINEBREAK_CRLF
	client, err := CreateSFTPClient(SFTP, "localhost", 2022, testFTPdirname, "*.dat", testFTPConfig)
	assert.Nil(t, err)

	var testMsg [][]byte
	testMsg = append(testMsg, []byte(TESTSTRING))
	count, err := client.Send(testMsg)
	assert.Equal(t, len(TESTSTRING), count)
	assert.Nil(t, err)

	// wait for data or timeout
	assert.False(t, myHandler.hadError)
	// uncomment line below to check your files
	err = os.RemoveAll(testFTPdirname)
	assert.Nil(t, err, "Please delete ftp test directory:", testFTPdirname)
}
