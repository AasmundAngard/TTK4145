package sequenceassigner

// Package sequenceassigner delegates hall-call assignments across the
// distributed elevator system.
//
// It prepares a JSON representation of all active elevator states
// and current hall requests, then invokes an external assignment executable
// to compute which elevator should serve each hall call. The module filters
// out elevators that cannot participate (e.g., obstructed or stopped), runs
// the assigner, and returns the resulting hall-call allocation.

import (
	"encoding/json"
	"os"
	"os/exec"
	"root/config"
	"root/elevsync"
	"runtime"
)

type assignerState struct {
	Behaviour         string                 `json:"behaviour"`
	Floor             int                    `json:"floor"`
	Direction         string                 `json:"direction"`
	ConfirmedCabCalls [config.NumFloors]bool `json:"cabRequests"`
}

type assignerInput struct {
	ConfirmedHallCalls [config.NumFloors][2]bool `json:"hallRequests"`
	States             map[string]assignerState  `json:"states"`
}

func AssignCalls(
	allStates []elevsync.ConfirmedPeerElevator,
	hallCalls elevsync.ConfirmedHallCalls) elevsync.ConfirmedHallCalls {

	execFile := ""

	switch runtime.GOOS {
	case "linux":
		execFile = "sequenceassigner/utils/hall_request_assigner"
	case "windows":
		execFile = "sequenceassigner/utils/hall_request_assigner.exe"
	default:
		panic("OS not supported.")
	}

	os.Chmod(execFile, 0700)

	hallRequests := hallCalls
	states := make(map[string]assignerState)

	for i := range allStates {
		if allStates[i].State.MotorStop || allStates[i].State.DoorObstructed {
			continue
		}
		tempState := assignerState{
			Behaviour:         allStates[i].State.Behaviour.String(),
			Floor:             allStates[i].State.Floor,
			Direction:         allStates[i].State.Direction.String(),
			ConfirmedCabCalls: allStates[i].CabCalls,
		}
		states[allStates[i].Id] = tempState
	}

	if len(states) == 0 {
		return elevsync.ConfirmedHallCalls{}
	}

	input := assignerInput{
		ConfirmedHallCalls: hallRequests,
		States:             states,
	}

	jsonInput, _ := json.Marshal(input)

	assignerCmd, _ := exec.Command("./"+execFile, "-i", string(jsonInput)).CombinedOutput()

	var jsonOutput map[string][config.NumFloors][2]bool
	json.Unmarshal(assignerCmd, &jsonOutput)

	return (jsonOutput)[allStates[0].Id]
}
