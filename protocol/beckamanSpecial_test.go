package protocol

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
	"github.com/stretchr/testify/assert"
)

func TestOneMessageRequestResponse(t *testing.T) {

	host, instrument := net.Pipe()

	os.Setenv("PROTOLOG_ENABLE", "extended") // enable logging

	go func() { // This is the instrument (you must become the instrument yourself to read it)
		const expectedLatency_inMs_TimesTwo = 40 // ms
		bufferack := make([]byte, 1)

		_, err := instrument.Write([]byte{utilities.STX}) // no bcc)
		assert.Nil(t, err)
		start := time.Now()
		_, err = instrument.Read(bufferack)
		assert.Nil(t, err)
		duration := time.Now().Sub(start).Milliseconds()
		assert.Equal(t, []byte{utilities.ACK}, bufferack)
		assert.LessOrEqual(t, int64(500), duration)
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), duration)

		_, err = instrument.Write([]byte{'R', 'B', '0', '3', utilities.LF}) // no bcc)
		assert.Nil(t, err)
		//++ expect ACK
		start = time.Now()
		_, err = instrument.Read(bufferack)
		assert.Nil(t, err)
		duration = time.Now().Sub(start).Milliseconds()
		assert.Equal(t, []byte{utilities.ACK}, bufferack)
		assert.LessOrEqual(t, int64(500), duration)
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), duration)

		time.Sleep(2000*time.Millisecond + expectedLatency_inMs_TimesTwo)

		_, err = instrument.Write([]byte{utilities.STX}) // no bcc)
		assert.Nil(t, err)
		start = time.Now()
		_, err = instrument.Read(bufferack)
		assert.Nil(t, err)
		duration = time.Now().Sub(start).Milliseconds()
		assert.Equal(t, []byte{utilities.ACK}, bufferack)
		assert.LessOrEqual(t, int64(500), duration)
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), duration)

		instrument.Write([]byte{'R', ' ', '0', '3', /*instrument#*/
			'1', '1', '1', '1', /* RACK# */
			'0', '1', /*CUP*/
			' ',                /* Sample Type */
			'0', '0', '1', '6', /*SAMPLENO*/
			'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', /* SAMPLEID */
			utilities.LF}) // bo bcc
		//++ expect ACK
		start = time.Now()
		instrument.Read(bufferack)
		assert.Nil(t, err)
		duration = time.Since(start).Milliseconds()
		assert.Equal(t, []byte{utilities.ACK}, bufferack)
		assert.LessOrEqual(t, int64(500), duration)
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), duration)
		assert.Equal(t, []byte{utilities.ACK}, bufferack)

		sBeginMessage := make([]byte, 1) // including STX and 0A in the transmission
		instrument.Read(sBeginMessage)
		assert.Equal(t, []byte{utilities.STX}, sBeginMessage)
		assert.LessOrEqual(t, int64(522-expectedLatency_inMs_TimesTwo), duration) // NOT sure here maybe other timings
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), duration)
		// wait for 0.5 secs
		time.Sleep(500*time.Millisecond + expectedLatency_inMs_TimesTwo)
		// Send ACK
		_, err = instrument.Write([]byte{utilities.ACK})
		assert.Nil(t, err)

		sMessage := make([]byte, 31) // including STX and 0A in the transmission
		instrument.Read(sMessage)
		duration = time.Since(start).Milliseconds() // time from last ACK, then the stranmissiontime of SMESSAGE (522 ms)
		assert.Equal(t, []byte("S 34567890123456789012345678901"), sMessage)
		assert.LessOrEqual(t, int64(522-expectedLatency_inMs_TimesTwo), duration)
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), duration)

		// wait for 0.5 secs
		time.Sleep(500*time.Millisecond + expectedLatency_inMs_TimesTwo)
		// Send ACK
		_, err = instrument.Write([]byte{utilities.ACK})
		assert.Nil(t, err)

		// Wait T5 < x
		time.Sleep(2000*time.Millisecond + expectedLatency_inMs_TimesTwo)

		_, err = instrument.Write([]byte{utilities.STX, 'R', 'E', '0', '3', utilities.LF})
		assert.Nil(t, err)
		//++ expect ACK
		start = time.Now()
		_, err = instrument.Read(bufferack)
		assert.Nil(t, err)
		duration = time.Since(start).Milliseconds()
		assert.Equal(t, []byte{utilities.ACK}, bufferack)
		assert.LessOrEqual(t, int64(500), duration)
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), duration)
		assert.Equal(t, []byte{utilities.ACK}, bufferack)
	}()

	// from here on we become the host :) - (thats ourselfes)
	instance := Logger(BeckmanSpecialProtocol(DefaultBeckmanSpecialProtocolSettings().
		SetStartByte(0x02).
		SetEndByte(0x0A).
		SetLineBreakByte(0x0A).SetAcknowledgementTimeout(500 * time.Millisecond)))

	//rbmessage, err := instance.Receive(host)
	//assert.NotNil(t, err)
	//assert.Equal(t, "RB03", string(rbmessage))

	r1message, err := instance.Receive(host)
	assert.Nil(t, err)
	assert.Equal(t, "R 03111101 00160123456789", string(r1message))

	str := "S 34567890123456789012345678901" // 31 bytes (content dont care :)
	_, err = instance.Send(host, [][]byte{[]byte(str)})
	assert.Nil(t, err)

	remessage, err := instance.Receive(host)
	assert.Nil(t, err)
	assert.Equal(t, "RE03", string(remessage))
}

func TestMultipleMessageRequestResponse(t *testing.T) {
}

func TestMultipleMessageRequestResponseIncludingARetryOnSimulatedFail(t *testing.T) {
	// TODO
}
