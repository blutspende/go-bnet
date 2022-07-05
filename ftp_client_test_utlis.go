package bloodlabnet

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func getTimingConfigurations() TimingConfiguration {
	var DefaultFTPClientSettings = TimingConfiguration{
		Timeout:             time.Second * 3,
		Deadline:            time.Millisecond * 200,
		FlushBufferTimoutMs: 500,
		PollInterval:        time.Second * 60,
	}

	return DefaultFTPClientSettings
}

func getFtpUsername() string {
	username := "Arcane"

	if username == "YOUR_FTP_USERNAME" {
		panic("FTP is not configured properly")
	}

	return username
}
func getFtpPassword() string {
	return "shgxcer"
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
