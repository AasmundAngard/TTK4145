package elevator

import (
	"root/config"
	"root/elevio"
	"time"
)

type DoorState int

const (
	Closed DoorState = iota
	InCountDown
	Obstructed
)

func Door(
	doorClosedC		chan<- bool,
	doorOpenC 		<-chan bool,
	obstrucedC 		chan<- bool,
	){

	elevio.SetDoorOpenLamp(false)
	obstructionC := make(chan bool)
	go elevio.PollObstructionSwitch(obstructionC)

	obstruction := false
	timeCounter := time.NewTimer(time.Hour)
	ds := Closed
	timeCounter.Stop()

	for {
		select {
		case obstruction = <-obstructionC:
			if !obstruction && ds == Obstructed {
				elevio.SetDoorOpenLamp(false)
				doorClosedC <- true
				ds = Closed
			}
			if obstruction {
				obstrucedC <- true
			} else {
				obstrucedC <- false
			}

		case <-doorOpenC:
			if obstruction {
				obstrucedC <- true
			}
			switch ds {
			case Closed:
				elevio.SetDoorOpenLamp(true)
				timeCounter = time.NewTimer(config.DoorOpenDuration)
				ds = InCountDown
			case InCountDown:
				timeCounter = time.NewTimer(config.DoorOpenDuration)

			case Obstructed:
				timeCounter = time.NewTimer(config.DoorOpenDuration)
				ds = InCountDown

			default:
				panic("Door state not implemented")
			}
		case <-timeCounter.C:
			if ds != InCountDown {
				panic("Door state not implemented")
			}
			if obstruction {
				ds = Obstructed
			} else {
				elevio.SetDoorOpenLamp(false)
				doorClosedC <- true
				ds = Closed
			}
		}
	}
}
