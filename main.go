package main

import (
	"flag"
	"root/config"
	"root/elevio"
	"root/lights"
	"root/sync"
	"strconv"
)

type Behaviour int

const (
	Idle     Behaviour = 0
	Moving             = 1
	DoorOpen           = 2
)

type ElevState struct {
	Behaviour Behaviour
	Floor     int
	Direction Direction
}

func (e ElevState) toCabButtonEvent() elevio.ButtonEvent {
	return elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_Cab}
}
func (e ElevState) toHallButtonEvent() elevio.ButtonEvent {
	switch e.Direction {
	case Up:
		return elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallUp}
	case Down:
		return elevio.ButtonEvent{Floor: e.Floor, Button: elevio.BT_HallDown}
	default:
		panic("Invalid Direction to ButtonEvent")
	}
}

func NextState(hCalls sync.HallCallsBool, cCalls sync.CabCallsBool, state ElevState) ElevState {
	return ElevState{Behaviour: Moving, Floor: 0, Direction: Up}
}

func main() {

	idPtr := flag.Int("id", 0, "ID of elevator, overwrite using -id=<newId>")
	portPtr := flag.Int("fork", 20026, "Port of the elevator, overwrite using -port=<newPort>")
	flag.Parse()

	id := *idPtr
	port := *portPtr

	elevio.Init("localhost:"+strconv.Itoa(port), config.NumFloors) // Dette er til den lokale heisserveren man kan kjøre (alt. hardware)

	stopButtonC := make(chan bool, 16)
	floorSensorC := make(chan int, 1)
	openDoorC := make(chan bool, 1)
	doorClosedC := make(chan bool, 1)
	doorObstructedC := make(chan bool, 1)
	syncedVariablesC := make(chan sync.SyncedData, 16)
	localStateC := make(chan ElevState, 16)
	completedCallC := make(chan elevio.ButtonEvent, 16)

	go elevio.PollStopButton(stopButtonC)
	go elevio.PollFloorSensor(floorSensorC)
	go Door(openDoorC, doorClosedC, doorObstructedC)
	go sync.Sync(localStateC, completedCallC, syncedVariablesC)
	// func Sync(hardwareCalls chan CallEvent, finishedCalls chan CallEvent, networkMsg chan networkMsg, syncedData chan syncedData) {

	// Sync should not broadcast before main says so? Maybe uninitialized tag?

	// If between floors -> floor sensor registers no floors, go down until

	var state ElevState
	var syncedVariables sync.SyncedData
	var hCalls sync.HallCallsBool
	var cCalls sync.CabCallsBool

	for {

		select {
		case newFloor := <-floorSensorC:
			state.Floor = newFloor
			elevio.SetFloorIndicator(state.Floor)
			switch state.Behaviour {
			case Moving:
				nextState := sequenceAssigner.nextState(hCalls, cCalls, state)
				switch nextState.Behaviour {
				case DoorOpen:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Direction = nextState.Direction
					if cCalls[state.Floor] {
						cCalls[state.Floor] = false
						completedCallC <- state.toCabButtonEvent()
					}
					state.Behaviour = DoorOpen
				case Moving:
					state.Direction = nextState.Direction
					elevio.SetMotorDirection(state.Direction.toMD())
					state.Behaviour = Moving
				case Idle:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = Idle
				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = Idle
				}
				// switch {

				// case cCalls[state.Floor] && hCalls[state.Floor][state.Direction]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Behaviour = DoorOpen

				// 	completedCallC <- state.toCabButtonEvent()
				// 	cCalls[state.Floor] = false

				// case cCalls[state.Floor] && hCalls[state.Floor][state.Direction.Opposite()]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Direction = state.Direction.Opposite()
				// 	state.Behaviour = DoorOpen

				// 	completedCallC <- state.toCabButtonEvent()
				// 	cCalls[state.Floor] = false

				// case cCalls[state.Floor]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Behaviour = DoorOpen

				// 	completedCallC <- state.toCabButtonEvent()
				// 	cCalls[state.Floor] = false

				// case hCalls[state.Floor][state.Direction]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Behaviour = DoorOpen

				// case hCalls[state.Floor][state.Direction.Opposite()]:
				// 	elevio.SetMotorDirection(elevio.MD_Stop)
				// 	openDoorC <- true
				// 	state.Behaviour = DoorOpen
				// 	state.Direction = state.Direction.Opposite()

				// default:
				// 	if nextDirection == state.Direction.toMD() {
				// 		elevio.SetMotorDirection(state.Direction.toMD())

				// 	} else if nextDirection == state.Direction.Opposite().toMD() {
				// 		elevio.SetMotorDirection(state.Direction.Opposite().toMD())

				// 	} else {
				// 		elevio.SetMotorDirection(elevio.MD_Stop)
				// 		openDoorC <- true
				// 		state.Behaviour = DoorOpen
				// 	}

				// }
			default:
				panic("Impossible state")
			}

		case <-doorClosedC:
			switch state.Behaviour {
			case DoorOpen:
				nextState := sequenceAssigner.nextState(hCalls, cCalls, state)
				switch nextState.Behaviour {
				case Moving:
					elevio.SetMotorDirection(state.Direction.toMD())

					if hCalls[state.Floor][state.Direction] {
						completedCallC <- state.toHallButtonEvent()
						hCalls[state.Floor][state.Direction] = false
					}
					state.Behaviour = Moving
				case DoorOpen:
					openDoorC <- true
					state.Direction = state.Direction.Opposite()
					completedCallC <- state.toHallButtonEvent()
					hCalls[state.Floor][state.Direction] = false
					state.Behaviour = DoorOpen
				case Idle:
					state.Behaviour = Idle
				default:
					state.Behaviour = Idle
				}
			default:
				panic("Door closed in impossible state")
			}
		case syncedVariables = <-syncedVariablesC:

		drainChannel:
			for {
				select {
				case syncedVariables = <-syncedVariablesC:
				default:
					break drainChannel
				}
			}
			cCalls = syncedVariables.CallsBool.CabCalls[0]
			thisState := []sync.CompleteElevator{{State: state, CabCallsBool: cCalls}}
			allElevStates := append(thisState, syncedVariables.OtherElevators...)
			hCalls = sequenceAssigner.assignCalls(syncedVariables, allElevStates)

		case <-stopButtonC:
			elevio.SetMotorDirection(elevio.MD_Stop)
			state.Behaviour = Idle
		}
		lights.SetLights(syncedVariables.CallsBool)
		localStateC <- state
	}

}
