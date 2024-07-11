package protocol

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/blutspende/go-bnet/protocol/utilities"
	"github.com/stretchr/testify/assert"
)

func TestOneMessageRequestResponse(t *testing.T) {

	host, instrument := net.Pipe()

	os.Setenv("PROTOLOG_ENABLE", "extended") // enable logging

	go func() { // This is the instrument (you must become the instrument yourself to read it)
		const expectedLatency_inMs_TimesTwo = 40 // ms
		buffer_1Byte := make([]byte, 1)          // including STX and 0A in the transmission
		buffer_33Bytes := make([]byte, 33)       // including STX and 0A in the transmission
		buffer_SE := make([]byte, 4)

		//-- Send RB03 (Start of Request block)
		_, err := instrument.Write([]byte{utilities.STX, 'R', 'B', '0', '3', utilities.LF}) // no bcc)
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

		//-- Send R_ Request Message for instrument
		instrument.Write([]byte{utilities.STX, 'R', ' ', '0', '3', /*instrument#*/
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

		instrument.Read(buffer_33Bytes)
		timeOf_AfterSMessageRead := time.Now()
		// the transmission of 31 bytes takes 522 ms ! instrument has 9600 boud but the test environment has about 50 mio boud
		assert.LessOrEqual(t, int64(500), timeOf_AfterSMessageRead.Sub(timeOf_4thAck).Milliseconds())
		assert.GreaterOrEqual(t, int64(2522-expectedLatency_inMs_TimesTwo), timeOf_AfterSMessageRead.Sub(timeOf_4thAck).Milliseconds())
		assert.Equal(t, append([]byte{utilities.STX}, []byte("S 34567890123456789012345678901\n")...), buffer_33Bytes)

		// wait for 0.5 secs
		time.Sleep(500*time.Millisecond + expectedLatency_inMs_TimesTwo)

		//-- Send ACK
		_, err = instrument.Write([]byte{utilities.ACK})
		assert.Nil(t, err)

		// Wait T5 < x -- Thats how long it takes before device sends next message: device is stupid
		// we are just simulating the behaviour here
		time.Sleep(2000*time.Millisecond + expectedLatency_inMs_TimesTwo)

		_, err = instrument.Write([]byte{utilities.STX, 'R', 'E', '0', '3', utilities.LF})
		timeOf_EndTransferSTX := time.Now()
		assert.Nil(t, err)

		//++ expect ACK
		_, err = instrument.Read(buffer_1Byte)
		timeOf_6thAck := time.Now()
		assert.Nil(t, err)
		assert.LessOrEqual(t, int64(500), timeOf_6thAck.Sub(timeOf_EndTransferSTX).Milliseconds())
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), timeOf_6thAck.Sub(timeOf_EndTransferSTX).Milliseconds())
		assert.Equal(t, []byte{utilities.ACK}, buffer_1Byte)

		_, err = instrument.Read(buffer_SE)
		timeOf_SE := time.Now()
		assert.Nil(t, err)
		assert.LessOrEqual(t, int64(500), timeOf_SE.Sub(timeOf_6thAck).Milliseconds())
		assert.GreaterOrEqual(t, int64(2000-expectedLatency_inMs_TimesTwo), timeOf_6thAck.Sub(timeOf_EndTransferSTX).Milliseconds())
		assert.Equal(t, []byte{utilities.STX, 'S', 'E', utilities.LF}, buffer_SE)

		time.Sleep(500 * time.Millisecond)
		_, err = instrument.Write([]byte{utilities.ACK})
		assert.Nil(t, err)
	}()

	// from here on we become the host :) - (thats ourselfes)
	instance := Logger(AU6XXProtocol(DefaultAU6XXProtocolSettings().
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

	ackBytes, err := instance.NewInstance().Receive(host)
	assert.Equal(t, []byte{}, ackBytes)
	//assert.ErrorContainsf(t, err, `invalid character : "?"`, "")
}

func TestMultipleMessageRequestResponse(t *testing.T) {
}

func TestMultipleMessageRequestResponseIncludingARetryOnSimulatedFail(t *testing.T) {
	// TODO
}

func TestDataResultMessageWithANAK(t *testing.T) {
	// TODO
}

/*
An Request starts with

	RB00 (where 00 is the instrument)
	... request data
	RE00 (ending)
*/
func TestAfterRBThereShouldBeNoSEfromHost(t *testing.T) {
	/*
		fmt.Println("--- start test")
		host, instrument := net.Pipe()
		buffer_1Byte := make([]byte, 1)

		os.Setenv("PROTOLOG_ENABLE", "extended") // enable logging

		fmt.Println("Lets go")
		go func() {
			fmt.Println("Sending RB")
			_, err := instrument.Write([]byte{utilities.STX, 'R', 'B', '0', '3', utilities.LF}) // no bcc)
			assert.Nil(t, err)

			fmt.Println("Sending RB")
			//++ expect ACK
			_, err = instrument.Read(buffer_1Byte)

			_, err = instrument.Write([]byte{utilities.STX, 'R', 'E', '0', '3', utilities.LF}) // no bcc)
			assert.Nil(t, err)

			//++ expect ACK
			_, err = instrument.Read(buffer_1Byte)

			// establish stx
			buffer := make([]byte, 1024)
			n, err := instrument.Read(buffer)
			assert.Nil(t, err)
			fmt.Println(" n = ", n, " buffer = ", buffer[:n])
		}()

		instance := Logger(AU6XXProtocol(DefaultAU6XXProtocolSettings().
			SetStartByte(0x02).
			SetEndByte(0x0A).
			SetLineBreakByte(0x0A).SetAcknowledgementTimeout(500 * time.Millisecond)))

		fmt.Println("Reading buffer")
		buffer, err := instance.Receive(host)
		assert.Nil(t, err)
		assert.Equal(t, string(buffer), "RB03")

		_, err = instrument.Write([]byte{utilities.ACK})

		// Read "RB03"
		n, err := host.Read(buffer)
		assert.Nil(t, err)
		assert.Equal(t, 5, n)
		time.Sleep(time.Second)

		ackBytes, err := instance.NewInstance().Receive(host)
		assert.Nil(t, err)
		assert.Equal(t, []byte{}, ackBytes)
	*/
}

func TestServerRestartWhileInRequestMode(t *testing.T) {
	// TODO
}

func TestBugAfterREStartDataTransmission(t *testing.T) {
	// Send RB03
	// Send RE03
	// Send <STX>D.....
	// failed with this commit, state 16 did not lead to 1 (start receiving)
}
