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
	SenderID 		string
	RequesterID	 	string
	CabCalls 		elevsync.CabCalls
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
			if msg.RequesterID == id {
				if !slices.Contains(collectedIDs, msg.SenderID) {
					collectedCalls = append(collectedCalls, msg.CabCalls)
					collectedIDs = append(collectedIDs, msg.SenderID)
				}
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

func handleCabRequest(cabRequestRxC <- chan string, id string, otherCabCallsRequestC chan <- string, otherCabCallsToNetworkC <- chan elevsync.CabCalls, cabCallsTxC chan <- CabNetworkMsg) {
	for {
		requesterID := <-cabRequestRxC
		if requesterID != id {
			otherCabCallsRequestC <- requesterID

			var cabMsg CabNetworkMsg
			cabCalls := <-otherCabCallsToNetworkC
			cabMsg.CabCalls = cabCalls
			cabMsg.SenderID = id
			cabMsg.RequesterID = requesterID
			for i := 0; i < config.CabCallRetries; i++ {
				cabCallsTxC <- cabMsg
				time.Sleep(config.InitRetryInterval)
			}
		}
	}
}

// <direction><what><purpose>
// (ToSync or FromSync)(SelfData, PeerData, CabCalls etc.)(Request, Update, Broadcast)

func Network(id string,								// SelfId
	networkRequestSelfDataC chan<- struct{}, 		// ToSync_RequestSelfDataC
	selfDataToNetworkC <-chan elevsync.NetworkMsg, 	// FromSync_SelfDataC
	otherDataToSyncC chan<- elevsync.NetworkMsg, 	// ToSync_PeerDataUpdate
	alivePeersC chan<- []string,					// ToSync_AlivePeersC
	otherCabCallsRequestC chan<- string,			// ToSync_CabCallRequest (should include ID)
	otherCabCallsToNetworkC <-chan elevsync.CabCalls, // FromSync_CabCallsForBroadcast
	selfCabCallsToSyncC chan<- []elevsync.CabCalls) { // ToSync_SelfCabCallsUpdate 

	fmt.Println("initializing network")

	peerUpdateRxC := make(chan peers.PeerUpdate)
	peerTxEnableC := make(chan bool)
	go peers.Transmitter(config.PeerUpdatePort, id, peerTxEnableC)
	go peers.Receiver(config.PeerUpdatePort, peerUpdateRxC)

	stateTxC := make(chan elevsync.NetworkMsg)
	stateRxC := make(chan elevsync.NetworkMsg)

	go bcast.Transmitter(config.StateUpdatePort, stateTxC)
	go bcast.Receiver(config.StateUpdatePort, stateRxC)

	cabCallsTxC := make(chan CabNetworkMsg)
	cabRequestRxC := make(chan string)

	go bcast.Transmitter(config.CabCallPort, cabCallsTxC)
	go bcast.Receiver(config.CabRequestPort, cabRequestRxC)

	time.Sleep(time.Second)



	go broadcastState(stateTxC, networkRequestSelfDataC, selfDataToNetworkC)
	
	initElevator(id, selfCabCallsToSyncC)

	go handleCabRequest(cabRequestRxC, id, otherCabCallsRequestC, otherCabCallsToNetworkC, cabCallsTxC)

	fmt.Println("started network")
	for {
		select {
		case peerUpdate := <-peerUpdateRxC:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", peerUpdate.Peers)
			fmt.Printf("  New:      %q\n", peerUpdate.New)
			fmt.Printf("  Lost:     %q\n", peerUpdate.Lost)

			alivePeersC <- peerUpdate.Peers

		case stateUpdate := <-stateRxC:
			if stateUpdate.SenderID != id {
				otherDataToSyncC <- stateUpdate
			}
		}
	}
}
