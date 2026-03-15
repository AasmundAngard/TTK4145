package network

import (
	"flag"
	"fmt"
	"root/config"
	"root/elevstate"
	"root/elevsync"
	"root/network/bcast"
	"root/network/peers"
	"time"
)


type NetworkMsg struct {
	SenderID  string
	Version	  int
	Calls     elevsync.Calls
	State     elevstate.ElevState
}

func initElevator(id string, cabCallsFromNetwork chan<- []elevsync.CabCalls) {
    cabRequestTx := make(chan string)
    cabCallsRx := make(chan elevsync.CabCalls)

    go bcast.Transmitter(config.CabRequestPort, cabRequestTx)
    go bcast.Receiver(config.CabCallPort, cabCallsRx)

    go func() {
		for i := 0; i < config.CabCallRetries; i++ {
			cabRequestTx <- id
			time.Sleep(config.InitRetryInterval)
		}
	}()

    var collected []elevsync.CabCalls

    timeout := time.After(config.InitTimeout)
    for {
        select {
        case cabCalls := <-cabCallsRx:
            collected = append(collected, cabCalls)
        case <-timeout:
            cabCallsFromNetwork <- collected
            return
        }
    }
}


func broadcastState(bcastChannel chan<- NetworkMsg, requestStatus chan<- struct{}, stateToNetwork <- chan NetworkMsg) {
	for {
		requestStatus <- struct{}{}

		status := <- stateToNetwork

		status.Version += 1
		bcastChannel <- status
		time.Sleep(config.BroadcastTime)
	}
}

func handleCabRequest(cabRequestRx <- chan string, )




func Network (id string, requestState chan<- struct{}, stateToNetwork <- chan NetworkMsg, stateFromNetwork chan<- NetworkMsg, currentPeers chan<- []string, cabCallsRequest chan<- string, cabCallsToNetwork <- chan elevsync.CabCalls, cabCallsFromNetwork chan<- []elevsync.CabCalls) {

	fmt.Println("initializing network")

	// 1. TODO: Assign ports and ID (if a runtime flag has a certain value, set id by smth that lets you run multiple processes on one machine)
	flag.StringVar(&id, "id", "", "Elevator ID (for running multiple instances on one machine)")
	flag.Parse()

	if id == "" {
		// default behavior for normal single-process use
		id = "elevator_1" // or derive from hostname, etc.
	}

	// 2. Make channels for peerupdate and setup transmit/recv updates
	peerUpdateChRx := make(chan peers.PeerUpdate)
	peerTxEnable := make(chan bool)
	go peers.Transmitter(config.PeerUpdatePort, id, peerTxEnable)
	go peers.Receiver(config.PeerUpdatePort, peerUpdateChRx)

	// 3. Make channels for sending and recieving status (NetworkTransmitMsg) and setup transmit/recv status
	stateTx := make(chan NetworkMsg)
	stateRx := make(chan NetworkMsg)

	go bcast.Transmitter(config.StateUpdatePort, stateTx)
	go bcast.Receiver(config.StateUpdatePort, stateRx)

	// 4. Make channels for recieving cabCall requests and sending cabCalls, and recv/transmit
	cabCallsTx := make(chan elevsync.CabCalls)
	cabRequestRx := make(chan string)

	go bcast.Transmitter(config.CabCallPort, cabCallsTx)
	go bcast.Receiver(config.CabRequestPort, cabRequestRx)

	// Initialize (ask for cab calls)
	initElevator(id, cabCallsFromNetwork)

	// Dillemma: Need to broadcast status at set intervals, but the rest should just be a loop that collects responses from channels
	// 5. (Solulu): make a function for bcasting status that is its own thread duh
	go broadcastState(stateTx, requestState, stateToNetwork)


	// 6. Make a loop w./ select/case that listens to the channels for updates:


	go func() {
		for {
			requesterID := <-cabRequestRx
			fmt.Printf("Requested cab calls for id=%#v\n", requesterID)
			cabCallsRequest <- requesterID

			cabCalls := <-cabCallsToNetwork
			for i := 0; i < config.CabCallRetries; i++ {
				cabCallsTx <- cabCalls
				time.Sleep(config.InitRetryInterval)
			}
		}
	}()

	fmt.Println("started network")
	for {
		select {
		// case peerUpdate (aka smth has happened in the peer department):
			// put all current-peer ids into a []string and send on currentPeers channel
		case peerUpdate := <-peerUpdateChRx:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", peerUpdate.Peers)
			fmt.Printf("  New:      %q\n", peerUpdate.New)
			fmt.Printf("  Lost:     %q\n", peerUpdate.Lost)

			currentPeers <- peerUpdate.Peers

		// case NetworkTransmitMsg (recieved a status from the network)
			// create a NetworkRecieveMsg and add the info from NetworkTransmitMsg into it, plus sender id
			// Send on recvFromNetwork channel
		case stateUpdate := <-stateRx:
			fmt.Printf("Received: %#v\n", stateUpdate)
			stateFromNetwork <- stateUpdate

		}
	}
}
