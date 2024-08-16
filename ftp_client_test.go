package bloodlabnet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	filedriver "github.com/goftp/file-driver"
	"github.com/goftp/server"
)

type testHandler struct {
	receiveEvent chan bool
	dataReceived []byte
	t            *testing.T
}

func (th *testHandler) DataReceived(session Session, data []byte, receiveTimestamp time.Time) error {
	th.dataReceived = data
	th.receiveEvent <- true // inform everyone we received data
	return nil
}
func (th *testHandler) Connected(session Session) error {
	return nil
}
func (th *testHandler) Disconnected(session Session) {}
func (th *testHandler) Error(session Session, typeOfError ErrorType, err error) {
	th.t.Fail()
}

/*
	Receiving Data sourced from an ftp server

Every new file that is dropped in the in folder is treated like a transmission as if was through a socket
Based on the strategy the file is ignored, deleted or moved after processing

Test succeeds when the file arrives and the sourcefile is treated
*/
func TestReceiveFilesFromFtp(t *testing.T) {

	opts := &server.ServerOpts{
		Factory: &filedriver.FileDriverFactory{
			RootPath: "testfilesftp",
			Perm:     server.NewSimplePerm("user", "group"),
		},
		Port:     21,
		Hostname: "127.0.0.1",
		Auth:     &server.SimpleAuth{Name: "test", Password: "test"},
	}

	// Run FTP Server...
	go func() {
		server := server.NewServer(opts)
		err := server.ListenAndServe()
		assert.Nil(t, err)
		t.Fail()
	}()

	// Use bnet to connect
	bnetFtpClient := CreateNewFTPClient("127.0.0.1", 21, "test", "test",
		"TestReceiveFilesFromFtp", "*.dat", "out", ".out", DefaultFTPFilnameGenerator, PROCESS_STRATEGY_DONOTHING, "\n")

	th := &testHandler{
		t:            t,
		receiveEvent: make(chan bool),
	}
	go func() {
		err := bnetFtpClient.Run(th)
		assert.Equal(t, ErrExited, err)
	}()

	select {
	case <-th.receiveEvent:
		// this is the expectation: the file contents are delivered same as through a socket
		assert.Equal(t, "Some orderdata", string(th.dataReceived))
	case <-time.After(5 * time.Second):
		t.Log("Timed out while waiting on receiving data (see test description)")
		t.Fail()
	}
}

func TestReceiveFilesFromFtpStrategyMove2Save(t *testing.T) {

	opts := &server.ServerOpts{
		Factory: &filedriver.FileDriverFactory{
			RootPath: "testfilesftp",
			Perm:     server.NewSimplePerm("user", "group"),
		},
		Port:     21,
		Hostname: "127.0.0.1",
		Auth:     &server.SimpleAuth{Name: "test", Password: "test"},
	}

	// Run FTP Server...
	var ftpserver *server.Server
	go func() {
		ftpserver = server.NewServer(opts)
		err := ftpserver.ListenAndServe()
		assert.Nil(t, err)
		t.Fail()
	}()

	// Use bnet to connect
	bnetFtpClient := CreateNewFTPClient("127.0.0.1", 21, "test", "test",
		"/TestReceiveFilesFromFtpStrategySave", "*.dat",
		"out", ".out",
		DefaultFTPFilnameGenerator, PROCESS_STRATEGY_MOVE2SAVE, "\n")

	th := &testHandler{
		t:            t,
		receiveEvent: make(chan bool),
	}
	go func() {
		err := bnetFtpClient.Run(th)
		assert.Equal(t, ErrExited, err)
	}()

	select {
	case <-th.receiveEvent:
		bnetFtpClient.Stop()
		ftpserver.Shutdown()
		// this is the expectation for this test
		// ... the file contents are delivered same as if they came through a socket
		assert.Equal(t, "Some orderdata", string(th.dataReceived))
	case <-time.After(500 * time.Second):
		bnetFtpClient.Stop()
		ftpserver.Shutdown()
		t.Log("Timed out while waiting on receiving data (see test description)")
		t.Fail()
	}
}
