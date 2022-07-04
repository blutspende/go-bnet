package bloodlabnet

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/jlaffaye/ftp"
	//"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type ftpClientInstance struct {
	hostname string
	port     int
	path     string
	//fileMask           string
	user     string
	password string
	//pubKey             string
	//readFilePolicy     ReadFilePolicy
	//fileNamePattern    FileNameGeneration
	timings       TimingConfiguration
	sshConnection *ssh.Client
	//sftpClient         *sftp.Client
	ftpClient          *ftp.ServerConn
	isRunning          bool
	waitForTermination *sync.WaitGroup
}

func CreateNewFTPClient(
	hostname string,
	port int,
	path string,
	//fileMask string,
	user string,
	password string,
	//fileNamePattern FileNameGeneration,
	//readFilePolicy ReadFilePolicy,
	timings TimingConfiguration,
	secureConnectionOptions ...SecureConnectionOptions) ConnectionAndSessionInstance {
	if secureConnectionOptions != nil {
		panic("SFTP client is not implemented yet!")
	}

	return &ftpClientInstance{
		hostname: hostname,
		port:     port,
		path:     path,
		//fileMask:           fileMask,
		user:     user,
		password: password,
		//pubKey:             secureConnectionOptions.PublicKey,
		//readFilePolicy:     readFilePolicy,
		//fileNamePattern:    fileNamePattern,
		timings:            timings,
		waitForTermination: &sync.WaitGroup{},
	}
}

func (c *ftpClientInstance) Run(handler Handler) {
	c.waitForTermination.Add(1)
	c.isRunning = true

	for c.isRunning {
		time.Sleep(c.timings.PollInterval)
	}

	c.waitForTermination.Done()
}

func (c *ftpClientInstance) Stop() {
	c.isRunning = false
	c.waitForTermination.Wait()
}

func (instance *ftpClientInstance) FindSessionsByIp(ip string) []Session {
	sessions := make([]Session, 0)

	// todo: same as tcpclient, but wasnt done at the time of creation

	return sessions
}

// ---------- Session Methods starting here

func (c *ftpClientInstance) IsAlive() bool {
	return c.isRunning
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
	if !c.isRunning {
		return nil
	}

	if err := c.ftpClient.Quit(); err != nil {
		return err
	}

	c.isRunning = false

	return nil
}

func (c *ftpClientInstance) WaitTermination() error {
	return nil
}

func (c *ftpClientInstance) RemoteAddress() (string, error) {
	if c.sshConnection != nil {
		host, _, err := net.SplitHostPort(c.sshConnection.Conn.RemoteAddr().String())
		if err != nil {
			return host, err
		}
		return host, err
	} else {
		return "", errors.New("no ftp connection")
	}
}

func (c *ftpClientInstance) Connect() error {
	if c.isRunning {
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
	c.isRunning = true

	return nil
}
