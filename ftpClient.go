package bloodlabnet

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/jlaffaye/ftp"
)

type ftpClientInstance struct {
	hostname    string
	port        int
	path        string
	user        string
	password    string
	timings     TimingConfiguration
	ftpClient   *ftp.ServerConn
	isConnected bool
}

func CreateNewFTPClient(
	hostname string,
	port int,
	path string,
	user string,
	password string,
	timings TimingConfiguration,
	secureConnectionOptions ...SecureConnectionOptions) ConnectionAndSessionInstance {
	if secureConnectionOptions != nil {
		panic("SFTP client is not implemented yet!")
	}

	return &ftpClientInstance{
		hostname: hostname,
		port:     port,
		path:     path,
		user:     user,
		password: password,
		timings:  timings,
	}
}

func (c *ftpClientInstance) Run(handler Handler) {
	panic("Run is not implemented yet!")
}

func (c *ftpClientInstance) Stop() {
	panic("Stop is not implemented yet!")
}

func (instance *ftpClientInstance) FindSessionsByIp(ip string) []Session {
	panic("FindSessionsByIp is not implemented yet!")
}

// ---------- Session Methods starting here

func (c *ftpClientInstance) IsAlive() bool {
	return c.isConnected
}

func (c *ftpClientInstance) Send(msg []byte) (int, error) {
	reader := bytes.NewReader(msg)
	if err := c.ftpClient.Stor(c.path, reader); err != nil {
		return 0, err
	}

	return 0, nil
}

func (c *ftpClientInstance) Receive() ([]byte, error) {
	var ftpResponse *ftp.Response
	var err error

	if ftpResponse, err = c.ftpClient.Retr(c.path); err != nil {
		return []byte{}, err
	}

	var fileContent []byte
	if fileContent, err = ioutil.ReadAll(ftpResponse); err != nil {
		return []byte{}, err
	}

	return fileContent, nil
}

func (c *ftpClientInstance) Close() error {
	if !c.isConnected {
		return nil
	}

	if err := c.ftpClient.Quit(); err != nil {
		return err
	}

	c.isConnected = false

	return nil
}

func (c *ftpClientInstance) WaitTermination() error {
	panic("WaitTermination is not implemented yet!")
}

func (c *ftpClientInstance) RemoteAddress() (string, error) {
	panic("RemoteAddress is not implemented yet!")
}

func (c *ftpClientInstance) Connect() error {
	if c.isConnected {
		return nil
	}

	var err error
	var serverConnection *ftp.ServerConn

	ftpServerUrl := fmt.Sprintf("%s:%d", c.hostname, c.port)
	serverConnection, err = ftp.Dial(ftpServerUrl, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return err
	}

	if err = serverConnection.Login(c.user, c.password); err != nil {
		return err
	}

	c.ftpClient = serverConnection
	c.isConnected = true

	return nil
}
