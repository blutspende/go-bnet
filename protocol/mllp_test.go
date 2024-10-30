package protocol

import (
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
)

func TestMLLPProtocolReceive(t *testing.T) {
	host, instrument := net.Pipe()
	instance := Logger(MLLP())
	go func() {
		instrument.Write([]byte("\x0BHello\x0D"))
		instrument.Write([]byte("IsItMeYoureLookingFor\x0D"))
		instrument.Write([]byte("Done\x1C\x0D"))
	}()
	transmission, err := instance.Receive(host)
	assert.Nil(t, err)
	assert.Equal(t, "Hello\x0DIsItMeYoureLookingFor\x0DDone", string(transmission))
}

func TestMLLPProtocolReceiveWithExtraPrecedingCharacters(t *testing.T) {
	host, instrument := net.Pipe()
	instance := Logger(MLLP())
	go func() {
		instrument.Write([]byte("IShouldBeIgnored"))
		instrument.Write([]byte("\x0BHello\x0D"))
		instrument.Write([]byte("IsItMeYoureLookingFor\x0D"))
		instrument.Write([]byte("Done\x1C\x0D"))

	}()
	transmission, err := instance.Receive(host)
	assert.Nil(t, err)
	// without start and end byte
	assert.Equal(t, "Hello\x0DIsItMeYoureLookingFor\x0DDone", string(transmission))
}

func TestMLLPProtocolSend(t *testing.T) {
	host, instrument := net.Pipe()
	instance := Logger(MLLP())
	receivedBytesChan := make(chan []byte)
	go func() {
		buffer := make([]byte, 4096)
		n, err := instrument.Read(buffer)
		assert.Nil(t, err)
		receivedBytesChan <- buffer[:n]
	}()

	instance.Send(host, [][]byte{
		[]byte("First frame"),
		[]byte("Second frame"),
		[]byte("Last frame"),
	})

	receivedBytes := <-receivedBytesChan
	assert.Equal(t, []byte("\x0BFirst frame\x0DSecond frame\x0DLast frame\x0D\x1C\x0D"), receivedBytes)
}
