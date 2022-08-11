package bloodlabnet

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

/*
type ftpClientInstance struct {
	hostname    string
	port        int
	path        string
	user        string
	password    string
	hostpath    string
	timings     TCPServerConfiguration
	ftpClient   *ftp.ServerConn
	isConnected bool
}*/

/*
func CreateNewFTPClient(
	hostname string,
	port int,
	path string,
	user string,
	password string,
	timings TCPServerConfiguration,
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

func (c *ftpClientInstance) Send(msg [][]byte) (int, error) {
	for _, line := range msg {
		reader := bytes.NewReader(line)
		if err := c.ftpClient.Stor(c.path, reader); err != nil {
			return 0, err
		}
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
*/

type FTPType int

const FTP FTPType = 1
const SFTP FTPType = 2

type AuthenticationMethod int

const PASSWORD AuthenticationMethod = 1
const PUBKEY AuthenticationMethod = 2

type FTPConfiguration interface {
	UserPass(user, pass string)
	HostKey(hostkey string)
	PubKey(user, pubkey string)
	PollInterval(pollInterval time.Duration)
}

type ftpConfiguration struct {
	authMethod      AuthenticationMethod
	user            string
	key             string
	hostkey         string
	pollInterval    time.Duration
	deleteAfterRead bool
}

type ftpServerInstance struct {
	ftpType  FTPType
	host     string
	port     int
	hostpath string
	filemask string
	config   *ftpConfiguration
}

func CreateFTP(ftptype FTPType, host string, port int, hostpath, filemask string, config *ftpConfiguration) (ConnectionInstance, error) {
	instance := &ftpServerInstance{
		ftpType:  ftptype,
		host:     host,
		port:     port,
		config:   config,
		hostpath: hostpath,
		filemask: filemask,
	}
	return instance, nil
}

func (conf *ftpConfiguration) UserPass(user, pass string) *ftpConfiguration {
	conf.user = user
	conf.key = pass
	conf.authMethod = PASSWORD
	return conf
}

func (conf *ftpConfiguration) HostKey(hostkey string) *ftpConfiguration {
	conf.hostkey = hostkey
	return conf
}

func (conf *ftpConfiguration) PubKey(user, pubkey string) *ftpConfiguration {
	conf.user = user
	conf.key = pubkey
	conf.authMethod = PUBKEY
	return conf
}

func (conf *ftpConfiguration) PollInterval(pollInterval time.Duration) *ftpConfiguration {
	conf.pollInterval = pollInterval
	return conf
}

func (conf *ftpConfiguration) DeleteAfterRead() *ftpConfiguration {
	conf.deleteAfterRead = true
	return conf
}

func (conf *ftpConfiguration) DontDeleteAfterRead() *ftpConfiguration {
	conf.deleteAfterRead = false
	return conf
}

func DefaultFTPConfig() *ftpConfiguration {
	return &ftpConfiguration{
		authMethod:      PASSWORD,
		pollInterval:    60 * time.Second,
		deleteAfterRead: true,
	}
}

func (instance *ftpServerInstance) Run(handler Handler) {
	switch instance.ftpType {
	case FTP:
		instance.runWithFTP(handler)
	case SFTP:
		instance.runWithSFTP(handler)
	default:
		//TODO handler.Error()
	}
}

func (instance *ftpServerInstance) Stop() {

}

func (instance *ftpServerInstance) FindSessionsByIp(ip string) []Session {
	return []Session{}
}

func (instance *ftpServerInstance) runWithSFTP(handler Handler) {

	sshConfig := &ssh.ClientConfig{}

	switch instance.config.authMethod {
	case PASSWORD:
		sshConfig.User = instance.config.user
		sshConfig.Auth = []ssh.AuthMethod{
			ssh.Password(instance.config.key),
		}
	case PUBKEY:
		// TODO not implemented
	default:
		// this should never happen :)
		log.Panic().Msg("Invalid (s)FTP Authentication-Method provided")
	}

	if instance.config.hostkey != "" {
		base64Key := []byte(instance.config.hostkey)
		key := make([]byte, base64.StdEncoding.DecodedLen(len(base64Key)))
		n, err := base64.StdEncoding.Decode(key, base64Key)
		if err != nil {
			//TODO: error
			return
		}
		key = key[:n]
		hostKey, err := ssh.ParsePublicKey(key)
		if err != nil {
			//TODO: error
			return
		}
		sshConfig.HostKeyCallback = ssh.FixedHostKey(hostKey)
	} else {
		sshConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	ftpserver := fmt.Sprintf("%s:%s", instance.host, instance.port)

	sshConnection, err := ssh.Dial("tcp", ftpserver, sshConfig)
	if err != nil {
		log.Error().Err(err).Msg("Open - Dail")
		// TODO LOG Err
		return
	}

	sftpClient, err := sftp.NewClient(sshConnection)
	if err != nil {
		log.Error().Err(err).Msg("Open - NewClient")
		sshConnection.Close()
		// TODO LOG ERROR
		return
	}

	ftpPath := strings.TrimRight(instance.hostpath, "/") + "/"

	for {

		time.Sleep(instance.config.pollInterval)

		files, err := sftpClient.ReadDir(ftpPath)
		if err != nil {
			log.Error().Err(err).Msgf("ReadFiles - ReadDir %s", ftpPath)
			// TODO ERROR
			// return orderFiles
		}

		if len(files) == 0 {
			continue
		}

		for _, file := range files {
			match, err := sftp.Match(strings.ToUpper(instance.filemask), strings.ToUpper(file.Name()))
			if err != nil {
				log.Error().Err(err).Msg("ReadFiles - Match")
				continue
			}
			if match {
				log.Info().Msgf("readsFtpOrders filename: %s ModTime: %+v", file.Name(), file.ModTime())

				filename := fmt.Sprintf("%s%s", ftpPath, file.Name())

				// Rename the file to "in process" before opening it
				filename_processing := filename + ".processing"
				err = sftpClient.Rename(filename, filename_processing)
				if err != nil {
					log.Error().Err(err).Msg(fmt.Sprintf("Skipping file %s", filename_processing))
				}

				// open file and read content
				fileread, err := sftpClient.Open(filename + ".processing")
				if err != nil {
					log.Error().Err(err).Msg("ReadFiles - Open")
					continue
				}
				filebuffer, err := ioutil.ReadAll(fileread)
				if err != nil {
					log.Error().Err(err).Msg("ReadFiles - ReadAll")
					continue
				}

				err = handler.DataReceived(instance, filebuffer, file.ModTime())

				if err == nil && instance.config.deleteAfterRead { // if processing was successful
					err := sftpClient.Remove(filename + ".processing")
					if err != nil {
						log.Error().Err(err).Msg(filename + ".processing")
						// TODO Errorhandling
					}
				}

				err = fileread.Close()
				if err != nil {
					log.Error().Err(err).Msg("ReadFiles - Close")
					continue
				}
			}
		}
	}

	return
}

func (instance *ftpServerInstance) runWithFTP(handler Handler) {
}

func (instance *ftpServerInstance) IsAlive() bool {
	return true // TODO
}

func (instance *ftpServerInstance) Send(msg [][]byte) (int, error) {
	return 0, errors.New("Not implemented")

}

func (instance *ftpServerInstance) Receive() ([]byte, error) {
	return []byte{}, errors.New("Not implemented")
}

func (instance *ftpServerInstance) Close() error {
	return errors.New("Not implemented")
}

func (instance *ftpServerInstance) WaitTermination() error {
	return errors.New("Not implemented")
}

func (instance *ftpServerInstance) RemoteAddress() (string, error) {
	return instance.host, nil
}
