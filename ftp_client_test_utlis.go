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

// TODO: Remove these when FTP server is implemented
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
		"this-user-is-not-created",
		"its-a-wrong-password",
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
		getFtpUsername(),
		getFtpPassword(),
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
		getFtpUsername(),
		getFtpPassword(),
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
