package main

import "time"

type Behaviour int

const (
	idle     Behaviour = 0
	moving             = 1
	doorOpen           = 2
)

type Call struct {
	needService bool
	timeStamp   time.Duration
}

type ElevState struct {
	behaviour   Behaviour
	floor       int
	direction   elevio.MotorDirection
}

type CallList struct {
	hallCalls	[]Call // 2n elementer to Call-objekt for hver etasje, en for opp og en for ned: 
	// [Call_etg1_opp, Call_etg1_ned, Call_etg2_opp, Call_etg2_ned, ..., Call_etgn_opp, Call_etgn_ned]
	cabCalls	[config.NumFloors]Call // n elementer, ett Call-objekt for hver etasje
}


func main() {
	// Init Hardware
	elevState ElevState

	elevio.Init("localhost:15657", 3) // Dette er til den lokale heisserveren man kan kjøre (alt. hardware)
	
	// Polle stop-knapp og obstruction
	obstruction := make(chan bool,1)
	stopButton := make(chan bool,1)

	go elevio.PollObstructionSwitch(obstruction)
	go elevio.PollStopButton(stopButton)
	
	floorSensor := make(chan int,1)

	go elevio.PollFloorSensor(floorSensor)
	// Init sync -> Sync starts reading from the network and hardware
	// Lag kanaler som skal passe info fra main

	// Sync <-> main
	// sync -> main: alle heistilstander og hall calls
	// sync <- main: direction and movement
	// sync <- main: completed calls

	confirmedCallsC := make(chan CallList)  // From sync
	localStateC := make(chan ElevState)   // To sync
	completedCallC := make(chan ButtonEvent) // To sync

	go sync(callsC, currStateC, completedCallC)

	// main can poll floorsensor directly

	// Sync should not broadcast before main says so? Maybe uninitialized tag?

	// If between floors -> floor sensor registers no floors, go down until
	// sync registers a floor and says so to main

	// Read ID, num elevators and num floors from config file

	// Read common hall calls and cab calls from other elevators' broadcasts

	// Update own state with new info in sync

	// Main starts receiving common hall calls,
	// Main enters main loop
	obstruction
	stopButton	
	floorSensor
	for {
		switch (elevBehaviour) {
		case moving:
			select {
			case <-stopButton:
				// Stop
			case newFloor:= <-floorSensor:
				currentFloor = a
				if currentFloor in activeFloors {

				}
			case a:= 
			}

		case doorOpen:
			
		case idle:
			

		}

		// Main only receives common hall calls from sync
		// Light up hall buttons based on hall calls
		// Light up cab call buttons based on cab calls

		// Main calls sequenceAsigner, assigning current calls and finding direction

		// Main decides direction based on current state (doorOpen and result from sequenceAssigner)

		// Maybe make decisions based on channels and select case
		// If moving:
		// case a:=<-floorsensor:
		//		if ain hallrequests or a in cabRequests
		//			stopMotor
		// 			direction blir hallrequests retning eller assigner retning
		//
		// case a:=<-stopButton:
		// 		stopMotor
	// 			direction blir samme som før
		// else if doorOpen:
		//	case a:=<-timer:
		// 		if !obstruction && newdirection==olddirection
		// 			close door
		// 		else if obstruction
		// 			opendoor
		// 		else if newdirection != olddirection
		// 			opendoor
		// 			olddirection = newdirection
		// 	else if doorClosed && idle		
		// case 


		// Open door
		// Start timer (select case?)
		// After timer, check door sensor

		// If stopButton -> stop
		// If call at current floor -> stop
		// If shouldMove -> Start actuators
		// If shouldOpenDoor -> start door closing countdown
		// if doorOpen and doorCountdown over -> try closing door

	}
	if 

}
