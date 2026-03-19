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

func Elevator(
	selfStateToMainC chan<- elevstate.ElevState,
	completedCallToSyncC chan<- elevio.CallEvent,
	selfCallsToElevatorC <-chan elevsync.ConfirmedCalls,
	hardwareReconnectedC <-chan bool,
) {

	floorReachedC := make(chan int, 16)
	stopButtonC := make(chan bool, 16)

	openDoorC := make(chan bool, 16)
	doorClosedC := make(chan bool, 16)
	doorObstructedC := make(chan bool, 16)

	go elevio.PollFloorSensor(floorReachedC)
	go elevio.PollStopButton(stopButtonC)
	go Door(openDoorC, doorClosedC, doorObstructedC)

	var hCalls elevsync.ConfirmedHallCalls
	var cCalls elevsync.ConfirmedCabCalls

	state := elevstate.ElevState{Behaviour: elevstate.Idle, Direction: elevstate.Down}

	// Create dormant timer object
	motorTimeoutTimer := time.NewTimer(0)
	if !motorTimeoutTimer.Stop() {
		<-motorTimeoutTimer.C
	}

	var i int = 0 // Debugging

	for {

		select {
		case newFloor := <-floorReachedC:
			fmt.Println("newfloor:", newFloor)
			switch state.Behaviour {
			case elevstate.Moving:
				fmt.Println("newfloor moving")

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
				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = elevstate.Idle
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
				state.Floor = newFloor
				elevio.SetFloorIndicator(state.Floor)
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
		case selfCalls := <-selfCallsToElevatorC:
			DrainChannel(selfCallsToElevatorC, &selfCalls)
			hCalls, cCalls = selfCalls.HallCalls, selfCalls.CabCalls

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
		case doorObstructed := <-doorObstructedC:
			state.DoorObstructed = doorObstructed
		case <-hardwareReconnectedC:
			fmt.Println("hardware reconnected")
			elevio.SetMotorDirection(elevio.MD_Stop)
			currentFloor := elevio.GetFloor()
			switch {
			case currentFloor == -1:
				// Unknown floor, set to legal floor
				state.Floor = 2
				elevio.SetFloorIndicator(state.Floor)
				elevio.SetMotorDirection(state.Direction.ToMD())
				motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
				state.Behaviour = elevstate.Moving
			default:
				state.Floor = currentFloor
				elevio.SetFloorIndicator(state.Floor)
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

		selfStateToMainC <- state
	}

}
