package bloodlabnet

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TODO: get these out of here
func getTimingConfigurations() TimingConfiguration {
	var DefaultFTPClientSettings = TimingConfiguration{
		Timeout:             time.Second * 3,
		Deadline:            time.Millisecond * 200,
		FlushBufferTimoutMs: 500,
		PollInterval:        time.Second * 60,
	}

	return DefaultFTPClientSettings
}

// TODO: tmp only; use configurations instead of hardcoded user/pwd
func getFtpUsername() string {
	username := "YOUR_FTP_USERNAME"

	if username == "YOUR_FTP_USERNAME" {
		panic("FTP is not configured properly")
	}

	return username
}
func getFtpPassword() string {
	return "YOUR_FTP_PASSWORD"
}
func getFtpServerAddress() string {
	return "127.0.0.1"
}
func getExistingFilename() string {
	return "hello-ftp.txt"
}

func getFtpClientWithWrongUser() ConnectionAndSessionInstance {
	var DefaultFTPClientSettings = getTimingConfigurations()

	ftpClient := CreateNewFTPClient(
		getFtpServerAddress(),
		21,
		getExistingFilename(),
		//"",
		"this-user-is-not-created",
		"its-a-wrong-password",
		//"",
		//Default,
		//ReadAndLeaveFile,
		DefaultFTPClientSettings,
	)

	return ftpClient
}

func getFtpClient() ConnectionAndSessionInstance {
	var DefaultFTPClientSettings = getTimingConfigurations()

	ftpClient := CreateNewFTPClient(
		getFtpServerAddress(),
		21,
		getExistingFilename(),
		//"",
		getFtpUsername(),
		getFtpPassword(),
		//"",
		//Default,
		//ReadAndLeaveFile,
		DefaultFTPClientSettings,
	)

	return ftpClient
}

func getFtpClientWithNotExistingFile() ConnectionAndSessionInstance {
	var DefaultFTPClientSettings = getTimingConfigurations()

	ftpClient := CreateNewFTPClient(
		getFtpServerAddress(),
		21,
		"qwertzuiopasdfghjklyxcvbnm.txt",
		//"",
		getFtpUsername(),
		getFtpPassword(),
		//"",
		//Default,
		//ReadAndLeaveFile,
		DefaultFTPClientSettings,
	)

	return ftpClient
}

func startFTPMockServer() {
	// TODO
}

type testRunner func(t *testing.T, ftpClient ConnectionAndSessionInstance) error

func runTest(t *testing.T, ftpClient ConnectionAndSessionInstance, tr testRunner) {
	connErr := ftpClient.Connect()
	assert.Nil(t, connErr)

	testErr := tr(t, ftpClient)
	assert.Nil(t, testErr)

	dcErr := ftpClient.Close()
	assert.Nil(t, dcErr)
}

func Test_FTP_Client_Connect(t *testing.T) {
	runTest(t, getFtpClient(), func(t *testing.T, ftpClient ConnectionAndSessionInstance) error {
		if connectionIsAlive := ftpClient.IsAlive(); !connectionIsAlive {
			return errors.New("Failed to connect")
		}

		return nil
	})
}

func Test_FTP_Client_Disconnect(t *testing.T) {
	runTest(t, getFtpClient(), func(t *testing.T, ftpClient ConnectionAndSessionInstance) error {
		if err := ftpClient.Close(); err != nil {
			return err
		}

		if ftpClient.IsAlive() {
			return errors.New("Failed to disconnect")
		}

		return nil
	})
}

func Test_FTP_Client_Reconnect(t *testing.T) {
	runTest(t, getFtpClient(), func(t *testing.T, ftpClient ConnectionAndSessionInstance) error {
		var err error

		ftpClient.Close()
		if err = ftpClient.Connect(); err != nil {
			return err
		}

		if !ftpClient.IsAlive() {
			return errors.New("Failed to reconnect")
		}

		return nil
	})
}

func Test_FTP_Client_Open_Directory(t *testing.T) {
	runTest(t, getFtpClient(), func(t *testing.T, ftpClient ConnectionAndSessionInstance) error {
		var err error

		if _, err = ftpClient.Receive(); err != nil {
			return err
		}

		return nil
	})
}

// TODO
/*
func Test_FTP_Client_List_Directory_Content(t *testing.T) {
	runTest(t, getFtpClient(), func(t *testing.T, ftpClient ConnectionAndSessionInstance) error {
		panic("Test is not implemented yet!")

		// return nil
	})
}
*/

func Test_FTP_Client_Upload_File(t *testing.T) {
	runTest(t, getFtpClient(), func(t *testing.T, ftpClient ConnectionAndSessionInstance) error {
		var err error

		fileContent := []byte("Hello FTP")
		if _, err = ftpClient.Send(fileContent); err != nil {
			return err
		}

		return nil
	})
}

func Test_FTP_Client_Download_File(t *testing.T) {
	runTest(t, getFtpClient(), func(t *testing.T, ftpClient ConnectionAndSessionInstance) error {
		var err error
		var fileContent []byte

		newFileContent := []byte("Hello FTP")
		ftpClient.Send(newFileContent)

		if fileContent, err = ftpClient.Receive(); err != nil {
			return err
		}

		assert.NotNil(t, fileContent)
		assert.NotEqual(t, 0, len(fileContent))

		return nil
	})
}
