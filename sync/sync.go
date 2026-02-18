package sync

import(
	root/config
)

type Call struct {
	needService bool
	timeStamp   time
}

type hallCallsType 	[config.NumFloors][2]Call
type cabCallsType 	[config.NumFloors]Call

type hallBoolsType 	[config.NumFloors][2]bool
type cabBoolsType 	[config.NumFloors]bool


type hardwareCallsType struct {
	hallCalls hallCallsType
	cabCalls cabCallsType
}


type syncedDataType sturct {
	hallBools hallBoolsType
	cabBools cabBoolsType
}


func sync (hardwareCalls chan hardwareCallsType, syncedData chan syncedDataType) {
	var localCabCalls cabCallsType
	var localHallCalls hallCallsType

	for {
		select{
		case incomingHardwareCalls := <-hardwareCalls:
			
		}
	}
}