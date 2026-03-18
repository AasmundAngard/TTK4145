package network

import (
	"fmt"
	"root/config"
	"root/elevsync"
	"root/network/bcast"
	"root/network/peers"
	"slices"
	"time"
)

func initElevator(id string, selfCabCallsToSyncC chan<- []elevsync.CabCalls) {
	cabRequestTxC := make(chan string)
	cabCallsRxC := make(chan elevsync.CabNetworkMsg)

	// For sending ID and requesting local cab calls
	go bcast.Transmitter(config.CabRequestPort, cabRequestTxC)
	// For receiving cab call restorations from other elevators
	go bcast.Receiver(config.CabCallPort, cabCallsRxC)

	var collectedCalls []elevsync.CabCalls
	var collectedIDs []string

	timeout := time.After(config.InitTimeout)
	fmt.Println("init elevator network?")
	for len(collectedIDs) < config.NumElevators {
		fmt.Println("initElevatorloop")
		select {
		case msg := <-cabCallsRxC:
			fmt.Println("received")
			fmt.Println(msg)
			if msg.RequesterID == id {
				if !slices.Contains(collectedIDs, msg.SenderID) {
					fmt.Println("append msg")
					collectedCalls = append(collectedCalls, msg.CabCalls)
					fmt.Println("collectedCalls")
					fmt.Println(collectedCalls)
					collectedIDs = append(collectedIDs, msg.SenderID)
				}
			}

		case <-timeout:
			fmt.Println("init timeout")
			selfCabCallsToSyncC <- collectedCalls
			return
		default:
			fmt.Println("init default")
			cabRequestTxC <- id
			time.Sleep(config.InitRetryInterval)
		}
	}
	selfCabCallsToSyncC <- collectedCalls
}

func broadcastState(stateTxC chan<- elevsync.NetworkMsg, requestStatusC chan<- struct{}, selfDataToNetworkC <-chan elevsync.NetworkMsg) {
	for {
		requestStatusC <- struct{}{}

		status := <-selfDataToNetworkC
		// fmt.Println("broadcasting")
		// fmt.Println(status.Calls.HallCalls)

		stateTxC <- status
		time.Sleep(config.BroadcastTime)
	}
}

// func handleCabRequest(cabRequestRx <- chan string, )

func Network(id string,
	networkRequestSelfDataC chan<- struct{},
	selfDataToNetworkC <-chan elevsync.NetworkMsg,
	otherDataToSyncC chan<- elevsync.NetworkMsg,
	alivePeersC chan<- []string,
	otherCabCallsRequestC chan<- string,
	otherCabCallsToNetworkC <-chan elevsync.CabNetworkMsg,
	selfCabCallsToSyncC chan<- []elevsync.CabCalls) {

	fmt.Println("initializing network")

	// 1. TODO: Assign ports and ID (if a runtime flag has a certain value, set id by smth that lets you run multiple processes on one machine)
	// flag.StringVar(&id, "id", "", "Elevator ID (for running multiple instances on one machine)")
	// flag.Parse()

	if id == "" {
		// default behavior for normal single-process use
		id = "elevator_1" // or derive from hostname, etc.
	}

	// 2. Make channels for peerupdate and setup transmit/recv updates
	peerUpdateRxC := make(chan peers.PeerUpdate)
	peerTxEnableC := make(chan bool)
	go peers.Transmitter(config.PeerUpdatePort, id, peerTxEnableC)
	go peers.Receiver(config.PeerUpdatePort, peerUpdateRxC)

	// 3. Make channels for sending and recieving status (NetworkTransmitMsg) and setup transmit/recv status
	stateTxC := make(chan elevsync.NetworkMsg)
	stateRxC := make(chan elevsync.NetworkMsg)

	go bcast.Transmitter(config.StateUpdatePort, stateTxC)
	go bcast.Receiver(config.StateUpdatePort, stateRxC)

	// 4. Make channels for recieving cabCall requests and sending cabCalls, and recv/transmit
	cabCallsTxC := make(chan elevsync.CabNetworkMsg)
	cabRequestRxC := make(chan string)

	go bcast.Transmitter(config.CabCallPort, cabCallsTxC)
	go bcast.Receiver(config.CabRequestPort, cabRequestRxC)

	time.Sleep(time.Second)
	// Initialize (ask for cab calls)
	fmt.Println("init network start")
	initElevator(id, selfCabCallsToSyncC)
	fmt.Println("init network end")

	go broadcastState(stateTxC, networkRequestSelfDataC, selfDataToNetworkC)
	fmt.Println("broadcast state")

	// Dillemma: Need to broadcast status at set intervals, but the rest should just be a loop that collects responses from channels
	// 5. (Solulu): make a function for bcasting status that is its own thread duh

	// 6. Make a loop w./ select/case that listens to the channels for updates:

	go func() {
		for {
			requesterID := <-cabRequestRxC
			// Received ID asking for cab calls
			if requesterID != id {
				fmt.Println("not local id")
				otherCabCallsRequestC <- requesterID
			}
		}
	}()
	go func() {
		for {
			select {
			case cabCallMsg := <-otherCabCallsToNetworkC:
				// Denne metoden "låser" cab calls som skal sendes for en stund.
				// Om en heis connecter, krasjer og reconnecter innen kort tid,
				// fmt.Println("calls to send:")
				// fmt.Println(cabCalls)
				// var cabMsg elevsync.CabNetworkMsg
				// cabMsg.CabCalls = cabCalls
				// cabMsg.SenderID = id
				go broadCastCabCalls(cabCallMsg, cabCallsTxC)
			default:
				break

			}
		}
	}()

	fmt.Println("started network")
	for {
		select {
		// case peerUpdate (aka smth has happened in the peer department):
		// put all current-peer ids into a []string and send on currentPeers channel
		case peerUpdate := <-peerUpdateRxC:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", peerUpdate.Peers)
			fmt.Printf("  New:      %q\n", peerUpdate.New)
			fmt.Printf("  Lost:     %q\n", peerUpdate.Lost)

			alivePeersC <- peerUpdate.Peers

		// case NetworkTransmitMsg (recieved a status from the network)
		// create a NetworkRecieveMsg and add the info from NetworkTransmitMsg into it, plus sender id
		// Send on recvFromNetwork channel
		case stateUpdate := <-stateRxC:
			// fmt.Println("stateupdate:", stateUpdate.SenderID)
			if stateUpdate.SenderID != id {
				otherDataToSyncC <- stateUpdate
				// fmt.Println("received other")
				// fmt.Println(stateUpdate.Calls.HallCalls)
			}
		}
	}
}

func broadCastCabCalls(cabMsg elevsync.CabNetworkMsg, cabCallsTxC chan<- elevsync.CabNetworkMsg) {
	for i := 0; i < config.CabCallRetries; i++ {
		cabCallsTxC <- cabMsg
		time.Sleep(config.InitRetryInterval)
	}
}
