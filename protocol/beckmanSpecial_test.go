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
		buffer_1Byte := make([]byte, 1)          // including STX and 0A in the transmission
		buffer_32Bytes := make([]byte, 32)       // including STX and 0A in the transmission

		//-- send STX
		_, err := instrument.Write([]byte{utilities.STX}) // no bcc)
		FirstSTXSentTime := time.Now()
		assert.Nil(t, err)

		//-- expect ACK
		_, err = instrument.Read(buffer_1Byte)
		FirstACKReceiveTime := time.Now()
		assert.Nil(t, err)
		assert.Equal(t, []byte{utilities.ACK}, buffer_1Byte)
		assert.LessOrEqual(t, int64(500), FirstACKReceiveTime.Sub(FirstSTXSentTime).Milliseconds())
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), FirstACKReceiveTime.Sub(FirstSTXSentTime).Milliseconds())

		//-- Send RB03 (Start of Request block)
		_, err = instrument.Write([]byte{'R', 'B', '0', '3', utilities.LF}) // no bcc)
		timeOf_RequestBlock := time.Now()
		assert.Nil(t, err)

		//-- expect ACK (for the RB message)
		_, err = instrument.Read(buffer_1Byte)
		timeOf_2ndAck := time.Now()
		assert.Nil(t, err)
		assert.Equal(t, []byte{utilities.ACK}, buffer_1Byte)
		assert.LessOrEqual(t, int64(500), timeOf_2ndAck.Sub(timeOf_RequestBlock).Milliseconds())
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), timeOf_2ndAck.Sub(timeOf_RequestBlock).Milliseconds())

		//-- Do not send first request before 2 seconds (T5 in manual page 27)
		time.Sleep(2000*time.Millisecond + expectedLatency_inMs_TimesTwo)

		//-- send STX (to start the Request)
		_, err = instrument.Write([]byte{utilities.STX}) // no bcc)
		timeOf_FirstRequestSTX := time.Now()
		assert.Nil(t, err)

		//-- expect ACK (for the start of request)
		_, err = instrument.Read(buffer_1Byte)
		timeOf_3rdAck := time.Now()
		assert.Nil(t, err)
		assert.Equal(t, []byte{utilities.ACK}, buffer_1Byte)
		assert.LessOrEqual(t, int64(500), timeOf_3rdAck.Sub(timeOf_FirstRequestSTX).Milliseconds())
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), timeOf_3rdAck.Sub(timeOf_FirstRequestSTX).Milliseconds())

		//-- Send R_ Request Message for instrument
		instrument.Write([]byte{'R', ' ', '0', '3', /*instrument#*/
			'1', '1', '1', '1', /* RACK# */
			'0', '1', /*CUP*/
			' ',                /* Sample Type */
			'0', '0', '1', '6', /*SAMPLENO*/
			'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', /* SAMPLEID */
			utilities.LF}) // bo bcc
		timeOf_FirstRequest := time.Now()

		//-- expect ACK (for the Reuqst message)
		instrument.Read(buffer_1Byte)
		timeOf_4thAck := time.Now()
		assert.Nil(t, err)
		assert.LessOrEqual(t, int64(500), timeOf_4thAck.Sub(timeOf_FirstRequest).Milliseconds())
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), timeOf_4thAck.Sub(timeOf_FirstRequest).Milliseconds())
		assert.Equal(t, []byte{utilities.ACK}, buffer_1Byte)

		//-- expect STX (for the answer, always looking for answers :D)
		instrument.Read(buffer_1Byte)
		timeOf_SMessageSTX := time.Now()
		assert.Equal(t, []byte{utilities.STX}, buffer_1Byte)
		assert.LessOrEqual(t, int64(522-expectedLatency_inMs_TimesTwo), timeOf_SMessageSTX.Sub(timeOf_4thAck).Milliseconds()) // NOT sure here maybe other timings
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), timeOf_SMessageSTX.Sub(timeOf_4thAck).Milliseconds())

		// wait for 0.5 secs
		time.Sleep(500*time.Millisecond + expectedLatency_inMs_TimesTwo) // TODO: Check if that was not 2 secs

		//-- Send ACK to confirm the start of message
		_, err = instrument.Write([]byte{utilities.ACK})
		timeOf_SendingAckForSTXforSMessage := time.Now()
		assert.Nil(t, err)

		//! here must be 0,5 seconds (t4) !!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!

		instrument.Read(buffer_32Bytes)
		timeOf_AfterSMessageRead := time.Now()
		// the transmission of 31 bytes takes 522 ms ! instrument has 9600 boud but the test environment has about 50 mio boud
		assert.LessOrEqual(t, int64(500), timeOf_AfterSMessageRead.Sub(timeOf_SendingAckForSTXforSMessage).Milliseconds())
		assert.GreaterOrEqual(t, int64(2522-expectedLatency_inMs_TimesTwo), timeOf_AfterSMessageRead.Sub(timeOf_SendingAckForSTXforSMessage).Milliseconds())
		assert.Equal(t, []byte("S 34567890123456789012345678901\n"), buffer_32Bytes)

		// wait for 0.5 secs
		time.Sleep(500*time.Millisecond + expectedLatency_inMs_TimesTwo)

		//-- Send ACK
		_, err = instrument.Write([]byte{utilities.ACK})
		assert.Nil(t, err)

		// Wait T5 < x -- Thats how long it takes before device sends next message: device is stupid
		// we are just simulating the behaviour here
		time.Sleep(2000*time.Millisecond + expectedLatency_inMs_TimesTwo)

		//-- send STX (to start the Request)
		_, err = instrument.Write([]byte{utilities.STX}) // no bcc)
		timeOf_EndTransferSTX := time.Now()
		assert.Nil(t, err)

		_, err = instrument.Read(buffer_1Byte)
		assert.LessOrEqual(t, int64(500), time.Now().Sub(timeOf_EndTransferSTX).Milliseconds())
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), time.Now().Sub(timeOf_EndTransferSTX).Milliseconds())
		assert.Equal(t, []byte{utilities.ACK}, buffer_1Byte)

		_, err = instrument.Write([]byte{'R', 'E', '0', '3', utilities.LF})
		assert.Nil(t, err)

		//++ expect ACK
		_, err = instrument.Read(buffer_1Byte)
		timeOf_6thAck := time.Now()
		assert.Nil(t, err)
		assert.LessOrEqual(t, int64(500), timeOf_6thAck.Sub(timeOf_EndTransferSTX).Milliseconds())
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), timeOf_6thAck.Sub(timeOf_EndTransferSTX).Milliseconds())
		assert.Equal(t, []byte{utilities.ACK}, buffer_1Byte)

		// Sending a byte with a wrong symbol to stop the test
		instrument.Write([]byte{'?'})
	}()

	// from here on we become the host :) - (thats ourselfes)
	instance := Logger(BeckmanSpecialProtocol(DefaultBeckmanSpecialProtocolSettings().
		SetStartByte(0x02).
		SetEndByte(0x0A).
		SetLineBreakByte(0x0A).SetAcknowledgementTimeout(500 * time.Millisecond)))

	r1message, err := instance.Receive(host)
	assert.Nil(t, err)
	assert.Equal(t, "R 03111101 00160123456789", string(r1message))

	str := "S 34567890123456789012345678901" // 32 bytes (content dont care :)
	_, err = instance.Send(host, [][]byte{[]byte(str)})
	assert.Nil(t, err)

	// Need to wait here. Because some timing issue by the host
	time.Sleep(time.Second)

	_, err = instance.NewInstance().Receive(host)
	assert.ErrorContainsf(t, err, `invalid character : "?"`, "")
}

func TestMultipleMessageRequestResponse(t *testing.T) {
}

func TestMultipleMessageRequestResponseIncludingARetryOnSimulatedFail(t *testing.T) {
	// TODO
}
