package bloodlabnet

import (
	"os"
	"sync"
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
	th.t.Log(err)
	th.t.Fail()
}

func TestReceiveFilesFromFtpStrategyDoNothing(t *testing.T) {

	// prerequesite create testdir and a testorder
	os.Mkdir(".testfilesftp", 0755)
	TESTDIR := t.Name()
	err := os.RemoveAll(".testfilesftp/" + TESTDIR)
	assert.Nil(t, err)
	err = os.Mkdir(".testfilesftp/"+TESTDIR, 0755)
	assert.Nil(t, err)
	TESTORDERCONTENT := "Content of an Arbitrary"
	os.WriteFile(".testfilesftp/"+TESTDIR+"/order.dat", []byte(TESTORDERCONTENT), 0644)

	ftpServer := server.NewServer(&server.ServerOpts{
		Factory: &filedriver.FileDriverFactory{
			RootPath: ".testfilesftp",
			Perm:     server.NewSimplePerm("user", "group"),
		},
		Port:     21,
		Hostname: "127.0.0.1",
		Auth:     &server.SimpleAuth{Name: "test", Password: "test"},
	})
	defer ftpServer.Shutdown()
	// Run FTP Server...
	go func() {
		err := ftpServer.ListenAndServe()
		if err == server.ErrServerClosed {
			return
		}
		assert.Nil(t, err)
		t.Fail()
	}()
	time.Sleep(3 * time.Second)

	// Use bnet to connect
	bnetFtpClient := CreateNewFTPClient("127.0.0.1", 21, "test", "test",
		TESTDIR, "*.dat", "out", ".out",
		DefaultFTPFilnameGenerator,
		PROCESS_STRATEGY_DONOTHING, "\n", 10*time.Second)

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
		assert.Equal(t, TESTORDERCONTENT, string(th.dataReceived))
	case <-time.After(5 * time.Second):
		t.Log("Timed out while waiting on receiving data (see test description)")
		t.Fail()
	}
}

func TestReceiveFilesFromFtpStrategyMove2Save(t *testing.T) {

	// prerequesite> create testdir and a testorder
	os.Mkdir(".testfilesftp", 0755)
	TESTDIR := t.Name()
	err := os.RemoveAll(".testfilesftp/" + TESTDIR)
	assert.Nil(t, err)
	err = os.Mkdir(".testfilesftp/"+TESTDIR, 0755)
	assert.Nil(t, err)
	TESTORDERCONTENT := "Content of an Arbitrary"
	os.WriteFile(".testfilesftp/"+TESTDIR+"/order.dat", []byte(TESTORDERCONTENT), 0644)

	// Run FTP Server...
	var ftpserver *server.Server
	waitStartup := sync.Mutex{}
	waitStartup.Lock()
	go func() {
		ftpserver = server.NewServer(&server.ServerOpts{
			Factory: &filedriver.FileDriverFactory{
				RootPath: ".testfilesftp",
				Perm:     server.NewSimplePerm("user", "group"),
			},
			Port:     21,
			Hostname: "127.0.0.1",
			Auth:     &server.SimpleAuth{Name: "test", Password: "test"},
		})
		waitStartup.Unlock()
		ftpserver.ListenAndServe()
	}()
	waitStartup.Lock()

	// Use bnet to connect
	bnetFtpClient := CreateNewFTPClient("127.0.0.1", 21, "test", "test",
		TESTDIR, "*.dat",
		"out", ".out",
		DefaultFTPFilnameGenerator, PROCESS_STRATEGY_MOVE2SAVE, "\n",
		10*time.Second)

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
		assert.Equal(t, TESTORDERCONTENT, string(th.dataReceived))

		// The file must have been moved to save
		_, err := os.Stat(".testfilesftp/" + TESTDIR + "/order.dat")
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(".testfilesftp/" + TESTDIR + "/save/order.dat")
		assert.Nil(t, err)
	case <-time.After(5 * time.Second):
		bnetFtpClient.Stop()
		ftpserver.Shutdown()
		t.Log("Timed out while waiting on receiving data (see test description)")
		t.Fail()
	}
}

