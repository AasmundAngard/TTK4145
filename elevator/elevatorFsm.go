package elevator

import (
	"fmt"
	"root/config"
	"root/elevio"
	"root/elevstate"
	"root/elevsync"
	"strconv"
	"time"
)

func drainChannel[T any](variableC <-chan T, variable *T) {
drainChannel:
	for {
		select {
		case *variable = <-variableC:
		default:
			break drainChannel
		}
	}
}

func orderDone(state elevstate.ElevState, hCalls *elevsync.HallCallsBool, cCalls *elevsync.CabCallsBool, completedCallToSyncC chan<- elevio.CallEvent) {
	if cCalls[state.Floor] {
		cCalls[state.Floor] = false
		completedCallToSyncC <- state.ToCabCallEvent()
	}
	if hCalls[state.Floor][state.Direction] && !state.MotorStop && !state.DoorObstructed {
		hCalls[state.Floor][state.Direction] = false
		completedCallToSyncC <- state.ToHallCallEvent()
	}
}

func orderInDirection(direction elevstate.Direction, floor int, hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool) bool {
	if direction == elevstate.Up {
		return requestsAbove(hallCalls, cabCalls, floor)
	} else {
		return requestsBelow(hallCalls, cabCalls, floor)
	}
}
func requestsAbove(hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func requestsBelow(hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool, currentFloor int) bool {
	for f := 0; f < currentFloor; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func Elevator(fsmStateToMainC chan<- elevstate.ElevState, completedCallToSyncC chan<- elevio.CallEvent, callsToElevatorC <-chan elevsync.CommonCalls, hardwareReconnectedC <-chan bool) {

	stopButtonC := make(chan bool, 16)
	floorSensorC := make(chan int, 1)
	openDoorC := make(chan bool, 1)
	doorClosedC := make(chan bool, 1)
	doorObstructedC := make(chan bool, 1)

	go elevio.PollStopButton(stopButtonC)
	go elevio.PollFloorSensor(floorSensorC)
	go Door(openDoorC, doorClosedC, doorObstructedC)

	var hCalls elevsync.HallCallsBool
	var cCalls elevsync.CabCallsBool

	var state elevstate.ElevState
	var prevState elevstate.ElevState
	state.Behaviour = elevstate.Idle
	state.Direction = elevstate.Down

	// Create dormant timer object
	motorTimeoutTimer := time.NewTimer(0)
	if !motorTimeoutTimer.Stop() {
		<-motorTimeoutTimer.C
	}

	// Init
	floor := elevio.GetFloor()
	fmt.Println("startfloor:", floor)
	if floor != -1 {
		state.Floor = <-floorSensorC

	} else {
		elevio.SetMotorDirection(state.Direction.ToMD())
		state.Floor = <-floorSensorC
		elevio.SetMotorDirection(elevio.MD_Stop)
	}
	elevio.SetFloorIndicator(state.Floor)
	var i int = 0 // Debugging
	prevState = state
	prevState.Direction = state.Direction.Opposite()
	fsmStateToMainC <- state

	for {

		select {
		case newFloor := <-floorSensorC:
			switch state.Behaviour {
			case elevstate.Moving:
				state.Floor = newFloor
				elevio.SetFloorIndicator(state.Floor)
				motorTimeoutTimer.Stop()
				state.MotorStop = false
				switch {
				case hCalls[state.Floor][state.Direction] || cCalls[state.Floor]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = elevstate.DoorOpen
				case orderInDirection(state.Direction, state.Floor, hCalls, cCalls):
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
				case hCalls[state.Floor][state.Direction.Opposite()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Direction = state.Direction.Opposite()
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = elevstate.DoorOpen
				case orderInDirection(state.Direction.Opposite(), state.Floor, hCalls, cCalls):
					state.Direction = state.Direction.Opposite()
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = elevstate.Moving
				}

				switch state.Floor {
				case config.NumFloors:
					state.Direction = elevstate.Down
				case 0:
					state.Direction = elevstate.Up
				}

			default:
				fmt.Println("New floor in impossible state:" + strconv.Itoa(int(state.Behaviour)))
				elevio.SetMotorDirection(elevio.MD_Stop)
				motorTimeoutTimer.Stop()
				elevio.SetFloorIndicator(newFloor)
				openDoorC <- true
				state.Behaviour = elevstate.DoorOpen
				orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
			}
		case <-doorClosedC:
			switch state.Behaviour {
			case elevstate.DoorOpen:
				switch {
				case hCalls[state.Floor][state.Direction] || cCalls[state.Floor]:
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = elevstate.DoorOpen
				case orderInDirection(state.Direction, state.Floor, hCalls, cCalls):
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = elevstate.Moving
				case hCalls[state.Floor][state.Direction.Opposite()]:
					state.Direction = state.Direction.Opposite()
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = elevstate.DoorOpen
				case orderInDirection(state.Direction.Opposite(), state.Floor, hCalls, cCalls):
					state.Direction = state.Direction.Opposite()
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = elevstate.Moving
				default:
					state.Behaviour = elevstate.Idle
				}
			default:
				fmt.Println("Illegal state:", strconv.Itoa(int(state.Behaviour)))
				state.Behaviour = elevstate.Idle
			}
		case confirmedCalls := <-callsToElevatorC:
			drainChannel(callsToElevatorC, &confirmedCalls)
			hCalls, cCalls = confirmedCalls.HallCalls, confirmedCalls.CabCalls
			switch state.Behaviour {
			case elevstate.Moving:
				break
			case elevstate.DoorOpen:
				if hCalls[state.Floor][state.Direction] || cCalls[state.Floor] {
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = elevstate.DoorOpen
				}
			case elevstate.Idle:
				switch {
				case hCalls[state.Floor][state.Direction] || cCalls[state.Floor]:
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = elevstate.DoorOpen
				case hCalls[state.Floor][state.Direction.Opposite()]:
					state.Direction = state.Direction.Opposite()
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = elevstate.DoorOpen
				case orderInDirection(state.Direction, state.Floor, hCalls, cCalls):
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = elevstate.Moving
				case orderInDirection(state.Direction.Opposite(), state.Floor, hCalls, cCalls):
					state.Direction = state.Direction.Opposite()
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = elevstate.Moving
				}
			default:
				fmt.Println("Illegal state")
				state.Behaviour = elevstate.Idle
			}

		case <-motorTimeoutTimer.C:
			fmt.Println("Motor timed out")
			state.MotorStop = true
			if elevio.GetFloor() == -1 {
				elevio.SetMotorDirection(state.Direction.ToMD())
				motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
			}
		case doorIsObstructed := <-doorObstructedC:
			state.DoorObstructed = doorIsObstructed
		case <-hardwareReconnectedC:
			fmt.Println("reconnected")
			elevio.SetMotorDirection(elevio.MD_Stop)
			currentFloor := elevio.GetFloor()
			switch {
			case currentFloor == -1:
				elevio.SetMotorDirection(state.Direction.ToMD())
				motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
				state.Behaviour = elevstate.Moving
			default:
				openDoorC <- true
				state.Behaviour = elevstate.DoorOpen
			}

		case <-stopButtonC:
			elevio.SetMotorDirection(elevio.MD_Stop)
			state.Behaviour = elevstate.Moving
			state.MotorStop = true
		// Debug to monitor state and alive
		case <-time.After(3 * time.Second):
			i++
			fmt.Println("fsm", i, "state:", state.Floor, state.Direction, state.Behaviour)
		}

		fsmStateToMainC <- state
	}

}
