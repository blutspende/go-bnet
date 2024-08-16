package bloodlabnet

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

var ErrFtpLoginFailed = fmt.Errorf("FTP Login Failed")
var ErrConnectionFailed = fmt.Errorf("connction error")
var ErrListFilesFailed = fmt.Errorf("failed to access files for listing")
var ErrDownloadFileFailed = fmt.Errorf("donwload files failed")
var ErrSendFileFailed = fmt.Errorf("sending file failed")
var ErrDeleteFile = fmt.Errorf("failed to delete file")
var ErrNoSuchDir = fmt.Errorf("no such directory")

type ProcessStrategy string

// after processing do nothing with the file, even though it will get reprocessed on a restart
const PROCESS_STRATEGY_DONOTHING ProcessStrategy = "donothing"

// after processing delete the read file
const PROCESS_STRATEGY_DELETE ProcessStrategy = "delete"

// after processing move the file to save folder
const PROCESS_STRATEGY_MOVE2SAVE ProcessStrategy = "move2save"

type FTPFilenameGeneratorFunction func([]byte, string) (string, error)

// Default generator function
func DefaultFTPFilnameGenerator(data []byte, fileExtension string) (string, error) {
	timestamp := time.Now().Format("20060102_150405000000000")
	filename := fmt.Sprintf("%s.%s", timestamp, strings.Trim(fileExtension, "."))
	return filename, nil
}

type ftpConnectionAndSession struct {
	hostname                  string
	hostport                  int
	username                  string
	password                  string
	processStrategy           ProcessStrategy
	inputFilePath             string
	inputFilePattern          string
	outputFilePath            string
	outputFileExtension       string
	filenameGeneratorFunction FTPFilenameGeneratorFunction
	ftpConn                   *ftp.ServerConn
	PollInterval              time.Duration
	lineBreaks                string
	mainLoopActive            bool
	stopRequested             bool
}

func CreateNewFTPClient(hostname string, hostport int,
	username string, password string,
	inputFilePath, inputFilePattern,
	outputFilePath string,
	outputFileExtension string,
	filenameGeneratorFunction FTPFilenameGeneratorFunction,
	processStrategy ProcessStrategy,
	lineBreak string,
) *ftpConnectionAndSession {

	return &ftpConnectionAndSession{
		hostname:                  hostname,
		hostport:                  hostport,
		username:                  username,
		password:                  password,
		processStrategy:           processStrategy,
		inputFilePath:             inputFilePath,
		inputFilePattern:          inputFilePattern,
		outputFilePath:            outputFilePath,
		outputFileExtension:       outputFileExtension,
		filenameGeneratorFunction: filenameGeneratorFunction,
		PollInterval:              10 * time.Second,
		ftpConn:                   nil,
		lineBreaks:                lineBreak,
		mainLoopActive:            false,
		stopRequested:             false,
	}
}

func (ci *ftpConnectionAndSession) Send(data [][]byte) (int, error) {

	lineBreakBytes := []byte(ci.lineBreaks)
	dataWithLinebreaks := make([]byte, 0)
	for _, datarow := range data {
		dataWithLinebreaks = append(dataWithLinebreaks, datarow...)
		dataWithLinebreaks = append(dataWithLinebreaks, lineBreakBytes...)
	}

	generatedFilename, err := ci.filenameGeneratorFunction(dataWithLinebreaks, ci.outputFileExtension)
	if err != nil {
		return 0, fmt.Errorf("generator funciton failed - %v", err)
	}

	buffer := bytes.NewReader(dataWithLinebreaks)
	err = ci.ftpConn.Stor(filepath.Join(ci.outputFilePath, generatedFilename), buffer)
	if err != nil {
		log.Println(err)
		return 0, ErrSendFileFailed
	}

	return 0, nil
}

