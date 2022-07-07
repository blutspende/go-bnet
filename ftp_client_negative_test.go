package bloodlabnet

/* TODO: Disabled, doesnt work
func Test_FTP_Client_Connect_WithInvalidUser(t *testing.T) {
	ftpClient := getFtpClientWithWrongUser()
	connErr := ftpClient.Connect()

	assert.NotNil(t, connErr)
}

func Test_FTP_Client_Connect_MultipleTimes(t *testing.T) {
	runTest(t, getFtpClient(), func(t *testing.T, ftpClient ConnectionAndSessionInstance) error {
		err := ftpClient.Connect()

		return err
	})
}


func Test_FTP_Client_Disconnect_MultipleTimes(t *testing.T) {
	runTest(t, getFtpClient(), func(t *testing.T, ftpClient ConnectionAndSessionInstance) error {
		if err := ftpClient.Close(); err != nil {
			return err
		}
		if err := ftpClient.Close(); err != nil {
			return err
		}

		if ftpClient.IsAlive() {
			return errors.New("Failed to disconnect")
		}

		return nil
	})
}

func Test_FTP_Client_Download_NotExistingFile(t *testing.T) {
	var ftpClient ConnectionAndSessionInstance
	var err error

	if _, err = ftpClient.Receive(); err == nil {
		assert.Fail(t, "Invalid FTP response")
	}

	assert.Equal(t, "550 Couldn't open the file", err.Error())
}
*/
