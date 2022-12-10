package utilities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const LineReceived ActionCode = "LineReceived"

/**
This tests uses one of the basic requirements for the FSM: the LIS1A1 protocol.
**/
func TestBasic(t *testing.T) {
	automate := CreateFSM([]Rule{
		{FromState: Init, Symbols: []byte{ENQ}, ToState: 1, Scan: false},
		{FromState: 1, Symbols: []byte{STX}, ToState: 2, Scan: false},

		{FromState: 2, Symbols: []byte{ETB}, ToState: 5, Scan: false},
		{FromState: 5, Symbols: []byte{STX}, ToState: 2, Scan: false},

		{FromState: 2, Symbols: PrintableChars8Bit, ToState: 2, Scan: true},
		{FromState: 2, Symbols: []byte{CR}, ToState: 3, Scan: false, ActionCode: LineReceived},
		{FromState: 3, Symbols: []byte{ETX}, ToState: 7, Scan: false},
		{FromState: 3, Symbols: []byte{ETB}, ToState: Init, Scan: false},

		{FromState: 7, Symbols: []byte("01234567890ABCDEFabcdef"), ToState: 7, Scan: true},
		{FromState: 7, Symbols: []byte{CR}, ToState: 9, Scan: false, ActionCode: CheckSum},
		{FromState: 9, Symbols: []byte{LF}, ToState: 10, Scan: false},

		{FromState: 10, Symbols: []byte{STX}, ToState: 2, Scan: false},
		{FromState: 10, Symbols: []byte{ETB}, ToState: Init, Scan: false},
		{FromState: 10, Symbols: []byte{EOT}, ToState: Init, Scan: false, ActionCode: Finish},
	})

	communicationFlow := [][]byte{
		[]byte{ENQ},
		[]byte{STX},
		[]byte("1H|\\^&|||"),
		[]byte{CR},
		[]byte{ETX},
		[]byte{'5', '9'},
		[]byte{CR, LF},

		[]byte{STX},
		[]byte("2P|1||777025164810"),
		[]byte{CR},
		[]byte{ETX},
		[]byte{'A', '7'},
		[]byte{CR, LF},

		[]byte{STX},
		[]byte("3O|1|||^^^SARSCOV2IGG||20200811095913"),
		[]byte{CR},
		[]byte{ETX},
		[]byte{'B', '8'},
		[]byte{CR, LF},

		[]byte{STX},
		[]byte("4R|1|^^^SARSCOV2IGG|0,18|Ratio|"),
		[]byte{CR},
		[]byte{ETX},
		[]byte{'3', 'B'},
		[]byte{CR, LF},

		[]byte{STX},
		[]byte("5P|2||777642348910"),
		[]byte{CR},
		[]byte{ETX},
		[]byte{'B', '5'},
		[]byte{CR, LF},

		[]byte{STX},
		[]byte("6O|1|||^^^SARSCOV2IGG||20200811095913"),
		[]byte{CR},
		[]byte{ETX},
		[]byte{'B', 'B'},
		[]byte{CR, LF},

		[]byte{STX},
		[]byte("7R|1|^^^SARSCOV2IGG|0,18|Ratio|"),
		[]byte{CR},
		[]byte{ETX},
		[]byte{'3', 'E'},
		[]byte{CR, LF},

		[]byte{STX},
		[]byte("2L|1|N"),
		[]byte{CR},
		[]byte{ETX},
		[]byte{'0', '5', '&'},
		[]byte{CR, LF},
		[]byte{EOT},
	}

	exampleMessage := "1H|\\^&|||2P|1||7770251648103O|1|||^^^SARSCOV2IGG||202008110959134R|1|^^^SARSCOV2IGG|0,18|Ratio|5P|2||7776423489106O|1|||^^^SARSCOV2IGG||202008110959137R|1|^^^SARSCOV2IGG|0,18|Ratio|2L|1|N"
	exampleResult := make([]byte, 0)
	for _, rec := range communicationFlow {
		for _, token := range rec {
			buffer, action, err := automate.Push(token)
			switch action {
			case Error:
				assert.ErrorIs(t, err, ErrInvalidCharacter)
			case LineReceived:
				println(string(buffer))
				exampleResult = append(exampleResult, buffer...)
				automate.ResetBuffer()
			case CheckSum:
				println(string(buffer))
				automate.ResetBuffer()
			case Finish:
				// The end of the machine = recoginzed the message
				assert.Equal(t, exampleMessage, string(exampleResult))
			case Ok:
				// all fine
			default:
				t.Fatalf("default: no action found for: %s", action)
			}
		}
	}

}
