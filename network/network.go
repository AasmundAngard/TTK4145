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

type CabNetworkMsg struct {
	SenderID string
	CabCalls elevsync.CabCalls
}

func initElevator(id string, selfCabCallsToSyncC chan<- []elevsync.CabCalls) {
	cabRequestTxC := make(chan string)
	cabCallsRxC := make(chan CabNetworkMsg)

	go bcast.Transmitter(config.CabRequestPort, cabRequestTxC)
	go bcast.Receiver(config.CabCallPort, cabCallsRxC)

	var collectedCalls []elevsync.CabCalls
	var collectedIDs []string

	timeout := time.After(config.InitTimeout)

	for len(collectedIDs) < config.NumElevators {
		select {
		case msg := <-cabCallsRxC:
			if !slices.Contains(collectedIDs, msg.SenderID) {
				collectedCalls = append(collectedCalls, msg.CabCalls)
				collectedIDs = append(collectedIDs, msg.SenderID)
			}
		case <-timeout:
			selfCabCallsToSyncC <- collectedCalls
			return
		default:
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
	otherCabCallsToNetworkC <-chan elevsync.CabCalls,
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
	cabCallsTxC := make(chan CabNetworkMsg)
	cabRequestRxC := make(chan string)

	go bcast.Transmitter(config.CabCallPort, cabCallsTxC)
	go bcast.Receiver(config.CabRequestPort, cabRequestRxC)

	time.Sleep(time.Second)
	// Initialize (ask for cab calls)
	go broadcastState(stateTxC, networkRequestSelfDataC, selfDataToNetworkC)

	initElevator(id, selfCabCallsToSyncC)

	// Dillemma: Need to broadcast status at set intervals, but the rest should just be a loop that collects responses from channels
	// 5. (Solulu): make a function for bcasting status that is its own thread duh

	// 6. Make a loop w./ select/case that listens to the channels for updates:

	go func() {
		for {
			requesterID := <-cabRequestRxC
			if requesterID != id {
				otherCabCallsRequestC <- requesterID

				var cabMsg CabNetworkMsg
				cabCalls := <-otherCabCallsToNetworkC
				cabMsg.CabCalls = cabCalls
				cabMsg.SenderID = id
				for i := 0; i < config.CabCallRetries; i++ {
					cabCallsTxC <- cabMsg
					time.Sleep(config.InitRetryInterval)
				}
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
			if stateUpdate.SenderID != id {
				otherDataToSyncC <- stateUpdate
			}
		}
	}
}