func TestReceiveFilesFromFtpStrategyDelete(t *testing.T) {

	// prerequesite: create testdir and a testorder
	os.Mkdir(".testfilesftp", 0755)
	TESTDIR := t.Name()
	err := os.RemoveAll(".testfilesftp/" + TESTDIR)
	assert.Nil(t, err)
	err = os.Mkdir(".testfilesftp/"+TESTDIR, 0755)
	assert.Nil(t, err)
	TESTORDERCONTENT := "Content of an Arbitrary"
	os.WriteFile(".testfilesftp/"+TESTDIR+"/order.dat", []byte(TESTORDERCONTENT), 0644)

	// Run FTP Server...
	var ftpserver *server.Server
	waitStartup := sync.Mutex{}
	waitStartup.Lock()
	go func() {
		ftpserver = server.NewServer(&server.ServerOpts{
			Factory: &filedriver.FileDriverFactory{
				RootPath: ".testfilesftp",
				Perm:     server.NewSimplePerm("user", "group"),
			},
			Port:     21,
			Hostname: "127.0.0.1",
			Auth:     &server.SimpleAuth{Name: "test", Password: "test"},
		})
		waitStartup.Unlock()
		ftpserver.ListenAndServe()
	}()
	waitStartup.Lock()

	// Use bnet to connect
	bnetFtpClient := CreateNewFTPClient("127.0.0.1", 21, "test", "test",
		TESTDIR, "*.dat",
		"out", ".out",
		DefaultFTPFilnameGenerator, PROCESS_STRATEGY_DELETE, "\n",
		10*time.Second)

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
		assert.Equal(t, TESTORDERCONTENT, string(th.dataReceived))

		// The file had been deleted
		_, err := os.Stat(".testfilesftp/" + TESTDIR + "/order.dat")
		assert.True(t, os.IsNotExist(err))
	case <-time.After(5 * time.Second):
		bnetFtpClient.Stop()
		ftpserver.Shutdown()
		t.Log("Timed out while waiting on receiving data (see test description)")
		t.Fail()
	}
}

func TestSubmitFile(t *testing.T) {

	// prerequesite: create testdir and a testorder
	os.Mkdir(".testfilesftp", 0755)
	TESTDIR := t.Name()
	err := os.RemoveAll(".testfilesftp/" + TESTDIR)
	assert.Nil(t, err)
	err = os.Mkdir(".testfilesftp/"+TESTDIR, 0755)
	assert.Nil(t, err)

	// Run FTP Server...
	var ftpserver *server.Server
	waitStartup := sync.Mutex{}
	waitStartup.Lock()
	go func() {
		ftpserver = server.NewServer(&server.ServerOpts{
			Factory: &filedriver.FileDriverFactory{
				RootPath: ".testfilesftp",
				Perm:     server.NewSimplePerm("user", "group"),
			},
			Port:     2121,
			Hostname: "127.0.0.1",
			Auth:     &server.SimpleAuth{Name: "test", Password: "test"},
		})
		waitStartup.Unlock()
		ftpserver.ListenAndServe()
	}()
	waitStartup.Lock()

	// Use bnet to connect
	bnetFtpClient := CreateNewFTPClient("127.0.0.1", 2121, "test", "test",
		TESTDIR, "*.dat",
		TESTDIR, ".out",
		func([]byte, string) (string, error) { return "testfile.dat", nil },
		PROCESS_STRATEGY_DELETE, "\r\n",
		10*time.Second)

	TESTLINE1 := "A filecontent which will be split"
	TESTLINE2 := "in two lines. bnet adds the linebreaks as they should"
	nBbytes, err := bnetFtpClient.Send([][]byte{[]byte(TESTLINE1), []byte(TESTLINE2)})
	assert.Equal(t, 90, nBbytes)
	assert.Nil(t, err)

	filedata, err := os.ReadFile(".testfilesftp/" + TESTDIR + "/testfile.dat")
	assert.Nil(t, err)
	assert.Equal(t, TESTLINE1+"\r\n"+TESTLINE2+"\r\n", string(filedata))
}

func TestDealWithServerTimeout(t *testing.T) {
	// TODO: Given the crappy ftp Server implementation in go
	// there is no way to lower the settings for timeout
}
