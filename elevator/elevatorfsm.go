package elevator

import (
	"fmt"
	"root/config"
	"root/elevio"
	"strconv"
	"time"
)

func Elevator(
	selfStateToMainC chan<- ElevState,
	completedCallToSyncC chan<- elevio.CallEvent,
	selfCallsToElevatorC <-chan Calls,
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

	var hCalls HallCallsBool
	var cCalls CabCallsBool

	state := ElevState{Behaviour: Idle, Direction: Down}

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
			case Moving:
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
					state.Behaviour = DoorOpen
				case orderInDirection(state.Direction, state.Floor, hCalls, cCalls):
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
				case hCalls[state.Floor][state.Direction.Opposite()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Direction = state.Direction.Opposite()
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = DoorOpen
				case orderInDirection(state.Direction.Opposite(), state.Floor, hCalls, cCalls):
					state.Direction = state.Direction.Opposite()
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = Moving
				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = Idle
				}

				switch state.Floor {
				case config.NumFloors:
					state.Direction = Down
				case 0:
					state.Direction = Up
				}

			default:
				fmt.Println("New floor in impossible state:" + strconv.Itoa(int(state.Behaviour)))
				elevio.SetMotorDirection(elevio.MD_Stop)
				motorTimeoutTimer.Stop()
				state.Floor = newFloor
				elevio.SetFloorIndicator(state.Floor)
				openDoorC <- true
				state.Behaviour = DoorOpen
				orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
			}
		case <-doorClosedC:
			switch state.Behaviour {
			case DoorOpen:
				switch {
				case hCalls[state.Floor][state.Direction] || cCalls[state.Floor]:
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = DoorOpen
				case orderInDirection(state.Direction, state.Floor, hCalls, cCalls):
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = Moving
				case hCalls[state.Floor][state.Direction.Opposite()]:
					state.Direction = state.Direction.Opposite()
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = DoorOpen
				case orderInDirection(state.Direction.Opposite(), state.Floor, hCalls, cCalls):
					state.Direction = state.Direction.Opposite()
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = Moving
				default:
					state.Behaviour = Idle
				}
			default:
				fmt.Println("Illegal state:", strconv.Itoa(int(state.Behaviour)))
				state.Behaviour = Idle
			}
		case localCalls := <-selfCallsToElevatorC:
			DrainChannel(selfCallsToElevatorC, &localCalls)
			hCalls, cCalls = localCalls.HallCalls, localCalls.CabCalls

			switch state.Behaviour {
			case Moving:
				break
			case DoorOpen:
				if hCalls[state.Floor][state.Direction] || cCalls[state.Floor] {
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = DoorOpen
				}
			case Idle:
				switch {
				case hCalls[state.Floor][state.Direction] || cCalls[state.Floor]:
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = DoorOpen
				case hCalls[state.Floor][state.Direction.Opposite()]:
					state.Direction = state.Direction.Opposite()
					orderDone(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = DoorOpen
				case orderInDirection(state.Direction, state.Floor, hCalls, cCalls):
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = Moving
				case orderInDirection(state.Direction.Opposite(), state.Floor, hCalls, cCalls):
					state.Direction = state.Direction.Opposite()
					elevio.SetMotorDirection(state.Direction.ToMD())
					state.Behaviour = Moving
				}
			default:
				fmt.Println("Illegal state")
				state.Behaviour = Idle
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
				state.Behaviour = Moving
			default:
				state.Floor = currentFloor
				elevio.SetFloorIndicator(state.Floor)
				openDoorC <- true
				state.Behaviour = DoorOpen
			}

		case <-stopButtonC:
			elevio.SetMotorDirection(elevio.MD_Stop)
			state.Behaviour = Moving
			state.MotorStop = true
		// Debug to monitor state and alive
		case <-time.After(3 * time.Second):
			i++
			fmt.Println("fsm", i, "state:", state.Floor, state.Direction, state.Behaviour)
		}

		selfStateToMainC <- state
	}

}
