package go_bloodlab_net

import "time"

type Handler interface {
	DataReceived(source string, fileData []byte, receiveTimestamp time.Time)
	ClientConnected(source string, conn Session)
}
