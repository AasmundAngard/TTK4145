package main

import (
	"flag"
	"fmt"
	"root/config"
	"root/elevio"
	"root/elevstate"
	"root/elevsync"
	"root/lights"
	"root/sequenceassigner"
	"strconv"
	"time"
)

func NextState(hCalls elevsync.HallCallsBool, cCalls elevsync.CabCallsBool, state elevstate.ElevState) elevstate.ElevState {
	return elevstate.ElevState{Behaviour: elevstate.Moving, Floor: 0, Direction: elevstate.Up}
}

func main() {

	idPtr := flag.Int("id", 0, "ID of elevator, overwrite using -id=<newId>")
	portPtr := flag.Int("fork", config.HardwarePortNumber, "Port of the hardware server, overwrite using -port=<newPort>")
	flag.Parse()

	id := *idPtr
	fmt.Println(id)
	port := *portPtr

	elevio.Init("localhost:"+strconv.Itoa(port), config.NumFloors) // Dette er til den lokale heisserveren man kan kjøre (alt. hardware)

	stopButtonC := make(chan bool, 16)
	floorSensorC := make(chan int, 1)
	openDoorC := make(chan bool, 1)
	doorClosedC := make(chan bool, 1)
	doorObstructedC := make(chan bool, 1)

	hardWareCallsC := make(chan elevio.CallEvent, 16)
	localStateC := make(chan elevstate.ElevState, 16)
	completedCallC := make(chan elevio.CallEvent, 16)
	networkMsgC := make(chan elevsync.NetworkMsg, 16)
	syncedVariablesC := make(chan elevsync.SyncedData, 16)

	go elevio.PollStopButton(stopButtonC)
	go elevio.PollFloorSensor(floorSensorC)
	go elevio.PollButtons(hardWareCallsC)
	go Door(openDoorC, doorClosedC, doorObstructedC)
	go elevsync.Sync(hardWareCallsC, localStateC, completedCallC, networkMsgC, syncedVariablesC)
	// Sync should not broadcast before main says so? Maybe uninitialized tag?

	// If between floors -> floor sensor registers no floors, go down until

	var syncedVariables elevsync.SyncedData
	var hCalls elevsync.HallCallsBool
	var cCalls elevsync.CabCallsBool

	var state elevstate.ElevState
	var prevState elevstate.ElevState
	state.Behaviour = elevstate.Idle
	state.Direction = elevstate.Down

	floor := elevio.GetFloor()
	fmt.Println("startfloor:", floor)
	if floor != -1 {
		state.Floor = <-floorSensorC

	} else {
		elevio.SetMotorDirection(state.Direction.ToMD())
		state.Floor = <-floorSensorC
		elevio.SetMotorDirection(elevio.MD_Stop)
	}
	var i int = 0
	prevState = state
	prevState.Direction = state.Direction.Opposite()

	for {

		select {
		case newFloor := <-floorSensorC:
			fmt.Println("newfloor")
			state.Floor = newFloor
			elevio.SetFloorIndicator(state.Floor)
			switch state.Behaviour {
			case elevstate.Moving:
				nextState := sequenceassigner.NextState(hCalls, cCalls, state)
				switch nextState.Behaviour {
				case elevstate.DoorOpen:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Direction = nextState.Direction
					if cCalls[state.Floor] {
						cCalls[state.Floor] = false
						completedCallC <- state.ToCabCallEvent()
					}
					state.Behaviour = elevstate.DoorOpen
					localStateC <- state
				case elevstate.Moving:
					state.Direction = nextState.Direction
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = elevstate.Moving
				case elevstate.Idle:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = elevstate.Idle
					localStateC <- state
				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = elevstate.Idle
				}

			default:
				panic("New floor in impossible state")
			}

		case <-doorClosedC:
			switch state.Behaviour {
			case elevstate.DoorOpen:
				nextState := sequenceassigner.NextState(hCalls, cCalls, state)
				switch nextState.Behaviour {
				case elevstate.Moving:
					fmt.Println("doorclose: nextstate moving,", nextState.Direction)
					state.Direction = nextState.Direction

					elevio.SetMotorDirection(state.Direction.ToMD())

					if hCalls[state.Floor][state.Direction] {
						completedCallC <- state.ToHallCallEvent()
						hCalls[state.Floor][state.Direction] = false
					}
					state.Behaviour = elevstate.Moving
					localStateC <- state
				case elevstate.DoorOpen:
					fmt.Println("doorclose: nextstate dooropen")

					openDoorC <- true
					state.Direction = state.Direction.Opposite()
					completedCallC <- state.ToHallCallEvent()
					hCalls[state.Floor][state.Direction] = false
					state.Behaviour = elevstate.DoorOpen
				case elevstate.Idle:
					state.Behaviour = elevstate.Idle
					fmt.Println("doorclose: nextstate idle")
				default:
					state.Behaviour = elevstate.Idle
				}
			default:
				panic("Door closed in impossible state")
			}
		case syncedVariables = <-syncedVariablesC:
			fmt.Println("main received")

		drainChannel:
			for {
				select {
				case syncedVariables = <-syncedVariablesC:
				default:
					break drainChannel
				}
			}
			cCalls = syncedVariables.CallsBool.CabCallsBool[0]

			var allStates []elevstate.ElevState

			allStates = append(allStates, state)

			for _, otherElevator := range syncedVariables.OtherElevators {
				allStates = append(allStates, otherElevator.State)
			}
			hCalls = sequenceassigner.AssignCalls(allStates, syncedVariables.CallsBool)

			switch state.Behaviour {
			case elevstate.Moving:
				break
			case elevstate.DoorOpen:
				break
			case elevstate.Idle:
				if hCalls.HasCalls() || cCalls.HasCalls() {
					openDoorC <- true
					state.Behaviour = elevstate.DoorOpen
				}
			}
		case <-stopButtonC:
			elevio.SetMotorDirection(elevio.MD_Stop)
			state.Behaviour = elevstate.Idle
		// Debug to monitor state and alive
		case <-time.After(3 * time.Second):
			i++

			fmt.Println(i, "state:", state.Floor, state.Direction, state.Behaviour)
		}
		lights.SetLights(syncedVariables.CallsBool)

	}

}
