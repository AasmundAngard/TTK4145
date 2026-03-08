package network

import (
	"root/elevstate"
	"root/elevsync"
)

// Network må regelmessig sende egen state (NetworkMsgStandard)
// Network må ta imot states fra alle heiser (hver heis broadcaster egen state)
// Network må gi sync beskjed når en heis er avkoblet
// Network må sende cabcalls når en annen heis ber om det

// Network må kunne broadcaste:
//	1. Standard MSG (Egen state)
//	2. En gitt heis sine Cab calls

// Network må kunne motta:
//	1. Standard MSG (fra alle heiser)
// 	2. CabCall requested list

/* Pseudocode
Init () {
	Broadcast requests for Cab Calls
	Listen for sent cab calls
	Send Cab calls to sync
}

StandardBroadcast () {
	Broadcast state
}

StandardListen () {
	Listen for states
	Parse state into sync-friendly type
	Send states to sync
	if lost peer:
		send lost peer id to sync
}

InitListen ()  {
	Listen for request for Cab
	if request for Cab:
		ask sync for cabcalls by id
		broadcast requested cab calls with requester-id
}



*/

type NetworkMsgStandard struct {
	SenderID  int
	TimeStamp int64
	Calls     elevsync.Calls
	State     elevstate.ElevState
}


func Broadcast() {

}

func Listen() {

}