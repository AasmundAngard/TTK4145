package sequenceassigner

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"root/config"
	"root/elevsync"
	"runtime"
)

//encoding/json for translation for input and output .exe file
// Use json.Marshal and json.Unmarshal

//os/exec for running the executable

// JSON input and output structure
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

func requestsAbove(hallCalls elevsync.ConfirmedHallCalls, cabCalls elevsync.ConfirmedCabCalls, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func requestsBelow(hallCalls elevsync.ConfirmedHallCalls, cabCalls elevsync.ConfirmedCabCalls, currentFloor int) bool {
	for f := 0; f < currentFloor; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func requestsHere(hallCalls elevsync.ConfirmedHallCalls, cabCalls elevsync.ConfirmedCabCalls, currentFloor int) bool {
	if hallCalls[currentFloor][0] || hallCalls[currentFloor][1] || cabCalls[currentFloor] {
		return true
	}
	return false
}

func cabAbove(cabCalls elevsync.ConfirmedCabCalls, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if cabCalls[f] {
			return true
		}
	}
	return false
}

func cabBelow(cabCalls elevsync.ConfirmedCabCalls, currentFloor int) bool {
	for f := 0; f < currentFloor; f++ {
		if cabCalls[f] {
			return true
		}
	}
	return false
}

func AssignCalls(allStates []elevsync.ConfirmedPeerElevator, hallCalls elevsync.ConfirmedHallCalls) elevsync.ConfirmedHallCalls {
	execFile := ""

	switch runtime.GOOS {
	case "linux":
		execFile = "sequenceassigner/utils/hall_request_assigner"
	case "windows":
		execFile = "sequenceassigner/utils/hall_request_assigner.exe"
	default:
		panic("OS not supported.")
	}

	err := os.Chmod(execFile, 0700)
	if err != nil {
		fmt.Println("Error with file permissions: ", err)
		panic(err)
	}

	hallRequests := hallCalls
	states := make(map[string]assignerState)

	// fmt.Println("All hallcalls:")
	// for _, floor := range hallCalls {
	// 	fmt.Println(floor[0], floor[1])
	// }
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

	jsonInput, err := json.Marshal(input)
	if err != nil {
		fmt.Println("Problem with json.Marshal(): ", err)
		panic(err)
	}

	assignerCmd, err := exec.Command("./"+execFile, "-i", string(jsonInput)).CombinedOutput()
	if err != nil {
		fmt.Println("Problem with exec.Command: ", err)
		panic(err)
	}

	var jsonOutput map[string][config.NumFloors][2]bool
	err = json.Unmarshal(assignerCmd, &jsonOutput)
	if err != nil {
		fmt.Println("Problem with json.Unmarshal: ", err)
		panic(err)
	}
	// for elevnum, elev := range jsonOutput {
	// 	fmt.Println("Heis nummer:", elevnum)
	// 	for _, floor := range elev {
	// 		fmt.Println(floor[0], floor[1])
	// 	}
	// }
	return (jsonOutput)[allStates[0].Id]
}
