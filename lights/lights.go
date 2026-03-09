package lights

import (
	"root/config"
	"root/elevio"
	"root/elevsync"
)

func SetLights(cCalls elevsync.CabCallsBool, hCalls elevsync.HallCallsBool) {
	for floor := 0; floor < config.NumFloors; floor++ {
		elevio.SetButtonLamp(elevio.BT_HallUp, floor, hCalls[floor][0])
		elevio.SetButtonLamp(elevio.BT_HallDown, floor, hCalls[floor][1])
		elevio.SetButtonLamp(elevio.BT_Cab, floor, cCalls[floor])
	}
}
