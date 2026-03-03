package sequenceAssigner

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"root/config"
	"runtime"
	"strconv"
)

//encoding/json for translation for input and output .exe file
// Use json.Marshal and json.Unmarshal

//os/exec for running the executable

// TEMP (disse må en annen plass etterhvert):
type Behaviour int

const (
	idle     Behaviour = 0
	moving   Behaviour = 1
	doorOpen Behaviour = 2
)

func (b Behaviour) String() string {
	switch b {
	case idle:
		return "idle"
	case moving:
		return "moving"
	case doorOpen:
		return "doorOpen"
	default:
		fmt.Println("Behaviour not recognized.")
		panic(b)
	}
}

type Direction int

const (
	Up     Direction = 0
	Down   Direction = 1
)

func (d Direction) String() string {
	switch d {
	case Up:
		return "up"
	case Down:
		return "down"
	default:
		fmt.Println("Direction not recognized.")
		panic(d)
	}
}


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
	CabRequests	[config.NumFloors]bool	`json:"cabRequests"`
}

type assignerInput struct {
	HallRequests	[config.NumFloors][2]bool	`json:"hallRequests"`
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

func requestsHere(hallCalls HallCallsBool, cabCalls CabCallsBool, currentFloor int) bool {
	if hallCalls[currentFloor][0] || hallCalls[currentFloor][1] || cabCalls[currentFloor]{
		return true
	}
	return false
}

func cabAbove(cabCalls CabCallsBool, currentFloor int) bool {
	for f := currentFloor + 1; f < config.NumFloors; f++ {
		if cabCalls[f] {
			return true
		}
	}
	return false
}

func cabBelow(cabCalls CabCallsBool, currentFloor int) bool {
	for f:= 0; f < currentFloor; f++ {
		if cabCalls[f] {
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
			Behaviour:  allStates[i].behaviour.String(),
			Floor: allStates[i].floor,
			Direction: allStates[i].direction.String(),
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

	return (*jsonOutput)["1"]
}

// Returns next state (direction and behaviour) based on call-requests and current direction and floor
func nextState(hallCalls HallCallsBool, cabCalls CabCallsBool, currentState ElevState) ElevState {
	var nextState ElevState
	nextState.floor = currentState.floor
	// Inspired by the elevator algorithim in the project resources
	switch currentState.direction {
	case Up:
		switch {
		case requestsHere(hallCalls, cabCalls, currentState.floor) && !(currentState.behaviour == doorOpen):
			nextState.behaviour = doorOpen

			switch {
			case hallCalls[currentState.floor][Up]:
				nextState.direction = Up
			case hallCalls[currentState.floor][Down] && !cabAbove(cabCalls, currentState.floor):
				nextState.direction = Down
			default:
				nextState.direction = Up
			} 
		case requestsAbove(hallCalls, cabCalls, currentState.floor):
			nextState.direction = Up // Moving upwards, call(s) above
			nextState.behaviour = moving
		case requestsBelow(hallCalls, cabCalls, currentState.floor):
			nextState.direction = Down // Moving upwards, call(s) below
			nextState.behaviour = moving
		default:
			nextState.direction = Up
			nextState.behaviour = idle
		}

	case Down:
		switch {
		case requestsHere(hallCalls, cabCalls, currentState.floor) && !(currentState.behaviour == doorOpen):
			nextState.behaviour = doorOpen

			switch {
			case hallCalls[currentState.floor][Down]:
				nextState.direction = Down
			case hallCalls[currentState.floor][Up] && !cabBelow(cabCalls, currentState.floor):
				nextState.direction = Up
			default:
				nextState.direction = Down
			} 
		case requestsBelow(hallCalls, cabCalls, currentState.floor):
			nextState.direction = Down
			nextState.behaviour = moving
		case requestsAbove(hallCalls, cabCalls, currentState.floor):
			nextState.direction = Up
			nextState.behaviour = moving
		default:
			nextState.direction = Down
			nextState.behaviour = idle
		}

	default:
		nextState.behaviour = idle // elevio.Direction somehow neither Stop, Up or Down, aka. funkiness afoot
		nextState.direction = Up
	}

	return nextState
}

