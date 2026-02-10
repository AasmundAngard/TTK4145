package lights

import (
	"root/config"
	"root/distributor"
	"root/elevio"
)

func SetLights(cs distributor.CommonState, ElevatorID int) {
	for f := 0; f < config.NumFloors; f++ {
		for b := 0; b < 2; b++ {
			if cs.HallRequests[f][b] {
				elevio.SetButtonLamp(elevio.ButtonType(b), f, true)
			} else {
				elevio.SetButtonLamp(elevio.ButtonType(b), f, false)
			}

		}
	}
	for f := 0; f < config.NumFloors; f++ {
		if cs.States[ElevatorID].CabRequests[f] {
			elevio.SetButtonLamp(elevio.BT_Cab, f, true)
		} else {
			elevio.SetButtonLamp(elevio.BT_Cab, f, false)
		}

	}
}
