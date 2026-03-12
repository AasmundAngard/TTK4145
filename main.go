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

func drainChannel[T any](variableC chan T, variable *T) {
drainChannel:
	for {
		select {
		case *variable = <-variableC:
		default:
			break drainChannel
		}
	}
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
	networkMsgC := make(chan elevsync.NetworkReceiveMsg, 16)
	syncedVariablesC := make(chan elevsync.SyncedData, 16)

	// For network -> sync
	cabCallRequestOnInitC := make(chan string, 16)
	cabCallReceiveOnInitC := make(chan elevsync.CabCallsList, 16)
	cabCallSendOnRequestC := make(chan elevsync.CabCalls, 16)

	go elevio.PollStopButton(stopButtonC)
	go elevio.PollFloorSensor(floorSensorC)
	go elevio.PollButtons(hardWareCallsC)
	go Door(openDoorC, doorClosedC, doorObstructedC)
	go elevsync.Sync(
		hardWareCallsC,
		localStateC,
		completedCallC,
		networkMsgC,
		syncedVariablesC,
		cabCallRequestOnInitC,
		cabCallReceiveOnInitC,
		cabCallSendOnRequestC,
	)

	var syncedVariables elevsync.SyncedData
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

	for {

		select {
		case newFloor := <-floorSensorC:
			fmt.Println("newfloor")
			state.Floor = newFloor
			elevio.SetFloorIndicator(state.Floor)
			motorTimeoutTimer.Stop()
			fmt.Println("Stopped timer")
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
					if hCalls[state.Floor][state.Direction] {
						hCalls[state.Floor][state.Direction] = false
						completedCallC <- state.ToHallCallEvent()
					}
					state.Behaviour = elevstate.DoorOpen
					localStateC <- state
				case elevstate.Moving:
					state.Direction = nextState.Direction
					elevio.SetMotorDirection(state.Direction.ToMD())
					motorTimeoutTimer = time.NewTimer(config.MotorTimeoutTime)
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
				panic("New floor in impossible state:" + strconv.Itoa(int(state.Behaviour)))
			}

		case <-doorClosedC:
			switch state.Behaviour {
			case elevstate.DoorOpen:
				nextState := sequenceassigner.NextState(hCalls, cCalls, state)
				switch nextState.Behaviour {
				case elevstate.Moving:

					if state.Direction != nextState.Direction {
						fmt.Println("change direction")
						openDoorC <- true
						state.Direction = nextState.Direction
						state.Behaviour = elevstate.DoorOpen
					} else {
						fmt.Println("doorclose, moving")

						state.Direction = nextState.Direction
						elevio.SetMotorDirection(state.Direction.ToMD())

						if hCalls[state.Floor][state.Direction] {
							completedCallC <- state.ToHallCallEvent()
							hCalls[state.Floor][state.Direction] = false
						}
						state.Behaviour = elevstate.Moving
						localStateC <- state
					}

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
				panic("Door closed in impossible state" + strconv.Itoa(int(state.Behaviour)))
			}
		case syncedVariables = <-syncedVariablesC:
			fmt.Println("main received")
			drainChannel(syncedVariablesC, &syncedVariables)

			cCalls = syncedVariables.LocalCabCalls

			localState := elevsync.OtherElevatorBool{State: state, CabCallsBool: cCalls}
			allStates := append(
				[]elevsync.OtherElevatorBool{localState},
				syncedVariables.OtherElevatorListBool...,
			)
			hCalls = sequenceassigner.AssignCalls(allStates, syncedVariables.SyncedHallCalls)
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

			}
		case <-motorTimeoutTimer.C:
			fmt.Println("Motor timed out")
			state.Behaviour = elevstate.Motorstop
			localStateC <- state
		case <-stopButtonC:
			elevio.SetMotorDirection(elevio.MD_Stop)
			state.Behaviour = elevstate.Idle
		// Debug to monitor state and alive
		case <-time.After(3 * time.Second):
			i++

			fmt.Println(i, "state:", state.Floor, state.Direction, state.Behaviour)
		}
		lights.SetLights(cCalls, hCalls)

	}

}
