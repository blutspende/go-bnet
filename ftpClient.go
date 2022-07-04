package bloodlabnet

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type ftpClientInstance struct {
	hostname           string
	port               int
	path               string
	fileMask           string
	user               string
	password           string
	pubKey             string
	readFilePolicy     ReadFilePolicy
	fileNamePattern    FileNameGeneration
	timings            TimingConfiguration
	sshConnection      *ssh.Client
	sftpClient         *sftp.Client
	isRunning          bool
	waitForTermination *sync.WaitGroup
}

func CreateNewFTPClient(hostname string, port int, path, fileMask, user, password, pubKey string,
	fileNamePattern FileNameGeneration,
	readFilePolicy ReadFilePolicy,
	timings TimingConfiguration) ConnectionAndSessionInstance {
	return &ftpClientInstance{
		hostname:           hostname,
		port:               port,
		path:               path,
		fileMask:           fileMask,
		user:               user,
		password:           password,
		pubKey:             pubKey,
		readFilePolicy:     readFilePolicy,
		fileNamePattern:    fileNamePattern,
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
	return 0, nil
}
func (c *ftpClientInstance) Receive() ([]byte, error) {
	return nil, nil
}
func (c *ftpClientInstance) Close() error {
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

	var err error

	sshConfig := &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint
	}

	ftpserver := fmt.Sprintf("%s:%d", c.hostname, c.port)

	c.sshConnection, err = ssh.Dial("tcp", ftpserver, sshConfig)
	if err != nil {
		c.sshConnection = nil
		return err
	}

	c.sftpClient, err = sftp.NewClient(c.sshConnection)
	if err != nil {
		c.sshConnection.Close()
		c.sshConnection = nil
		c.sftpClient = nil
		return err
	}

	return nil
}
