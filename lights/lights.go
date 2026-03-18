package lights

import (
	"root/config"
	"root/elevator"
	"root/elevio"
)

func Lights(commonCallsToLightsC <-chan elevator.Calls) {
	for {
		calls := <-commonCallsToLightsC
		for floor := 0; floor < config.NumFloors; floor++ {
			elevio.SetButtonLamp(elevio.BT_HallUp, floor, calls.HallCalls[floor][0])
			elevio.SetButtonLamp(elevio.BT_HallDown, floor, calls.HallCalls[floor][1])
			elevio.SetButtonLamp(elevio.BT_Cab, floor, calls.CabCalls[floor])
		}
	}

}
