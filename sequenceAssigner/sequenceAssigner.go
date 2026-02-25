package sequenceAssigner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
)

// Asks for labtid:
// - Definerer vi nederste etasje som 0 eller 1? - 0
// - Se over oppførsel når requests_here()

//encoding/json for translation for input and output .exe file
// Use json.Marshal and json.Unmarshal

//os/exec for running the executable

// TEMP (disse må en annen plass etterhvert):
type Behaviour int

const (
	idle     Behaviour = 0
	moving             = 1
	doorOpen           = 2
)

type Direction int

const (
	Up     Direction = 0
	Down             = 1
)


type HallCallsBool [config.NumFloors][2]bool
type CabCallsBool [config.NumFloors]bool
type CallsBool struct {
	HallCallsBool HallCallsBool
	CabCallsBool  [config.NumElevators]CabCallsBool
}

type ElevState struct {
	behaviour   Behaviour
	floor       int
	direction   Direction
}


// JSON input and output structure
type assignerState struct {
	Behaviour 	string 	`json:"behaviour"`
	Floor		int		`json:"floor"`
	Direction 	string 	`json:"direction"`
	CabRequests	[]bool	`json:"cabRequests"`
}

type assignerInput struct {
	HallRequests	[config.NumFloors][2]bool					`json:"hallRequests"`
	States			map[string]assignerState	`json:"states"`
}

func requestsAbove(hallCalls HallCallsBool, cabCalls CabCallsBool, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]){
			return true
		}
	}
	return false
}

func requestsBelow(hallCalls HallCallsBool, cabCalls CabCallsBool, currentFloor int) bool {
	for f:= 0; f < currentFloor; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]){
			return true
		}
	}
	return false
}


func assignCalls(allStates [config.NumElevators]ElevState, allCalls CallsBool) HallCallsBool {
	execFile := ""

	switch runtime.GOOS {
	case "linux": 		execFile = "utils/hall_request_assigner"
	case "windows":		execFile = "utils/hall_request_assigner.exe"
	default:			panic("OS not supported.")
	}

	hallRequests 	:= allCalls.HallCallsBool
	states 			:= make(map[string]assignerState)

	for i := 0; i < config.NumElevators; i++ {
		tempState := assignerState {
			Behaviour: allStates[i].behaviour,
			Floor: allStates[i].floor,
			Direction: allStates[i].direction,
			CabRequests: allCalls.CabCallsBool[i],
		}
		states[strconv.Itoa(i)] = tempState
	}

	input := assignerInput {
		HallRequests: hallRequests,
		States: states,
	}

	jsonInput, err := json.Marshal(input)
	if err != nil {
		fmt.Println("Problem with json.Marshal(): ", err)
		panic(err)
	}

	assignerCmd, err := exec.Command("../"+execFile, "-i", string(jsonInput)).CombinedOutput()
	if err != nil {
		fmt.Println("Problem with exec.Command: ", err)
		panic(err)
	}

	jsonOutput := new(map[string][config.NumFloors][2]bool)
	err = json.Unmarshal(assignerCmd, &jsonOutput)
	if err != nil {
		fmt.Println("Problem with json.Unmarshal: ", err)
		panic(err)
	}

	return *jsonOutput["1"]
}

/*
// Returns next direction and behaviour based on call-requests and current direction and floor
func nextState(hallCalls HallCallsBool, cabCalls CabCallsBool, currentState ElevState) ElevState {
	// Decide which direction to go (up, down , stop) and behaviour (idle, moving, doorOpen)
	// Return direction and behaviour instructions
	var nextState ElevState
	// Inspired by the elevator algorithim in the project resources
	switch currentState.direction {
	case elevio.MD_Up:
		switch {
		case requestsAbove(hallCalls, cabCalls, currentState.floor):
			return elevio.MD_Up // Moving upwards, call(s) above
		case requestsBelow(hallCalls, cabCalls, currentState.floor):
			return elevio.MD_Down // Moving upwards, call(s) below
		default:
			return elevio.MD_Stop // No requests (or some other undefined behaviour that does not crash)
		}

	case elevio.MD_Down:
		switch {
		case requestsBelow(hallCalls, cabCalls, currentState.floor):
			return elevio.MD_Down
		case requestsAbove(hallCalls, cabCalls, currentState.floor):
			return elevio.MD_Up
		default:
			return elevio.MD_Stop
		}

	case elevio.MD_Stop:
		switch {
		case requestsAbove(hallCalls, cabCalls, currentState.floor):
			return elevio.MD_Up
		case requestsBelow(hallCalls, cabCalls, currentState.floor):
			return elevio.MD_Down
		default:
			return elevio.MD_Stop
		}
	default:
		return elevio.MD_Stop // elevio.Direction somehow neither Stop, Up or Down, aka. funkiness afoot
	}
}
*/
