package lights

import (
	"root"
)

func setLights(confirmedCalls CallList) {

	for f := 0; f < config.NumFloors; f++ {
		for d := 0; d < 2; d++ {
			if confirmedCalls.hallCalls[2*f + d].needService {
				// Turn hall light f, d on
			}
			else {
				// Turn hall light f, d off
			}
			}
		}
	for f := 0; f < config.NumFloors; f++ {
		if confirmedCalls.cabCalls[f].needService {
			// Turn cablight f on
		}
		else {
			// Turn cablight f off
		}
	}
	}

