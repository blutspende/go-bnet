package protocol

/*
type ProtocolReader interface {
	Init() error
	Scan(byte) error
	IsCompleteMessage() bool
	GetMessage() ([]byte, error)
}

type RawProtocolReader struct {
	buffer []byte
	state  int
}

func (rpr *RawProtocolReader) Init() {
	rpr.buffer = make([]byte, 0)
	rpr.state = 0
}

func (rpr *RawProtocolReader) Scan(b byte) error {
	rpr.buffer = append(rpr.buffer, b)
	return nil
}

func (rpr *RawProtocolReader) IsCompleteMessage() {

}

func (rpr *RawProtocolReader) GetMessage() ([]byte, error) {

}


*/
