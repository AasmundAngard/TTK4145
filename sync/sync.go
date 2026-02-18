package sync

import (
	"root/elevio"
)

type msgtosync struct {
	hallCalls hallCalls
	cabCalls  cabCalls
}

type hallBools [config.NumFloors][2]bool
type cabBools [config.NumFloors]bool

type msgtomain struct {
	hallBools hallBools
	cabBools  [3]cabBools
	elevatorStates [3]ElevState
}

func Sync(hardwareCalls chan msgtosync, syncedData chan msgtomain) {
	for{
		select {
		}
	}
}
