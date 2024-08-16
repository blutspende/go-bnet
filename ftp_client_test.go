package bloodlabnet

import (
	"testing"
	"time"

	ftpserver "github.com/fclairamb/ftpserverlib"
	gkwrap "github.com/fclairamb/go-log/gokit"
	"github.com/stretchr/testify/assert"

	"github.com/fclairamb/ftpserver/config"
	"github.com/fclairamb/ftpserver/config/confpar"
	"github.com/fclairamb/ftpserver/server"
)

type testHandler struct {
	dataReceived []byte
	t            *testing.T
}

func (th *testHandler) DataReceived(session Session, data []byte, receiveTimestamp time.Time) error {
	th.dataReceived = data
	return nil
}
func (th *testHandler) Connected(session Session) error {
	return nil
}
func (th *testHandler) Disconnected(session Session) {}
func (th *testHandler) Error(session Session, typeOfError ErrorType, err error) {
	th.t.Fail()
}

func TestVanillaConnect(t *testing.T) {

	var ftpServer *ftpserver.FtpServer
	var driver *server.Server

	logger := gkwrap.New()

	conf := &config.Config{
		Content: &confpar.Content{
			ListenAddress: ":21",
			PublicHost:    "127.0.0.1",
			Accesses: []*confpar.Access{{
				User: "test",
				Pass: "test",
				Fs:   "os",
				Params: map[string]string{
					"basePath": "tmp",
				},
			},
			},
			PassiveTransferPortRange: &confpar.PortRange{
				Start: 30000,
				End:   35000,
			},
		},
	}

	driver, errNewServer := server.NewServer(conf, logger.With("component", "driver"))
	assert.Nil(t, errNewServer)

	go func() {
		ftpServer = ftpserver.NewFtpServer(driver)
		err := ftpServer.ListenAndServe()
		assert.Nil(t, err)
	}()

	// Here is the actual test...
	bnetFtpClient := CreateNewFTPClient("127.0.0.1", 21, "test", "test",
		"/in", "*.dat", "out", ".out", DefaultFTPFilnameGenerator, PROCESS_STRATEGY_DONOTHING, "\n")

	th := &testHandler{
		t: t,
	}
	outcome := bnetFtpClient.Run(th)
	assert.Nil(t, outcome)

	err := driver.WaitGracefully(time.Second * 5)
	assert.Nil(t, err)
}
