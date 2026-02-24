package main

import (
	"flag"
	"root/config"
	"root/elevio"
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

func nextDirection(state ElevState, cl CallList) Direction {
	// COMES FROM SEQUENCEASSIGNER? NEED TO FIX
	return Down
}

func main() {

	idPtr := flag.Int("id", 0, "ID of elevator, overwrite using -id=<newId>")
	portPtr := flag.Int("fork", 20026, "Port of the elevator, overwrite using -port=<newPort>")
	flag.Parse()

	id := *idPtr
	port := *portPtr

	elevio.Init("localhost:"+strconv.Itoa(port), config.NumFloors) // Dette er til den lokale heisserveren man kan kjøre (alt. hardware)

	// Init Hardware
	// Lag kanaler som skal sende info fra main
	// Init sync -> Sync leser fra network og hardware

	stopButton := make(chan bool, 1)
	go elevio.PollStopButton(stopButton)

	floorSensor := make(chan int, 1)
	go elevio.PollFloorSensor(floorSensor)

	openDoorC := make(chan bool)
	doorClosedC := make(chan bool)
	doorObstructedC := make(chan bool)
	go Door(openDoorC, doorClosedC, doorObstructedC)

	// Sync <-> main
	// sync -> main: alle heistilstander og hall calls
	// sync <- main: direction and movement
	// sync <- main: completed calls

	confirmedCallsC := make(chan CallList)          // From sync
	localStateC := make(chan ElevState)             // To sync
	completedCallC := make(chan elevio.ButtonEvent) // To sync

	go sync(confirmedCallsC, localStateC, completedCallC)

	// main can poll floorsensor directly

	// Sync should not broadcast before main says so? Maybe uninitialized tag?

	// If between floors -> floor sensor registers no floors, go down until
	// sync registers a floor and says so to main

	// Read ID, num elevators and num floors from config file

	// Read common hall calls and cab calls from other elevators' broadcasts

	// Update own state with new info in sync

	// Main starts receiving common hall calls,
	// Main enters main loop

	// Siden handling/output i systemet (fra fsm) er basert på inputet, og ikke nødvendigvis state,
	// bør systemet implementeres som en handlings-sentrert Mealy-maskin.

	// Det krever større fokus på kanaler som varsler fsm-en om handligner, som doorClosed,
	// floorSensor (newFloor?) o.l.

	var state ElevState
	var calls CallList
	// var hCalls [config.NumFloors][2]bool
	// var cCalls [config.NumFloors]bool
	var hCalls HallCalls
	var cCalls CabCalls
	hCalls, AllCabCalls := calls.toBool()
	cCalls = AllCabCalls[0]

	for {

		select {
		case newFloor := <-floorSensor:
			state.Floor = newFloor
			elevio.SetFloorIndicator(state.Floor)
			switch state.Behaviour {
			case Moving:
				// Dersom egne distribuerte hall call i etasjen:
				// Stans motor
				// Åpne dør
				// Dersom egne cab calls i etasjen
				// Stans motor
				// Åpne dør
				// Marker cab call som ferdig!

				// Sequence assigner burde ta inn state og confirmedCalls, og gi hvilken retning den bør ta fra etasjen
				switch {
				case cCalls[state.Floor] && hCalls[state.Floor][state.Direction]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Behaviour = DoorOpen

					completedCallC <- state.toCabButtonEvent()
					cCalls[state.Floor] = false

				case cCalls[state.Floor] && hCalls[state.Floor][state.Direction.Opposite()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Direction = state.Direction.Opposite()
					state.Behaviour = DoorOpen

					completedCallC <- state.toCabButtonEvent()
					cCalls[state.Floor] = false

				case cCalls[state.Floor]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Behaviour = DoorOpen

					completedCallC <- state.toCabButtonEvent()
					cCalls[state.Floor] = false

				case hCalls[state.Floor][state.Direction]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Behaviour = DoorOpen

				case hCalls[state.Floor][state.Direction.Opposite()]:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Behaviour = DoorOpen
					state.Direction = state.Direction.Opposite()

				case nextDirection(state, calls) == state.Direction:
					// keep going
				default:
					elevio.SetMotorDirection(elevio.MD_Stop)
					openDoorC <- true
					state.Behaviour = DoorOpen

				}
			default:
				panic("Impossible state")
			}

		case <-doorClosedC:
			switch state.Behaviour {
			case DoorOpen:
				// Hvordan skal man bestemme retningen? Sier sequenceAssigner?
				// Foreløpig: sequenceAssigner/nextDirection bestemmer retning
				switch {
				case cCalls.empty() && hCalls.empty():
					state.Behaviour = Idle
				case state.Direction == nextDirection(state, calls):
					elevio.SetMotorDirection(state.Direction.toMD())
					state.Behaviour = Moving
					completedCallC <- state.toHallButtonEvent()
					hCalls[state.Floor][state.Direction] = false
				default:
					state.Direction = state.Direction.Opposite()
					completedCallC <- state.toHallButtonEvent()
					hCalls[state.Floor][state.Direction] = false
					openDoorC <- true

				}
				// Dersom sameDirection(state.Direction, assignedSequence)
				// Sett motorretning til state.Direction
				// Klarér hall call i samme retning
				state.Behaviour = Moving

				//Dersom IKKE sameDirection(state.Direction, assignedSequence)
				// Åpne døra en gang til, og annonser retningsendring
				state.Direction = state.Direction.Opposite()
				// Sett retningsindikator elevio
				openDoorC <- true
				state.Behaviour = DoorOpen

			default:
				panic("Door closed in impossible state")
			}
		case calls = <-confirmedCallsC:

		case <-stopButton:
			state.Behaviour = Idle
			elevio.SetMotorDirection(elevio.MD_Stop)
		}

		localStateC <- state

		// Main only receives common hall calls from sync
		// Light up hall buttons based on hall calls
		// Light up cab call buttons based on cab calls
		// Main calls sequenceAsigner, assigning current calls and finding direction

	}

}