func (ci *ftpConnectionAndSession) Receive() ([]byte, error) {

	err := ci.ftpConn.ChangeDir(ci.inputFilePath)
	if err != nil {
		return nil, ErrNoSuchDir
	}

	var file *ftp.Entry

	for {
		var files []*ftp.Entry
		files, err := ci.ftpConn.List(ci.inputFilePattern)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// When the error was a read/timout, then
				// just reconnect
				// TODO: Limt amout of reconnection
				time.Sleep(1 * time.Second)
				log.Println("Lost connection, trying to reconnect to server...")
				ci.connectToServer()
				continue
			} else {
				return nil, ErrListFilesFailed
			}
		}
		if len(files) > 0 {
			file = files[0]
			break
		}
		time.Sleep(ci.PollInterval)
	}

	fileReader, err := ci.ftpConn.Retr(file.Name)
	if err != nil {
		return nil, ErrDownloadFileFailed
	}
	defer fileReader.Close()

	content, err := io.ReadAll(fileReader)
	if err != nil {
		log.Println("Failed to read ", err)
		return nil, ErrDownloadFileFailed
	}

	switch ci.processStrategy {
	case PROCESS_STRATEGY_DONOTHING:
	case PROCESS_STRATEGY_DELETE:
		err = ci.ftpConn.Delete(file.Name)
		if err != nil {
			log.Println("Failed to delete file ", err)
			return []byte{}, ErrDeleteFile
		}
	case PROCESS_STRATEGY_MOVE2SAVE:
		err = ci.ftpConn.Rename(file.Name, filepath.Join("save", file.Name))
		if err != nil {
			log.Println("Failed to rename file ", err)
			return []byte{}, ErrDownloadFileFailed
		}
	}

	return content, nil
}

func (ci *ftpConnectionAndSession) Run(handler Handler) error {
	var err error
	ci.mainLoopActive = true
	ci.stopRequested = false

	err = ci.connectToServer()
	if err != nil {
		return ErrConnectionFailed
	}

	err = handler.Connected(ci)
	if err != nil {
		ci.stopRequested = true
	}

	for ci.mainLoopActive && !ci.stopRequested {
		data, err := ci.Receive()
		//TODO Reconnect on timeout

		switch err {
		case nil:
			handler.DataReceived(ci, data, time.Now())
		case ErrListFilesFailed:
			handler.Error(ci, ErrorReceive, err)
		case ErrDownloadFileFailed:
			handler.Error(ci, ErrorReceive, err)
		case ErrSendFileFailed:
			handler.Error(ci, ErrorReceive, err)
		default:
			handler.Error(ci, ErrorReceive, err)
		}

		time.Sleep(ci.PollInterval * time.Second)
	}

	ci.mainLoopActive = false

	handler.Disconnected(ci)

	return nil
}

func (ci *ftpConnectionAndSession) connectToServer() error {
	var err error
	ftpURL := fmt.Sprintf("%s:%d", ci.hostname, ci.hostport)
	ci.ftpConn, err = ftp.Dial(ftpURL, ftp.DialWithTimeout(10*time.Second))
	if err != nil {
		return ErrConnectionFailed
	}
	err = ci.ftpConn.Login(ci.username, ci.password)
	if err != nil {
		return ErrFtpLoginFailed
	}
	return nil
}

func (ci *ftpConnectionAndSession) Stop() {
	ci.stopRequested = true
}

// With FTP/server this function always returns its own instance
func (ci *ftpConnectionAndSession) FindSessionsByIp(ip string) []Session {
	return []Session{ci}
}

func (ci *ftpConnectionAndSession) WaitReady() bool {
	return true
}

func (ci *ftpConnectionAndSession) IsAlive() bool {
	return ci.mainLoopActive
}

func (ci *ftpConnectionAndSession) Close() error {
	ci.Stop()
	ci.WaitTermination()
	return ci.ftpConn.Quit()
}

func (ci *ftpConnectionAndSession) WaitTermination() error {
	return nil
}

func (ci *ftpConnectionAndSession) RemoteAddress() (string, error) {
	return fmt.Sprintf("%s:%d", ci.hostname, ci.hostport), nil
}
