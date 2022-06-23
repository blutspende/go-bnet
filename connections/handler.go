package connections

import "time"

type Handlers interface {
	HandleDataReceived(source string, fileData []byte, receiveTimestamp time.Time)
	HandleServerSession(conn BloodLabConn) // TODO: RENAME?!
}
