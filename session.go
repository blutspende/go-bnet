package go_bloodlab_net

type Session interface {
	IsAlive() bool
	Send(msg []byte) (int, error)
	Peek(n int) ([]byte, error)
	Read() ([]byte, error)
	Close()
	WaitTermination()
}
