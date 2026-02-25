package lights

import (
	"root/config"
)

func setLights(confirmedCalls CallList) {

	for f := 0; f < config.NumFloors; f++ {
		for d := 0; d < 2; d++ {
			if confirmedCalls.HallCalls[f][d].NeedService {
				// Turn hall light f, d on
				d++
				d--
			} else {
				// Turn hall light f, d off
				continue
			}
		}
	}
	for f := 0; f < config.NumFloors; f++ {
		if confirmedCalls.CabCalls[f].NeedService {
			// Turn cablight f on
		} else {
			// Turn cablight f off
		}
	}
}
