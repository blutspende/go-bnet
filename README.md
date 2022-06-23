# Go BloodLab Net

# Introduction



//
//func (session *SessionHandlerTCP) handleDataReived(source string, filedata []byte, receivetimestamp time.Time) err {
//	message := unmarshal(filedata....)
//	repo = repostiory.persist_val(message)
//	trigger_transmissioN_to_cia(repo)
//	return nil
//}
//
//// for servermode: this method is called on connect.ö the conneciton remains active for as long as the funciton does not exit
//func (session *SessionHandlerTCP) serverSession(OurConnection netHandle, source string) {
//	if !isListed(con) {
//		return
//	}
//	// alternative. non-blocking
//	// logic: check for new data... send new data
//	for netHandle.isAlive() {
//		analysis_request := <-events_from_elswehere_in_our_application // blocking!
//		netHandle.Send(anayls_request.data)
//
//		if schlechte_laune {
//			con.stop() // close connection
//		}
//	}
//	// return = disconnect this connection
//}
//