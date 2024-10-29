package protocol

import (
	"github.com/blutspende/go-bnet/protocol/utilities"
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
		instrument.Write([]byte("Done\x0D\x1C"))
	}()
	transmission, err := instance.Receive(host)
	assert.Nil(t, err)
	// without start and end byte
	assert.Equal(t, "Hello\x0DIsItMeYoureLookingFor\x0DDone\x0D", string(transmission))
}

func TestMLLPProtocolReceiveMultipleStartEndBytes(t *testing.T) {
	host, instrument := net.Pipe()
	settings := DefaultMLLPProtocolSettings().SetStartBytes([]byte{utilities.VT, utilities.CR}).SetEndBytes([]byte{utilities.FS, utilities.CR})
	instance := Logger(MLLP(settings))
	go func() {
		instrument.Write([]byte("\x0B\x0DHello\x0D"))
		instrument.Write([]byte("IsItMeYoureLookingFor\x0D"))
		instrument.Write([]byte("Done\x1C\x0D"))

	}()
	transmission, err := instance.Receive(host)
	assert.Nil(t, err)
	// without start and end byte
	assert.Equal(t, "Hello\x0DIsItMeYoureLookingFor\x0DDone", string(transmission))
}

func TestMLLPProtocolReceiveMultipleStartEndBytesWithExtraPrecedingCharacters(t *testing.T) {
	host, instrument := net.Pipe()
	settings := DefaultMLLPProtocolSettings().SetStartBytes([]byte{utilities.VT, utilities.CR}).SetEndBytes([]byte{utilities.FS, utilities.CR})
	instance := Logger(MLLP(settings))
	go func() {
		instrument.Write([]byte("IShouldBeIgnored"))
		instrument.Write([]byte("\x0B\x0DHello\x0D"))
		instrument.Write([]byte("IsItMeYoureLookingFor\x0D"))
		instrument.Write([]byte("Done\x1C\x0D"))

	}()
	transmission, err := instance.Receive(host)
	assert.Nil(t, err)
	// without start and end byte
	assert.Equal(t, "Hello\x0DIsItMeYoureLookingFor\x0DDone", string(transmission))
}
