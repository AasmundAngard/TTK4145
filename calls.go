package main // Foreløpig

import (
	"root/config"
	"time"
)

type Call struct {
	NeedService bool
	TimeStamp   time.Duration
}

type CallList struct {
	HallCalls [config.NumFloors][2]Call // 2n elementer to Call-objekt for hver etasje, en for opp og en for ned:
	// [Call_etg1_opp, Call_etg1_ned, Call_etg2_opp, Call_etg2_ned, ..., Call_etgn_opp, Call_etgn_ned]
	CabCalls [config.NumElevators][config.NumFloors]Call // n elementer, ett Call-objekt for hver etasje
}

type HallCallsBool [config.NumFloors][2]bool
type CabCallsBool [config.NumFloors]bool

func (cCalls CabCallsBool) isEmpty() bool {
	for _, call := range cCalls {
		if call {
			return false
		}
	}
	return true
}
func (hCalls HallCallsBool) isEmpty() bool {
	for _, floor := range hCalls {
		for _, direction := range floor {
			if direction {
				return false
			}
		}
	}
	return true
}

func (cList CallList) toBool() (HallCallsBool, [config.NumElevators]CabCallsBool) {
	var hCallsBool HallCallsBool
	var cCallsBool [config.NumElevators]CabCallsBool
	hCalls := cList.HallCalls
	cCalls := cList.CabCalls
	for floorIndex, floor := range hCalls {
		for dirIndex, dir := range floor {
			hCallsBool[floorIndex][dirIndex] = dir.NeedService
		}
	}
	for elevatorIndex, elevator := range cCalls {
		for floorIndex, floor := range elevator {
			cCallsBool[elevatorIndex][floorIndex] = floor.NeedService
		}

	}
	return hCallsBool, cCallsBool
}
