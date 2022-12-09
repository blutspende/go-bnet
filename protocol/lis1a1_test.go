package protocol

import (
	"fmt"
	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestComputeChecksum(t *testing.T) {
	message := "This is transmission text for which we need a checksum"
	frameNumber := "1"
	specialChars := []byte{utilities.ETX}

	expectedChecksum := []byte(fmt.Sprintf("%02X", 61))
	checksum := computeChecksum([]byte(frameNumber), []byte(message), specialChars)

	assert.Equal(t, expectedChecksum, checksum)

	frameNumber = "2"
	checksum = computeChecksum([]byte(frameNumber), []byte(message), specialChars)
	assert.NotEqual(t, expectedChecksum, checksum)
}

func TestSendData(t *testing.T) {

	// this is like the instrument would behave
	var mc mockConnection
	mc.scriptedProtocol = make([]scriptedProtocol, 0)
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.ENQ}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "tx", bytes: []byte{utilities.ACK}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.STX}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte([]byte("1H||||"))})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.ETX}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{54, 67}}) // checksum
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{13, 10}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "tx", bytes: []byte{utilities.ACK}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.STX}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte([]byte("2O|1|||||"))})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.ETX}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{57, 68}}) // checksum
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{13, 10}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "tx", bytes: []byte{utilities.ACK}})
	mc.scriptedProtocol = append(mc.scriptedProtocol, scriptedProtocol{receiveOrSend: "rx", bytes: []byte{utilities.EOT}})

	mc.currentRecord = 0

	// this is "our" sid of the protocol
	message := [][]byte{}
	message = append(message, []byte("H||||"))
	message = append(message, []byte("O|1|||||"))

	os.Setenv("PROTOLOG_ENABLE", "true") // enable logging
	instance := Logger(Lis1A1Protocol(DefaultLis1A1ProtocolSettings()))

	_, err := instance.Send(&mc, message)

	assert.Nil(t, err)
}
