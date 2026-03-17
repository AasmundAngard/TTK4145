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

	MotorTimeoutTime  = 5 * time.Second
	DisconnectTime    = 1 * time.Second
	DoorOpenTime      = 3 * time.Second
	BroadcastTime     = 10 * time.Millisecond
	InitTimeout       = 2 * time.Second
	InitRetryInterval = 400 * time.Millisecond
<<<<<<< HEAD
	// SyncTimeout       = 1 * time.Second
	SyncTimeout = 100 * time.Millisecond
=======
	//SyncTimeout       = 1 * time.Second
	SyncTimeout = 500 * time.Millisecond
>>>>>>> ae6303bcf974f8c56bcf36b734af8962499ca805
)
