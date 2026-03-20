package elevator

import (
	"fmt"
	"root/config"
	"root/elevio"
	s "root/elevstate"
	"root/elevsync"
	"strconv"
	"time"
)

// Elevator handles all the elevator logic.
//
// It implements a finite state machine that receives assigned calls,
// controls the elevator movement and door, and interacts with hardware.
// Elevator practically runs as a single-elevator algorithm.
//
// Input:
// 	 <- selfCallsToElevatorC	: Receives calls to be serviced by the elevator
//
// Output:
// 	 -> completedCallToSyncC	: Reports its serviced calls to sync.
// 	 -> selfStateToMainC		: Passes its local state to main.
//
// Responsible for all hardware IO except button lights, and delegates
// door timing and obstruction handling to the Door routine.

func Elevator(
	selfStateToMainC chan<- s.ElevState,
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

	state := s.ElevState{Behaviour: s.Idle, Direction: s.Down}

	// Create dormant timer object
	motorTimeoutTimer := time.NewTimer(0)
	if !motorTimeoutTimer.Stop() {
		<-motorTimeoutTimer.C
	}

	for {

		select {
		case newFloor := <-floorReachedC:
			switch state.Behaviour {
			case s.Moving:
				state.Floor = newFloor
				elevio.SetFloorIndicator(state.Floor)
				motorTimeoutTimer.Stop()
				state.MotorStop = false
				switch {
				case hCalls[state.Floor][state.Direction] || cCalls[state.Floor]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					clearCall(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = s.DoorOpen
				case callInDirection(state.Direction, state.Floor, hCalls, cCalls):
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
				case hCalls[state.Floor][state.Direction.Opposite()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Direction = state.Direction.Opposite()
					clearCall(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = s.DoorOpen
				case callInDirection(state.Direction.Opposite(), state.Floor, hCalls, cCalls):
					state.Direction = state.Direction.Opposite()
					elevio.SetMotorDirection(state.Direction.ToMD())
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
					state.Behaviour = s.Moving
				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					state.Behaviour = s.Idle
				}

				switch state.Floor {
				case config.NumFloors:
					state.Direction = s.Down
				case 0:
					state.Direction = s.Up
				}

			default:
				fmt.Println("New floor in impossible state:" + strconv.Itoa(int(state.Behaviour)))
				elevio.SetMotorDirection(elevio.MD_Stop)
				motorTimeoutTimer.Stop()
				state.Floor = newFloor
				elevio.SetFloorIndicator(state.Floor)
				openDoorC <- true
				state.Behaviour = s.DoorOpen
				clearCall(state, &hCalls, &cCalls, completedCallToSyncC)
			}
		case <-doorClosedC:
			switch state.Behaviour {
			case s.DoorOpen:
				switch {
				case hCalls[state.Floor][state.Direction] || cCalls[state.Floor]:
					clearCall(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = s.DoorOpen
				case callInDirection(state.Direction, state.Floor, hCalls, cCalls):
					elevio.SetMotorDirection(state.Direction.ToMD())
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
					state.Behaviour = s.Moving
				case hCalls[state.Floor][state.Direction.Opposite()]:
					state.Direction = state.Direction.Opposite()
					clearCall(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = s.DoorOpen
				case callInDirection(state.Direction.Opposite(), state.Floor, hCalls, cCalls):
					state.Direction = state.Direction.Opposite()
					elevio.SetMotorDirection(state.Direction.ToMD())
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
					state.Behaviour = s.Moving
				default:
					state.Behaviour = s.Idle
				}
			default:
				fmt.Println("Door closed in illegal state:", strconv.Itoa(int(state.Behaviour)))
				state.Behaviour = s.Idle
			}
		case selfCalls := <-selfCallsToElevatorC:
			DrainChannel(selfCallsToElevatorC, &selfCalls)
			hCalls, cCalls = selfCalls.HallCalls, selfCalls.CabCalls

			switch state.Behaviour {
			case s.Moving:
				break
			case s.DoorOpen:
				if hCalls[state.Floor][state.Direction] || cCalls[state.Floor] {
					clearCall(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = s.DoorOpen
				}
			case s.Idle:
				switch {
				case hCalls[state.Floor][state.Direction] || cCalls[state.Floor]:
					clearCall(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = s.DoorOpen
				case hCalls[state.Floor][state.Direction.Opposite()]:
					state.Direction = state.Direction.Opposite()
					clearCall(state, &hCalls, &cCalls, completedCallToSyncC)
					openDoorC <- true
					state.Behaviour = s.DoorOpen
				case callInDirection(state.Direction, state.Floor, hCalls, cCalls):
					elevio.SetMotorDirection(state.Direction.ToMD())
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
					state.Behaviour = s.Moving
				case callInDirection(state.Direction.Opposite(), state.Floor, hCalls, cCalls):
					state.Direction = state.Direction.Opposite()
					elevio.SetMotorDirection(state.Direction.ToMD())
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
					state.Behaviour = s.Moving
				}
			default:
				fmt.Println("Illegal state")
				state.Behaviour = s.Idle
			}

		case <-motorTimeoutTimer.C:
			fmt.Println("Motor timed out - motorstop detected")
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
			motorTimeoutTimer.Stop()
			currentFloor := elevio.GetFloor()
			switch {
			case currentFloor == -1:
				// Unknown floor, set to legal floor, avoid impossible state
				state.Floor = 0
				state.Direction = s.Up
				elevio.SetFloorIndicator(state.Floor)
				elevio.SetMotorDirection(state.Direction.ToMD())
				motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
				state.MotorStop = true
				state.Behaviour = s.Moving
			default:
				state.Floor = currentFloor
				elevio.SetFloorIndicator(state.Floor)
				openDoorC <- true
				state.Behaviour = s.DoorOpen
			}

		case <-stopButtonC:
			elevio.SetMotorDirection(elevio.MD_Stop)
			state.Behaviour = s.Moving
			state.MotorStop = true
		}

		selfStateToMainC <- state
	}

}
