package elevator

import (
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
func Elevator(fsmStateToMainC chan<- elevstate.ElevState, completedCallToSyncC chan<- elevio.CallEvent, callsToElevatorC <-chan elevsync.CallsBool, hardwareReconnectedC <-chan bool) {

	stopButtonC := make(chan bool, 16)
	floorSensorC := make(chan int, 1)
	openDoorC := make(chan bool, 1)
	doorClosedC := make(chan bool, 1)
	doorObstructedC := make(chan bool, 1)

	go elevio.PollStopButton(stopButtonC)
	go elevio.PollFloorSensor(floorSensorC)
	go Door(openDoorC, doorClosedC, doorObstructedC)

	// var confirmedCalls elevsync.CallsBool
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
	lights.SetLights(cCalls, hCalls)

	for {

		select {
		case newFloor := <-floorSensorC:
			switch state.Behaviour {
			case elevstate.Moving:
				state.Floor = newFloor
				elevio.SetFloorIndicator(state.Floor)
				motorTimeoutTimer.Stop()
				nextState := sequenceassigner.NextState(hCalls, cCalls, state)
				switch nextState.Behaviour {
				case elevstate.DoorOpen:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Direction = nextState.Direction
					if cCalls[state.Floor] {
						cCalls[state.Floor] = false
						completedCallToSyncC <- state.ToCabCallEvent()
					}
					if hCalls[state.Floor][state.Direction] {
						hCalls[state.Floor][state.Direction] = false
						completedCallToSyncC <- state.ToHallCallEvent()
					}
					state.Behaviour = elevstate.DoorOpen
				case elevstate.Moving:
					state.Direction = nextState.Direction
					elevio.SetMotorDirection(state.Direction.ToMD())
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
					state.Behaviour = elevstate.Moving
				case elevstate.Idle:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = elevstate.Idle
				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = elevstate.Idle
				}

			default:
				fmt.Println("New floor in impossible state:" + strconv.Itoa(int(state.Behaviour)))
				elevio.SetMotorDirection(elevio.MD_Stop)
				motorTimeoutTimer.Stop()
				elevio.SetFloorIndicator(newFloor)
				openDoorC <- true
				state.Behaviour = elevstate.DoorOpen
			}

		case <-doorClosedC:
			switch state.Behaviour {
			case elevstate.DoorOpen:
				nextState := sequenceassigner.NextState(hCalls, cCalls, state)

				switch nextState.Behaviour {
				case elevstate.Moving:
					if state.Direction != nextState.Direction {
						openDoorC <- true
						state.Direction = nextState.Direction
						state.Behaviour = elevstate.DoorOpen
					} else {
						state.Direction = nextState.Direction
						elevio.SetMotorDirection(state.Direction.ToMD())
						motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)

						if hCalls[state.Floor][state.Direction] {
							completedCallToSyncC <- state.ToHallCallEvent()
							hCalls[state.Floor][state.Direction] = false
						}
						state.Behaviour = elevstate.Moving
					}

				case elevstate.DoorOpen:
					openDoorC <- true
					state.Direction = state.Direction.Opposite()
					completedCallToSyncC <- state.ToHallCallEvent()
					hCalls[state.Floor][state.Direction] = false
					state.Behaviour = elevstate.DoorOpen

				case elevstate.Idle:
					state.Behaviour = elevstate.Idle
				default:
					state.Behaviour = elevstate.Idle
				}
			default:
				fmt.Println("Door closed in impossible state:" + strconv.Itoa(int(state.Behaviour)))
				elevio.SetMotorDirection(elevio.MD_Stop)
				motorTimeoutTimer.Stop()
				openDoorC <- true
				state.Behaviour = elevstate.DoorOpen

			}
		case confirmedCalls := <-callsToElevatorC:
			drainChannel(callsToElevatorC, &confirmedCalls)
			hCalls, cCalls = confirmedCalls.HallCallsBool, confirmedCalls.CabCallsBool
			switch state.Behaviour {
			case elevstate.Moving:
				break
			case elevstate.DoorOpen:
				break
			case elevstate.Idle:
				state = sequenceassigner.NextState(hCalls, cCalls, state)
				switch state.Behaviour {
				case elevstate.DoorOpen:
					openDoorC <- true
				case elevstate.Moving:
					elevio.SetMotorDirection(state.Direction.ToMD())
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
				default:
				}
			default:
				break

			}
		case <-motorTimeoutTimer.C:
			fmt.Println("Motor timed out")
			state.Behaviour = elevstate.Motorstop
			if elevio.GetFloor() == -1 {
				elevio.SetMotorDirection(state.Direction.ToMD())
				motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
			}
		case <-stopButtonC:
			elevio.SetMotorDirection(elevio.MD_Stop)
			state.Behaviour = elevstate.Moving
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

		// Debug to monitor state and alive
		case <-time.After(3 * time.Second):
			i++
			fmt.Println("fsm", i, "state:", state.Floor, state.Direction, state.Behaviour)
		}
		lights.SetLights(cCalls, hCalls)

		fsmStateToMainC <- state
	}

}
