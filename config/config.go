package config

import (
	"time"
)

const (
	NumFloors          = 4
	NumElevators       = 3
	NumButtons         = 3
	HardwarePortNumber = 15657

	CabRequestPort  = 16571
	CabCallPort     = 16572
	PeerUpdatePort  = 15147
	StateUpdatePort = 16569
	CabCallRetries  = 10

	StartupWait = 5 * time.Second

	MotorTimeoutTime  = 5 * time.Second
	DisconnectTime    = 1 * time.Second
	DoorOpenTime      = 3 * time.Second
	BroadcastTime     = 25 * time.Millisecond
	InitTimeout       = 2 * time.Second
	InitRetryInterval = 400 * time.Millisecond
)
