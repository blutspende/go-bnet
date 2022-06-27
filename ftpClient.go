package bloodlabnet

import (
	"errors"
)

type ftpClientInstance struct {
	hostname        string
	port            int
	path            string
	fileMask        string
	user            string
	password        string
	pubKey          string
	readFilePolicy  ReadFilePolicy
	fileNamePattern FileNameGeneration
}

func CreateNewFTPClient(hostname string, port int, path, fileMask, user, password, pubKey string,
	fileNamePattern FileNameGeneration,
	readFilePolicy ReadFilePolicy) ConnectionInstance {
	return &ftpClientInstance{
		hostname:        hostname,
		port:            port,
		path:            path,
		fileMask:        fileMask,
		user:            user,
		password:        password,
		pubKey:          pubKey,
		readFilePolicy:  readFilePolicy,
		fileNamePattern: fileNamePattern,
	}
}

func (c *ftpClientInstance) Stop() {}

func (c *ftpClientInstance) Receive() ([]byte, error) {
	return nil, errors.New("Not implemented")
}

func (c *ftpClientInstance) Run(handler Handler) {

}

func (c ftpClientInstance) Send(msg []byte) (int, error) {
	return 0, nil
}
