package elevsync

import (
	"errors"
	"fmt"
	"reflect"
	"root/config"
	"root/elevator"
	"slices"
	"strconv"
)

type NetworkMsg struct {
	Version  int64
	SenderId string
	Calls    Calls
	State    elevator.ElevState
}

type CabNetworkMsg struct {
	SenderId    string
	RequesterId string
	CabCalls    CabCalls
}

type peerElevator struct {
	Id      string
	Version int64
	Calls   Calls
	State   elevator.ElevState
	Alive   bool
}

type ConfirmedPeerElevator struct {
	Id       string
	State    elevator.ElevState
	CabCalls ConfirmedCabCalls
}

type SystemStatus struct {
	SelfCabCalls    ConfirmedCabCalls
	CommonHallCalls ConfirmedHallCalls
	PeerElevators   []ConfirmedPeerElevator
}

func (systemStatus *SystemStatus) format(commonCalls ConfirmedCalls, peers peerElevatorList) {
	systemStatus.SelfCabCalls = commonCalls.CabCalls
	systemStatus.CommonHallCalls = commonCalls.HallCalls
	systemStatus.PeerElevators = peers.workingElevatorsToStates()
}

func (thisSystemStatus *SystemStatus) Equals(thatSystemStatus SystemStatus) bool {
	if thisSystemStatus.SelfCabCalls != thatSystemStatus.SelfCabCalls {
		return false
	}
	if thisSystemStatus.CommonHallCalls != thatSystemStatus.CommonHallCalls {
		return false
	}
	if len(thisSystemStatus.PeerElevators) != len(thatSystemStatus.PeerElevators) {
		return false
	}
	if !reflect.DeepEqual(thisSystemStatus.PeerElevators, thatSystemStatus.PeerElevators) {
		return false
	}
	return true
}

type peerElevatorList []peerElevator

func (peers *peerElevatorList) getAlive(Id string) (bool, error) {
	for _, peerElevator := range *peers {
		if peerElevator.Id == Id {
			return peerElevator.Alive, nil
		}
	}
	return false, errors.New("id not found")
}

func (peers *peerElevatorList) setAlive(senderId string, aliveStatus bool) {
	for i, elevator := range *peers {
		if elevator.Id == senderId {
			(*peers)[i].Alive = aliveStatus
			return
		}
	}
}

func (peers *peerElevatorList) resetVersion(senderId string) {
	for i, elevator := range *peers {
		if elevator.Id == senderId {
			(*peers)[i].Version = 0
			return
		}
	}
}

func (peers *peerElevatorList) setHallCalls(senderId string, hallCalls hallCalls) {
	for i, elevator := range *peers {
		if elevator.Id == senderId {
			(*peers)[i].Calls.HallCalls = hallCalls
			return
		}
	}
}

func (peers peerElevatorList) findNewAndLostPeers(alivePeers []string) ([]string, []string) {
	var newPeers []string
	var lostPeers []string

	for _, peerId := range alivePeers {
		isKnown := false
		for _, elevator := range peers {
			if elevator.Id == peerId {
				isKnown = true
				break
			}
		}
		if !isKnown {
			newPeers = append(newPeers, peerId)
		}
	}

	for _, elevator := range peers {
		if !slices.Contains(alivePeers, elevator.Id) {
			lostPeers = append(lostPeers, elevator.Id)
		}
	}

	return newPeers, lostPeers
}

func (peers *peerElevatorList) detectReconnect(previousAlivePeers []string) bool {
	for _, peerElevator := range *peers {
		if peerElevator.Alive && !slices.Contains(previousAlivePeers, peerElevator.Id) {
			return true
		}
	}
	return false
}

func (peers peerElevatorList) getCabCallsFromId(peerId string) CabCalls {
	for _, peerElevator := range peers {
		if peerElevator.Id == peerId {
			return peerElevator.Calls.CabCalls
		}
	}
	return newCabCalls()
}

func (peers *peerElevatorList) update(incoming NetworkMsg) {
	elevatorFound := false

	for i, peerElevator := range *peers {
		if peerElevator.Id == incoming.SenderId {
			if peerElevator.Version < incoming.Version {
				(*peers)[i].State = incoming.State
				(*peers)[i].Calls = incoming.Calls
				(*peers)[i].Version = incoming.Version
			}
			elevatorFound = true
			break
		}
	}

	if !elevatorFound {
		*peers = append(*peers, peerElevator{Id: incoming.SenderId, Version: incoming.Version, State: incoming.State, Calls: incoming.Calls, Alive: true})
		if len(*peers) > config.NumElevators-1 {
			panic("too many elevators in the system: " + strconv.Itoa(len(*peers)) + " " + peers.getIdsString())
		}
	}
}

func (peers *peerElevatorList) updateWithoutVersionCheck(incoming NetworkMsg) {
	elevatorFound := false

	for i, peerElevator := range *peers {
		if peerElevator.Id == incoming.SenderId {
			(*peers)[i].State = incoming.State
			(*peers)[i].Calls = incoming.Calls
			(*peers)[i].Version = incoming.Version
			elevatorFound = true
			break
		}
	}

	if !elevatorFound {
		*peers = append(*peers, peerElevator{Id: incoming.SenderId, Version: incoming.Version, State: incoming.State, Calls: incoming.Calls, Alive: true})
		if len(*peers) > config.NumElevators-1 {
			panic("too many elevators in the system: " + strconv.Itoa(len(*peers)) + " " + peers.getIdsString())
		}
	}
}

func (peers *peerElevatorList) updateAliveStatus(alivePeers []string) []string {
	returnedElevators := []string{}

	for i, peerElevator := range *peers {
		alive := slices.Contains(alivePeers, peerElevator.Id)
		if !(*peers)[i].Alive && alive {
			returnedElevators = append(returnedElevators, peerElevator.Id)
		}
		(*peers)[i].Alive = alive
		if !alive {
			fmt.Println("Elevator " + peerElevator.Id + " is dead.")
		}
	}

	return returnedElevators
}

func (peers peerElevatorList) workingElevatorsToStates() []ConfirmedPeerElevator {
	var peerStates []ConfirmedPeerElevator

	for _, peerElevator := range peers {
		if peerElevator.Alive {
			peerStates = append(peerStates, ConfirmedPeerElevator{Id: peerElevator.Id, State: peerElevator.State, CabCalls: peerElevator.Calls.CabCalls.confirm()})
		}
	}

	return peerStates
}

func (peers peerElevatorList) getIdsString() string {
	var ids string
	for _, peerElevator := range peers {
		ids += peerElevator.Id + " "
	}
	return ids
}
