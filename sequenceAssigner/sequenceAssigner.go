package sequenceassigner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"root/config"
	"root/elevstate"
	"root/elevsync"
	"runtime"
	"strconv"
)

//encoding/json for translation for input and output .exe file
// Use json.Marshal and json.Unmarshal

//os/exec for running the executable

// JSON input and output structure
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

func requestsAbove(hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func requestsBelow(hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool, currentFloor int) bool {
	for f := 0; f < currentFloor; f++ {
		if (hallCalls[f][0]) || (hallCalls[f][1]) || (cabCalls[f]) {
			return true
		}
	}
	return false
}

func requestsHere(hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool, currentFloor int) bool {
	if hallCalls[currentFloor][0] || hallCalls[currentFloor][1] || cabCalls[currentFloor] {
		return true
	}
	return false
}

func cabAbove(cabCalls elevsync.CabCallsBool, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if cabCalls[f] {
			return true
		}
	}
	return false
}

func cabBelow(cabCalls elevsync.CabCallsBool, currentFloor int) bool {
	for f := 0; f < currentFloor; f++ {
		if cabCalls[f] {
			return true
		}
	}
	return false
}

func AssignCalls(allStates []elevstate.ElevState, allCalls elevsync.CallsBool) elevsync.HallCallsBool {
	execFile := ""

	switch runtime.GOOS {
	case "linux":
		execFile = "utils/hall_request_assigner"
	case "windows":
		execFile = "utils/hall_request_assigner.exe"
	default:
		panic("OS not supported.")
	}

	hallRequests := allCalls.HallCallsBool
	states := make(map[string]assignerState)

	for i := range allStates {
		tempState := assignerState{
			Behaviour:   allStates[i].Behaviour.String(),
			Floor:       allStates[i].Floor,
			Direction:   allStates[i].Direction.String(),
			CabRequests: allCalls.CabCallsBool[i],
		}
		states[strconv.Itoa(i)] = tempState
	}

	input := assignerInput{
		HallRequests: hallRequests,
		States:       states,
	}

	jsonInput, err := json.Marshal(input)
	if err != nil {
		fmt.Println("Problem with json.Marshal(): ", err)
		panic(err)
	}

	assignerCmd, err := exec.Command("./sequenceAssigner/"+execFile, "-i", string(jsonInput)).CombinedOutput()
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

	return (*jsonOutput)["0"]
}

// Returns next state (direction and behaviour) based on call-requests and current direction and floor
func NextState(hallCalls elevsync.HallCallsBool, cabCalls elevsync.CabCallsBool, currentState elevstate.ElevState) elevstate.ElevState {
	var nextState elevstate.ElevState
	nextState.Floor = currentState.Floor
	// Inspired by the elevator algorithim in the project resources
	switch currentState.Direction {
	case elevstate.Up:
		switch {
		case requestsHere(hallCalls, cabCalls, currentState.Floor) && !(currentState.Behaviour == elevstate.DoorOpen):
			nextState.Behaviour = elevstate.DoorOpen

			switch {
			case hallCalls[currentState.Floor][elevstate.Up]:
				nextState.Direction = elevstate.Up
			case hallCalls[currentState.Floor][elevstate.Down] && !cabAbove(cabCalls, currentState.Floor):
				nextState.Direction = elevstate.Down
			default:
				nextState.Direction = elevstate.Up
			}
		case requestsAbove(hallCalls, cabCalls, currentState.Floor):
			fmt.Println("requestabove")
			nextState.Direction = elevstate.Up // Moving upwards, call(s) above
			nextState.Behaviour = elevstate.Moving
		case requestsBelow(hallCalls, cabCalls, currentState.Floor):
			fmt.Println("requestbelow")
			nextState.Direction = elevstate.Down // Moving upwards, call(s) below
			nextState.Behaviour = elevstate.Moving
		default:
			nextState.Direction = elevstate.Up
			nextState.Behaviour = elevstate.Idle
		}

	case elevstate.Down:
		switch {
		case requestsHere(hallCalls, cabCalls, currentState.Floor) && !(currentState.Behaviour == elevstate.DoorOpen):
			nextState.Behaviour = elevstate.DoorOpen

			switch {
			case hallCalls[currentState.Floor][elevstate.Down]:
				nextState.Direction = elevstate.Down
			case hallCalls[currentState.Floor][elevstate.Up] && !cabBelow(cabCalls, currentState.Floor):
				nextState.Direction = elevstate.Up
			default:
				nextState.Direction = elevstate.Down
			}
		case requestsBelow(hallCalls, cabCalls, currentState.Floor):
			nextState.Direction = elevstate.Down
			nextState.Behaviour = elevstate.Moving
		case requestsAbove(hallCalls, cabCalls, currentState.Floor):
			nextState.Direction = elevstate.Up
			nextState.Behaviour = elevstate.Moving
		default:
			nextState.Direction = elevstate.Down
			nextState.Behaviour = elevstate.Idle
		}

	default:
		nextState.Behaviour = elevstate.Idle // elevio.Direction somehow neither Stop, Up or Down, aka. funkiness afoot
		nextState.Direction = elevstate.Up
	}

	return nextState
}
