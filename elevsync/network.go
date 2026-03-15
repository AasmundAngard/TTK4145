package elevsync

import (
	"root/config"
	"root/elevstate"
	"strconv"
)

type NetworkMsg struct {
	Version   int64
	SenderID  string
	Calls     Calls
	State     elevstate.ElevState
}

type OtherElevator struct {
	ID        string
	Version	  int64
	Calls     Calls
	State     elevstate.ElevState
	Alive     bool
}
type OtherElevatorList []OtherElevator
type OtherElevatorBool struct {
	//ID		   	 int
	State        elevstate.ElevState
	CabCallsBool CabCallsBool
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
		*OtherElevatorList = append(*OtherElevatorList, OtherElevator{ID: incomingNetworkMsg.SenderID, Version: incomingNetworkMsg.Version, State: incomingNetworkMsg.State, Calls: incomingNetworkMsg.Calls})
		if len(*OtherElevatorList) > config.NumElevators {
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
			// Sus, should reset Version when dead, but not disconnect????
			(*OtherElevatorList)[i].Version = 0
		}
		(*OtherElevatorList)[i].Alive = alive
	}
}

func (OtherElevatorList OtherElevatorList) workingElevsOnlyToBool() []OtherElevatorBool {
	var OtherElevatorBoolList []OtherElevatorBool

	for _, otherElevator := range OtherElevatorList {
		if otherElevator.Alive == true {
			OtherElevatorBoolList = append(OtherElevatorBoolList, OtherElevatorBool{State: otherElevator.State, CabCallsBool: otherElevator.Calls.CabCalls.toBool()})
		}
	}

	return OtherElevatorBoolList
}

func (OtherElevatorList OtherElevatorList) getIDsString() string {
	var IDs string

	for _, otherElevator := range OtherElevatorList {
		IDs += otherElevator.ID + " "
	}

	return IDs
}

type ConfirmedData struct {
	LocalCabCalls         CabCallsBool
	SyncedHallCalls       HallCallsBool
	OtherElevatorBoolList []OtherElevatorBool
}

func (syncedData *ConfirmedData) format(confirmedCalls CallsBool, OtherElevatorList OtherElevatorList) {
	syncedData.LocalCabCalls = confirmedCalls.CabCallsBool
	syncedData.SyncedHallCalls = confirmedCalls.HallCallsBool
	syncedData.OtherElevatorBoolList = OtherElevatorList.workingElevsOnlyToBool()
}