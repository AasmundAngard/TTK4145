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

func initElevator(selfId string, selfCabCallsToSyncC chan<- []elevsync.CabCalls) {
	cabRequestTxC := make(chan string)
	cabCallsRxC := make(chan elevsync.CabNetworkMsg)

	go bcast.Transmitter(config.CabRequestPort, cabRequestTxC)
	go bcast.Receiver(config.CabCallPort, cabCallsRxC)

	var collectedCalls []elevsync.CabCalls
	var collectedIDs []string

	timeout := time.After(config.InitTimeout)
	for len(collectedIDs) < config.NumElevators {
		select {
		case cabMsg := <-cabCallsRxC:
			if cabMsg.RequesterID == selfId {
				if !slices.Contains(collectedIDs, cabMsg.SenderID) {
					collectedCalls = append(collectedCalls, cabMsg.CabCalls)
					collectedIDs = append(collectedIDs, cabMsg.SenderID)
				}
			}

		case <-timeout:
			selfCabCallsToSyncC <- collectedCalls
			return

		default:
			cabRequestTxC <- selfId
			time.Sleep(config.InitRetryInterval)
		}
	}
	selfCabCallsToSyncC <- collectedCalls
}

func broadcastStatus(
	statusTxC chan<- elevsync.NetworkMsg, 
	requestStatusC chan<- struct{}, 
	selfStatusToNetworkC <-chan elevsync.NetworkMsg) {
	for {
		requestStatusC <- struct{}{}

		selfStatus := <-selfStatusToNetworkC

		statusTxC <- selfStatus
		time.Sleep(config.BroadcastTime)
	}
}

func broadcastCabCalls(cabMsg elevsync.CabNetworkMsg, cabCallsTxC chan<- elevsync.CabNetworkMsg) {
	for i := 0; i < config.CabCallRetries; i++ {
		cabCallsTxC <- cabMsg
		time.Sleep(config.InitRetryInterval)
	}
}

func Network(
	selfId string, 
	selfStatusRequestToSyncC chan<- struct{},  
	selfStatusToNetworkC <-chan elevsync.NetworkMsg, 
	peerStatusUpdateToSyncC chan<- elevsync.NetworkMsg, 
	alivePeersToSyncC chan<- []string, 
	peerRequestCabCallsToSyncC chan<- string, 
	peerCabCallsToNetworkC <-chan elevsync.CabNetworkMsg, 
	selfCabCallsToSyncC chan<- []elevsync.CabCalls) { 

	peerUpdateRxC := make(chan peers.PeerUpdate)
	peerTxEnableC := make(chan bool)
	go peers.Transmitter(config.PeerUpdatePort, selfId, peerTxEnableC)
	go peers.Receiver(config.PeerUpdatePort, peerUpdateRxC)

	statusTxC := make(chan elevsync.NetworkMsg)
	statusRxC := make(chan elevsync.NetworkMsg)
	go bcast.Transmitter(config.StateUpdatePort, statusTxC)
	go bcast.Receiver(config.StateUpdatePort, statusRxC)

	cabCallsTxC := make(chan elevsync.CabNetworkMsg)
	cabRequestRxC := make(chan string)
	go bcast.Transmitter(config.CabCallPort, cabCallsTxC)
	go bcast.Receiver(config.CabRequestPort, cabRequestRxC)

	time.Sleep(time.Second)

	initElevator(selfId, selfCabCallsToSyncC)

	go broadcastStatus(statusTxC, selfStatusRequestToSyncC, selfStatusToNetworkC)

	// Send request for cab calls to sync
	go func() {  
		for {
			requesterID := <-cabRequestRxC
			if requesterID != selfId {
				peerRequestCabCallsToSyncC <- requesterID
			}
		}
	}()

	// Receive requested cab calls from sync and broadcast
	go func() {
		for {
			cabCallMsg := <-peerCabCallsToNetworkC
			fmt.Println("cab calls to network")
			go broadcastCabCalls(cabCallMsg, cabCallsTxC)
		}
	}()

	for {
		select {
		case peerUpdate := <-peerUpdateRxC:
			fmt.Printf("Peer update:\n")
			fmt.Printf("  Peers:    %q\n", peerUpdate.Peers)
			fmt.Printf("  New:      %q\n", peerUpdate.New)
			fmt.Printf("  Lost:     %q\n", peerUpdate.Lost)

			alivePeersToSyncC <- peerUpdate.Peers

		case statusUpdate := <-statusRxC:
			if statusUpdate.SenderID != selfId {
				peerStatusUpdateToSyncC <- statusUpdate
			}
		}
	}
}