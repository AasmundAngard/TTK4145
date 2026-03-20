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
	"root/elevator"
	"root/elevsync"
	"runtime"
)

type assignerState struct {
	Behaviour   string                 `json:"behaviour"`
	Floor       int                    `json:"floor"`
	Direction   string                 `json:"direction"`
	CabRequests [config.NumFloors]bool `json:"cabRequests"`
}

type assignerInput struct {
	HallRequests [config.NumFloors][2]bool `json:"hallRequests"`
	States       map[string]assignerState  `json:"states"`
}

func AssignCalls(
	allStates []elevsync.OtherElevatorBool,
	hallCalls elevator.HallCallsBool) elevator.HallCallsBool {

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
			Behaviour:   allStates[i].State.Behaviour.String(),
			Floor:       allStates[i].State.Floor,
			Direction:   allStates[i].State.Direction.String(),
			CabRequests: allStates[i].CabCallsBool,
		}
		states[allStates[i].ID] = tempState
	}

	if len(states) == 0 {
		return elevator.HallCallsBool{}
	}

	input := assignerInput{
		HallRequests: hallRequests,
		States:       states,
	}

	jsonInput, _ := json.Marshal(input)

	assignerCmd, _ := exec.Command("./"+execFile, "-i", string(jsonInput)).CombinedOutput()

	var jsonOutput map[string][config.NumFloors][2]bool
	json.Unmarshal(assignerCmd, &jsonOutput)

	return (jsonOutput)[allStates[0].ID]
}
