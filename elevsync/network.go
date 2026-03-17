package elevsync

import (
	"fmt"
	"reflect"
	"root/config"
	"root/elevstate"
	"slices"
	"strconv"
)

type NetworkMsg struct {
	Version  int64
	SenderID string
	Calls    Calls
	State    elevstate.ElevState
}

type OtherElevator struct {
	ID      string
	Version int64
	Calls   Calls
	State   elevstate.ElevState
	Alive   bool
}
type OtherElevatorList []OtherElevator
type OtherElevatorBool struct {
	ID           string
	State        elevstate.ElevState
	CabCallsBool CabCallsBool
}

func (OtherElevatorList *OtherElevatorList) detectReconnect(prevAlivePeers []string) bool {
	for _, otherElevator := range *OtherElevatorList {
		if otherElevator.Alive == true && slices.Contains(prevAlivePeers, otherElevator.ID) == false {
			return true
		}
	}
	return false
}

func (self *Calls) mergeHallCallsForgiving(OtherElevatorList *OtherElevatorList) {
	for floor := 0; floor < config.NumFloors; floor++ {
		for btn := 0; btn < 2; btn++ {

			maxVersion := int64(0)
			needService := false

			for _, otherElevator := range *OtherElevatorList {
				if maxVersion < otherElevator.Calls.HallCalls[floor][btn].Version && otherElevator.Alive == true {
					maxVersion = otherElevator.Calls.HallCalls[floor][btn].Version
				}
				if otherElevator.Calls.HallCalls[floor][btn].NeedService == UnservicedCall && otherElevator.Alive == true {
					needService = UnservicedCall
				}
			}

			// Check self as well, and update version number and needService for the elevators.
			if maxVersion < self.HallCalls[floor][btn].Version {
				maxVersion = self.HallCalls[floor][btn].Version
			}
			(*self).HallCalls[floor][btn].Version = maxVersion + 1

			if self.HallCalls[floor][btn].NeedService == true {
				needService = true
			}
			(*self).HallCalls[floor][btn].NeedService = needService

			for i, otherElevator := range *OtherElevatorList {
				if otherElevator.Alive == true {
					(*OtherElevatorList)[i].Calls.HallCalls[floor][btn].Version = maxVersion + 1
					(*OtherElevatorList)[i].Calls.HallCalls[floor][btn].NeedService = needService
				}
			}
		}
	}
}

func (OtherElevatorList *OtherElevatorList) updateSelfInOthersAndOthersInSelf(alivePeersList []string,
	//Blocking, to make sure the elevators have synchronized data before ruining everything

	otherDataToSyncC <-chan NetworkMsg,
	networkRequestSelfDataC <-chan struct{},
	selfDataToNetworkC chan<- NetworkMsg,
	NetworkMsgVersion int64, id string, localCallsPtr *Calls, localStatePtr *elevstate.ElevState) int64 {
	var ReconnectRespondents []string

	for len(ReconnectRespondents) < len(alivePeersList)-1 {
		print("Waiting for responses")
		select {
		case incomingNetworkMsg := <-otherDataToSyncC:
			if !slices.Contains(ReconnectRespondents, incomingNetworkMsg.SenderID) {
				ReconnectRespondents = append(ReconnectRespondents, incomingNetworkMsg.SenderID)

				(*OtherElevatorList).update(incomingNetworkMsg)
			}

		case <-networkRequestSelfDataC:
			selfDataToNetworkC <- NetworkMsg{Version: NetworkMsgVersion, SenderID: id, Calls: *localCallsPtr, State: *localStatePtr}
			NetworkMsgVersion++
		}
	}

	return NetworkMsgVersion
}

func (otherElevatorList OtherElevatorList) getCabCallsfromID(ID string) CabCalls {
	cabCalls := newCabCalls()

	for _, otherElevator := range otherElevatorList {
		if otherElevator.ID == ID {
			return otherElevator.Calls.CabCalls
		}
	}
	return cabCalls
}

func (OtherElevatorList *OtherElevatorList) update(incomingNetworkMsg NetworkMsg) {
	elevatorFound := false

	for i, otherElevator := range *OtherElevatorList {
		if otherElevator.ID == incomingNetworkMsg.SenderID {
			if otherElevator.Version < incomingNetworkMsg.Version {
				(*OtherElevatorList)[i].State = incomingNetworkMsg.State
				(*OtherElevatorList)[i].Calls = incomingNetworkMsg.Calls
				(*OtherElevatorList)[i].Version = incomingNetworkMsg.Version
			}
			elevatorFound = true
			break
		}
	}

	if !elevatorFound {
		*OtherElevatorList = append(*OtherElevatorList, OtherElevator{ID: incomingNetworkMsg.SenderID, Version: incomingNetworkMsg.Version, State: incomingNetworkMsg.State, Calls: incomingNetworkMsg.Calls, Alive: true})
		if len(*OtherElevatorList) > config.NumElevators-1 {
			panic("Too many elevators in the system:" + strconv.Itoa(len(*OtherElevatorList)) + " " + OtherElevatorList.getIDsString())
		}
	}
}

func (OtherElevatorList *OtherElevatorList) updateAliveStatus(alivePeersList []string) {

	for i, otherElevator := range *OtherElevatorList {
		alive := false
		for _, alivePeer := range alivePeersList {
			if otherElevator.ID == alivePeer {
				alive = true
				break
			}
		}
		(*OtherElevatorList)[i].Alive = alive
		if !alive {
			fmt.Println("Elevator " + otherElevator.ID + " is dead.")
		}
	}
}

func (OtherElevatorList OtherElevatorList) workingElevsOnlyToBool() []OtherElevatorBool {
	var OtherElevatorBoolList []OtherElevatorBool

	for _, otherElevator := range OtherElevatorList {
		if otherElevator.Alive == true {
			OtherElevatorBoolList = append(OtherElevatorBoolList, OtherElevatorBool{ID: otherElevator.ID, State: otherElevator.State, CabCallsBool: otherElevator.Calls.CabCalls.toBool()})
		}
	}

	return OtherElevatorBoolList
}

func (OtherElevatorList OtherElevatorList) getIDsString() string {
	var IDs string

	//Inneficient, but only used for debug
	for _, otherElevator := range OtherElevatorList {
		IDs += otherElevator.ID + " "
	}

	return IDs
}

type SyncedData struct {
	LocalCabCalls         CabCallsBool
	SyncedHallCalls       HallCallsBool
	OtherElevatorBoolList []OtherElevatorBool
}

func (syncedData *SyncedData) format(confirmedCalls CommonCalls, OtherElevatorList OtherElevatorList) {
	syncedData.LocalCabCalls = confirmedCalls.CabCalls
	syncedData.SyncedHallCalls = confirmedCalls.HallCalls
	syncedData.OtherElevatorBoolList = OtherElevatorList.workingElevsOnlyToBool()
}
func (thisSyncedData *SyncedData) Equals(otherSyncedData SyncedData) bool {
	if thisSyncedData.LocalCabCalls != otherSyncedData.LocalCabCalls {
		return false
	} else if thisSyncedData.SyncedHallCalls != otherSyncedData.SyncedHallCalls {
		return false
	} else if len(thisSyncedData.OtherElevatorBoolList) != len(otherSyncedData.OtherElevatorBoolList) {
		// Not handled by DeepEqual :(, and needed to update main when elevators disconnect as main rejects messages with same data as earlier
		return false
	} else if !reflect.DeepEqual(thisSyncedData.OtherElevatorBoolList, otherSyncedData.OtherElevatorBoolList) {
		return false
	}
	return true
}
