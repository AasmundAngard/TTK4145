package network

import (
	"flag"
	"fmt"
	"root/config"
	"root/elevsync"
	"root/network/bcast"
	"root/network/peers"
	"time"
)


func initElevator(id string, selfCabCallsToSyncC chan<- []elevsync.CabCalls) {
	cabRequestTxC := make(chan string)
	cabCallsRxC := make(chan elevsync.CabCalls)

	go bcast.Transmitter(config.CabRequestPort, cabRequestTxC)
	go bcast.Receiver(config.CabCallPort, cabCallsRxC)

	go func() {
		for i := 0; i < config.CabCallRetries; i++ {
			cabRequestTxC <- id
			time.Sleep(config.InitRetryInterval)
		}
	}()

	var collected []elevsync.CabCalls

	timeout := time.After(config.InitTimeout)
	for {
		select {
		case cabCalls := <-cabCallsRxC:
			collected = append(collected, cabCalls)
		case <-timeout:
			selfCabCallsToSyncC <- collected
			return
		}
	}
}

func broadcastState(stateTxC chan<- elevsync.NetworkMsg, requestStatusC chan<- struct{}, selfDataToNetworkC <-chan elevsync.NetworkMsg) {
	for {
		requestStatusC <- struct{}{}

		status := <-selfDataToNetworkC

		status.Version += 1
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
	flag.StringVar(&id, "id", "", "Elevator ID (for running multiple instances on one machine)")
	flag.Parse()

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
	cabCallsTxC := make(chan elevsync.CabCalls)
	cabRequestRxC := make(chan string)

	go bcast.Transmitter(config.CabCallPort, cabCallsTxC)
	go bcast.Receiver(config.CabRequestPort, cabRequestRxC)

	// Initialize (ask for cab calls)
	initElevator(id, selfCabCallsToSyncC)

	// Dillemma: Need to broadcast status at set intervals, but the rest should just be a loop that collects responses from channels
	// 5. (Solulu): make a function for bcasting status that is its own thread duh
	go broadcastState(stateTxC, networkRequestSelfDataC, selfDataToNetworkC)

	// 6. Make a loop w./ select/case that listens to the channels for updates:

	go func() {
		for {
			requesterID := <-cabRequestRxC
			fmt.Printf("Requested cab calls for id=%#v\n", requesterID)
			otherCabCallsRequestC <- requesterID

			cabCalls := <-otherCabCallsToNetworkC
			for i := 0; i < config.CabCallRetries; i++ {
				cabCallsTxC <- cabCalls
				time.Sleep(config.InitRetryInterval)
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
			fmt.Printf("Received: %#v\n", stateUpdate)
			otherDataToSyncC <- stateUpdate

		}
	}
}
