package bloodlabnet

/*
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
*/
