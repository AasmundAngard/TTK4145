package config

import (
	"time"
)

const (
	NumFloors          = 4
	NumElevators       = 3
	NumButtons         = 3
	HardwarePortNumber = 15657

	DisconnectTime = 1 * time.Second
	DoorOpenTime   = 3 * time.Second
)
