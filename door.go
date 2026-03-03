package main //Foreløpig bare

import (
	"root/config"
	"root/elevio"
	"time"
)

// Note to self:
// Døra har egen intern timer og egen definert logikk og funksjonalitet, så gir mye mening å programmere
// som egen statemachine, inkludert separere logikk
// Vi sier bare når døra skal åpne, så sier døra selv når den klarer å lukke
// Døra varsler om problemer (obstruction) den får
// Dørlukking designes ikke for brukervennlighet, men for å fylle kravene og å være rask.
// Legg til ekstra timer ved obstruction for brukervennlighet

type DoorState int

const (
	Closed DoorState = 0
	OpenCountdown
	OpenWaiting
)

func Door(
	openDoorC <-chan bool,
	doorClosedC chan<- bool,
	doorObstructedC chan<- bool,
) {
	obstructedC := make(chan bool, 1)
	go elevio.PollObstructionSwitch(obstructedC)

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	doorState := Closed
	obstructed := false

	select {
	case obstructed := <-obstructedC:

		if !obstructed && doorState == OpenWaiting {
			elevio.SetDoorOpenLamp(false)
			doorState = Closed
			doorClosedC <- true
		}

		doorObstructedC <- obstructed

	case <-openDoorC:
		elevio.SetDoorOpenLamp(true)
		timer = time.NewTimer(config.DoorOpenTime)
		doorState = OpenCountdown

		if obstructed {
			doorObstructedC <- true
		}

	case <-timer.C:
		switch doorState {
		case OpenCountdown:
			if obstructed {
				doorState = OpenWaiting
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
