package go_bloodlab_net

type Session interface {
	IsAlive() bool
	Send(msg []byte) (int, error)
	Read(buff []byte) (int, error)
	Close()
	WaitTermination()
	RemoteAddress() string
}
