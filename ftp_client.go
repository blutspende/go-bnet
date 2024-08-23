package bloodlabnet

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"path/filepath"
	"slices"
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

// 10-180 Seconds is good for a pollInterval depending on the server
func CreateNewFTPClient(hostname string, hostport int,
	username string, password string,
	inputFilePath, inputFilePattern,
	outputFilePath string,
	outputFileExtension string,
	filenameGeneratorFunction FTPFilenameGeneratorFunction,
	processStrategy ProcessStrategy,
	lineBreak string,
	pollInterval time.Duration,
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
		PollInterval:              pollInterval,
		ftpConn:                   nil,
		lineBreaks:                lineBreak,
		mainLoopActive:            false,
		stopRequested:             false,
	}
}

func (ci *ftpConnectionAndSession) Send(data [][]byte) (int, error) {

	if ci.ftpConn == nil {
		err := ci.connectToServer()
		if err != nil {
			return 0, err
		}
	}

	lineBreakBytes := []byte(ci.lineBreaks)
	dataWithLinebreaks := make([]byte, 0)
	for _, datarow := range data {
		dataWithLinebreaks = append(dataWithLinebreaks, datarow...)
		dataWithLinebreaks = append(dataWithLinebreaks, lineBreakBytes...) //TODO: Let a flag toggle wether we need the last lb
	}

	generatedFilename, err := ci.filenameGeneratorFunction(dataWithLinebreaks, ci.outputFileExtension)
	if err != nil {
		return 0, fmt.Errorf("generator function failed - %v", err)
	}

	buffer := bytes.NewReader(dataWithLinebreaks)
	err = ci.ftpConn.Stor(filepath.Join(ci.outputFilePath, generatedFilename), buffer)
	if err != nil {
		log.Println(err)
		return 0, ErrSendFileFailed
	}

	return len(dataWithLinebreaks), nil
}

func (ci *ftpConnectionAndSession) Receive() ([]byte, error) {

	err := ci.ftpConn.ChangeDir(ci.inputFilePath)
	if err != nil {
		return nil, fmt.Errorf("directory not found: '%s' - %v", ci.inputFilePath, err)
	}

	var file *ftp.Entry
outer:
	for {
		var files []*ftp.Entry
		files, err := ci.ftpConn.List("")
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				time.Sleep(time.Second)
				log.Println("Lost connection, trying to reconnect to server...")
				err = ci.connectToServer()
				if err != nil {
					log.Printf("Failed to reconnect (delay 60s now)- %v", err)
					time.Sleep(60 * time.Second)
				}
				continue
			} else {
				return nil, ErrListFilesFailed
			}
		}
		for _, ifile := range files {
			matched, err := filepath.Match(ci.inputFilePattern, ifile.Name)
			if err != nil {
				log.Printf("failed to match file with '%s' skipping - %v", ci.inputFilePattern, err)
				continue
			}
			if matched {
				file = ifile
				break outer
			}
		}
		time.Sleep(ci.PollInterval)
	}

	fileReader, err := ci.ftpConn.Retr(file.Name)
	if err != nil {
		return nil, ErrDownloadFileFailed
	}

	content, err := io.ReadAll(fileReader)
	if err != nil {
		log.Println("Failed to read ", err)
		fileReader.Close()
		return nil, ErrDownloadFileFailed
	}
	fileReader.Close()

	switch ci.processStrategy {
	case PROCESS_STRATEGY_DONOTHING:
	case PROCESS_STRATEGY_DELETE:
		err = ci.ftpConn.Delete(file.Name)
		if err != nil {
			log.Println("Failed to delete file ", err)
			return []byte{}, ErrDeleteFile
		}
	case PROCESS_STRATEGY_MOVE2SAVE:
		lst, err := ci.ftpConn.NameList("")
		if !slices.Contains[[]string, string](lst, "save") {
			err = ci.ftpConn.MakeDir("save")
		}
		//if err != nil {
		//	log.Println("Failed to rename file ", err)
		//	return []byte{}, fmt.Errorf("failed to create folder - %v", err)
		//}

		err = ci.ftpConn.Stor(filepath.Join("save/", file.Name), bytes.NewReader(content))
		if err != nil {
			return []byte{}, fmt.Errorf("failed to move file %s - %v", file.Name, err)
		}

		err = ci.ftpConn.Delete(file.Name)
		if err != nil {
			return []byte{}, fmt.Errorf("failed to rename file %s - %v", file.Name, err)
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
	if err != nil { /* a veto from handler instantly stops the loop (it might be for a good reason) */
		return err
	}

	for ci.mainLoopActive && !ci.stopRequested {
		data, err := ci.Receive()
		if err != nil {
			handler.Error(ci, ErrorReceive, err)
			return err
		}

		err = handler.DataReceived(ci, data, time.Now())
		if err != nil {
			//TODO: what should happen on an error ? mbe restore the file ?
			// At the time of design the use-case wasnt clear, thats why
			// this is only logged
			log.Println("failed to process a transmsission. It is not yet implemented what has to happen in this case. Currently the file had been treated accordingly to strategy and will not be reprocessed")
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
