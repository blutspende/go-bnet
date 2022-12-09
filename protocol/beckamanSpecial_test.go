package protocol

import (
	"net"
	"os"
	"testing"

	"github.com/DRK-Blutspende-BaWueHe/go-bloodlab-net/protocol/utilities"
	"github.com/stretchr/testify/assert"
)

func TestTimings(t *testing.T) {

	//var mc mockOnlineConn

	host, instrument := net.Pipe()

	os.Setenv("PROTOLOG_ENABLE", "true") // enable logging
	/*	instance := Logger(BeckmanSpecialProtocol(DefaultBeckmanSpecialProtocolSettings().
		SetStartByte(0x02).
		SetEndByte(0x0A).
		SetLineBreakByte(0x0A)))
	*/
	//request := instance.Receive(mc, c)

	instrument.Write([]byte{2})
	//expect ACK

	RB := []byte{utilities.STX, 'R', 'B', '0', '3', utilities.LF, 0 /*bcc*/}
	instrument.Write(RB)
	//expect ACK

	R1 := []byte{utilities.STX, 'R', ' ', '0', '3', '1', '1', '1', '1', /* RACK# */
		'0', '1' /*CUP*/, '2', '2', '2', '2', /*SAMPLENO*/
		'9', '9', '9', '9', '9', '9', '9', '9', '9', '9', utilities.LF, 0 /*bcc*/}
	instrument.Write(R1)
	//expect ACK

	// expect SB
	// expect S .....

	RE := []byte{utilities.STX, 'R', 'E', '0', '3', utilities.LF, 0 /*bcc*/}
	instrument.Write(RE)

	//expect ACK
	buffer1 := make([]byte, 1)
	instrument.Read(buffer1)
	assert.Equal(t, utilities.ACK, buffer1)

}
