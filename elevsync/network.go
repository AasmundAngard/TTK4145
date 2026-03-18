package elevsync

import (
	"errors"
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

func (OtherElevatorList *OtherElevatorList) getAlive(ID string) (bool, error) {
	for _, otherElevator := range *OtherElevatorList {
		if otherElevator.ID == ID && otherElevator.Alive {
			return true, nil
		} else if otherElevator.ID == ID && !otherElevator.Alive {
			return false, nil
		}
	}
	return false, errors.New("ID not found")
}

func (OtherElevatorList *OtherElevatorList) setAlive(SenderID string, aliveStatus bool) {
	for i, elevator := range *OtherElevatorList {
		if elevator.ID == SenderID {
			(*OtherElevatorList)[i].Alive = aliveStatus
			return
		}
	}
}
func (OtherElevatorList *OtherElevatorList) setHallCalls(SenderID string, HallCalls HallCalls) {
	for i, elevator := range *OtherElevatorList {
		if elevator.ID == SenderID {
			(*OtherElevatorList)[i].Calls.HallCalls = HallCalls
			return
		}
	}
}

func (OtherElevatorList OtherElevatorList) findNewAndLostPeers(alivePeersList []string) ([]string, []string) {
	var newPeers []string
	var lostPeers []string
	for _, peer := range alivePeersList {
		newPeers = append(newPeers, peer)
		for _, elevator := range OtherElevatorList {
			if elevator.ID == peer {
				newPeers = newPeers[:len(newPeers)-1]
			}
		}
	}
	for _, elevator := range OtherElevatorList {
		lostPeers = append(lostPeers, elevator.ID)
		for _, peer := range alivePeersList {
			if elevator.ID == peer {
				lostPeers = lostPeers[:len(lostPeers)-1]
			}
		}
	}
	return newPeers, lostPeers
}

func (OtherElevatorList *OtherElevatorList) detectReconnect(prevAlivePeers []string) bool {
	for id := range prevAlivePeers {
		print(id)
	}

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
			if self.HallCalls[floor][btn].NeedService == true {
				needService = true

			}
			(*self).HallCalls[floor][btn].NeedService = needService
			(*self).HallCalls[floor][btn].Version = maxVersion + 1
			for i, otherElevator := range *OtherElevatorList {
				if otherElevator.Alive == true {
					(*OtherElevatorList)[i].Calls.HallCalls[floor][btn].Version = maxVersion + 1
					(*OtherElevatorList)[i].Calls.HallCalls[floor][btn].NeedService = needService
				}
			}
		}
	}
}

// Blocking, to make sure the elevators have synchronized data before ruining everything
func (OtherElevatorList *OtherElevatorList) updateSelfInOthersAndOthersInSelf(alivePeersList []string,
	alivePeersC <-chan []string,
	otherDataToSyncC <-chan NetworkMsg,
	networkRequestSelfDataC <-chan struct{},
	selfDataToNetworkC chan<- NetworkMsg,
	NetworkMsgVersion int64, id string, localCallsPtr *Calls, localStatePtr *elevstate.ElevState, otherCabCallsRequestC <-chan string, otherCabCallsToNetworkC chan<- CabCalls) int64 {

	var ReconnectRespondents []string
	var incomingNetworkMsg NetworkMsg

	DrainChannel(otherDataToSyncC, &incomingNetworkMsg)

	print("Waiting for responses")
	for len(ReconnectRespondents) < len(alivePeersList)-1 {
		if (len(alivePeersList)) == 1 {
			break
		}

		select {
		//===========================
		// Denne må stå først, for at heisen skal begynne å sende ut det den har, før den mottar oppdatert state.
		// Hvorfor skal vi egentlig måtte ha de andre?
		case ID := <-otherCabCallsRequestC:
			// Heis sender ID og spør om sine cab calls
			//print("Request calls")
			fmt.Println("request calls 2:")
			// otherCabCallsToNetworkC <- OtherElevatorList.getCabCallsfromID(ID)
			for _, elev := range *OtherElevatorList {
				for _, floor := range elev.Calls.CabCalls {
					fmt.Println(floor.NeedService)
				}
			}
			// Dette gir false for alle når requesten skjer

			cabBalls := OtherElevatorList.getCabCallsfromID(ID)
			for _, floor := range cabBalls {
				fmt.Println(floor.NeedService)
			}
			otherCabCallsToNetworkC <- cabBalls
			continue

		case incomingNetworkMsg := <-otherDataToSyncC:
			// Mottar nettverk-melding
			// Kan motta melding fra oppstartende heis med feil cab calls
			if !slices.Contains(ReconnectRespondents, incomingNetworkMsg.SenderID) {
				ReconnectRespondents = append(ReconnectRespondents, incomingNetworkMsg.SenderID)
				fmt.Println("before without version check")
				for _, elev := range *OtherElevatorList {
					for _, floor := range elev.Calls.CabCalls {
						fmt.Println(floor.NeedService)
					}
				}
				fmt.Println(incomingNetworkMsg)
				// Oppstartende heis sender sin egen state før den har mottatt sine cab calls
				(*OtherElevatorList).updateWithoutVersionCheck(incomingNetworkMsg)
				fmt.Println("after without version check")
				for _, elev := range *OtherElevatorList {
					for _, floor := range elev.Calls.CabCalls {
						fmt.Println(floor.NeedService)
					}
				}
			}

		case <-networkRequestSelfDataC:
			// Network spør hele tiden om lokal state for å sende den
			selfDataToNetworkC <- NetworkMsg{Version: NetworkMsgVersion, SenderID: id, Calls: *localCallsPtr, State: *localStatePtr}
			NetworkMsgVersion++

		case alivePeersList := <-alivePeersC:
			// Her mottas oppdatering om ny peer
			// Burde man bare sende cab calls med en gang kanskje?
			// Samme kanal som elevsync.go
			OtherElevatorList.updateAliveStatus(alivePeersList)

		}
	}

	print("Received responses from all alive elevators, continuing")
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

func (OtherElevatorList *OtherElevatorList) updateWithoutVersionCheck(incomingNetworkMsg NetworkMsg) {
	elevatorFound := false

	for i, otherElevator := range *OtherElevatorList {
		if otherElevator.ID == incomingNetworkMsg.SenderID {
			(*OtherElevatorList)[i].State = incomingNetworkMsg.State
			(*OtherElevatorList)[i].Calls = incomingNetworkMsg.Calls
			(*OtherElevatorList)[i].Version = incomingNetworkMsg.Version
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

func (OtherElevatorList *OtherElevatorList) updateAliveStatus(alivePeersList []string) []string {
	returnedElevators := []string{}

	for i, otherElevator := range *OtherElevatorList {
		alive := false
		for _, alivePeer := range alivePeersList {
			if otherElevator.ID == alivePeer {
				alive = true
				break
			}
		}
		if (*OtherElevatorList)[i].Alive == false && alive == true {
			// Reconnect
			returnedElevators = append(returnedElevators, (*OtherElevatorList).getIDsString())

		}
		(*OtherElevatorList)[i].Alive = alive
		if !alive {
			fmt.Println("Elevator " + otherElevator.ID + " is dead.")
		}
	}
	return returnedElevators
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

func DrainChannel[T any](variableC <-chan T, variable *T) {
drainChannel:
	for {
		select {
		case *variable = <-variableC:
		default:
			break drainChannel
		}
	}
}
