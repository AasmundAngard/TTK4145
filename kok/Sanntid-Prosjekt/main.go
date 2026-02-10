package main

import (
	"flag"
	"fmt"
	"root/assigner"
	"root/config"
	"root/distributor"
	"root/elevator"
	"root/elevio"
	"root/lights"
	"root/network/bcast"
	"root/network/peers"
	"strconv"
)

var Port int
var id int

func main() {

	port := flag.Int("port", 15301, "<-- Default verdi, men kan overskrives som en command line argument ved bruk av -port=xxxxx")
	elevatorId := flag.Int("id", 0, "<-- Default verdi, men kan overskrives som en command line argument ved bruk av -id=xxxxx")
	flag.Parse()

	Port = *port
	id = *elevatorId

	fmt.Println()
	elevio.Init("localhost:"+strconv.Itoa(Port), config.NumFloors)

	fmt.Println("Elevator initialized with ID", id, "on port", Port)
	fmt.Println("System has", config.NumFloors, "floors and", config.NumElevators, "elevators.")

	newOrderC 				:= make(chan elevator.Orders, config.Buffer)
	deliveredOrderC 		:= make(chan elevio.ButtonEvent, config.Buffer)
	newLocalStateC  		:= make(chan elevator.State, config.Buffer)
	confirmedCommonstateC 	:= make(chan distributor.CommonState, config.Buffer)
	networkTx 				:= make(chan distributor.CommonState, config.Buffer)
	networkRx 				:= make(chan distributor.CommonState, config.Buffer)
	peersRx 				:= make(chan peers.PeerUpdate, config.Buffer)
	peersTx 				:= make(chan bool, config.Buffer)

	go peers.Receiver(config.PeersPortNumber, peersRx)
	go peers.Transmitter(config.PeersPortNumber, id, peersTx)

	go bcast.Receiver(config.BcastPortNumber, networkRx)
	go bcast.Transmitter(config.BcastPortNumber, networkTx)

	go distributor.Distributor(
		deliveredOrderC,
		newLocalStateC,
		networkTx,
		networkRx,
		confirmedCommonstateC,
		peersRx,
		id)

	go elevator.Elevator(
		newOrderC,
		deliveredOrderC,
		newLocalStateC)

	for {
		select {
		case cs := <-confirmedCommonstateC:
			localOrder := assigner.CalculateOptimalOrders(cs, id)
			newOrderC <- localOrder
			lights.SetLights(cs, id)

		default:
			continue
		}
	}
}
