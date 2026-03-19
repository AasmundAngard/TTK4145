package elevator

import (
	"root/config"
	"root/elevio"
	"time"
)

// Door controls the elevator door.
//
// It listens for "open door" command, monitors obstructions,
// and controls the door lamp and timer internally.
//
// Input:
// 		<-openDoorC: Command to open door or restart door timer
// Output:
// 		doorClosedC<-: Reports when door closed successfully
// 		doorObstructedC<-true: Alerts when timer expired, but door failed to close because of an obstruction
// 		doorObstructedC<-false: Notifies when obstruction clears after door previously failed to close

type DoorState int

const (
	Closed        DoorState = 0
	OpenCountdown           = 1
	OpenWaiting             = 2
)

func Door(
	openDoorC <-chan bool,
	doorClosedC chan<- bool,
	doorObstructedC chan<- bool,
) {
	obstructedC := make(chan bool, 1)
	go elevio.PollObstructionSwitch(obstructedC)

	// Create dormant timer object
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	doorState := Closed
	obstructed := false
	for {

		select {
		case obstructed = <-obstructedC:
			if !obstructed && doorState == OpenWaiting {
				elevio.SetDoorOpenLamp(false)
				doorState = Closed
				doorClosedC <- true
				doorObstructedC <- false
			}

		case <-openDoorC:
			elevio.SetDoorOpenLamp(true)
			timer = time.NewTimer(config.DoorOpenTime)
			doorState = OpenCountdown
		case <-timer.C:
			switch doorState {
			case OpenCountdown:
				if obstructed {
					doorState = OpenWaiting
					doorObstructedC <- true
				} else {
					elevio.SetDoorOpenLamp(false)
					doorClosedC <- true
					doorState = Closed
				}
			default:
				panic("Timer ended in illegal state")
			}
		}
	}
}
