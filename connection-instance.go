package go_bloodlab_net

type ConnectionInstance interface {
	Run(handler Handler)
	Send(data []byte) (int, error)
	Stop()
	Receive() ([]byte, error)
}
