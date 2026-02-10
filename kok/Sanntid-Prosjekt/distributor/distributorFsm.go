package distributor

import (
	"fmt"
	"root/config"
	"root/elevator"
	"root/elevio"
	"root/network/peers"
	"time"
)

type StashType int

const (
	None	StashType = iota
	Remove
	Add
	State
)

func Distributor(
	deliveredOrderC 		<-chan elevio.ButtonEvent,
	newLocalStateC			<-chan elevator.State,
	networkTx				chan<- CommonState,
	networkRx 				<-chan CommonState,
	confirmedCommonstateC	chan<- CommonState,
	peersC 					<-chan peers.PeerUpdate,
	id 						int,
	){

	addOrderC := make(chan elevio.ButtonEvent, config.Buffer)

	go elevio.PollButtons(addOrderC)

	var removeStash elevio.ButtonEvent
	var addStash elevio.ButtonEvent
	var stateStash elevator.State
	var stashType StashType
	var peers peers.PeerUpdate
	var cs CommonState

	disconnectTimer := time.NewTimer(config.DisconnectTime)
	heartbeatTimer := time.NewTicker(config.HeartbeatTime)

	idle := true
	notOnNetwork := false

	for {
		select {
		case <-disconnectTimer.C:
			cs.makeOthersUnavailable(id)
			fmt.Println("Lost connection to network")
			notOnNetwork = true

		case P := <-peersC:
			peers = P
			cs.makeOthersUnavailable(id)
			idle = false

		case <-heartbeatTimer.C:
			networkTx <- cs

		default:
		}

		switch {
		case idle:
			select {
			case newOrder := <-addOrderC:
				stashType = Add
				addStash = newOrder
				cs.prepNewCs(id)
				cs.addOrder(newOrder, id)
				cs.Ackmap[id] = Acked
				idle = false

			case orderToRemove := <-deliveredOrderC:
				stashType = Remove
				removeStash = orderToRemove
				cs.prepNewCs(id)
				cs.removeOrder(orderToRemove, id)
				cs.Ackmap[id] = Acked
				idle = false

			case newLocalState := <-newLocalStateC:
				stashType = State
				stateStash = newLocalState
				cs.prepNewCs(id)
				cs.updateLocalState(newLocalState, id)
				cs.Ackmap[id] = Acked
				idle = false

			case arrivedCs := <-networkRx:
				disconnectTimer = time.NewTimer(config.DisconnectTime)
				if (arrivedCs.Origin > cs.Origin && arrivedCs.SeqNum == cs.SeqNum) || arrivedCs.SeqNum > cs.SeqNum {
					cs = arrivedCs
					cs.makeLostPeersUnavailable(peers)
					cs.Ackmap[id] = Acked
					idle = false
				}

			default:
			}

		case notOnNetwork:
			select {
			case <-networkRx:
				if cs.States[id].CabRequests == [config.NumFloors]bool{} {
					fmt.Println("Regained connection to network")
					notOnNetwork = false
				} else {
					cs.Ackmap[id] = NotAvailable
				}

			case newOrder := <-addOrderC:
				if cs.States[id].State.Motorstop {
					break
				}
				cs.Ackmap[id] = Acked
				cs.addCabCall(newOrder, id)
				confirmedCommonstateC <- cs

			case orderToRemove := <-deliveredOrderC:
				cs.Ackmap[id] = Acked
				cs.removeOrder(orderToRemove, id)
				confirmedCommonstateC <- cs

			case newLocalState := <-newLocalStateC:
				if !(newLocalState.Obstructed || newLocalState.Motorstop) {
					cs.Ackmap[id] = Acked
					cs.updateLocalState(newLocalState, id)
					confirmedCommonstateC <- cs
				}

			default:
			}

		case !idle:
			select {
			case arrivedCs := <-networkRx:
				if arrivedCs.SeqNum < cs.SeqNum {
					break
				}
				disconnectTimer = time.NewTimer(config.DisconnectTime)

				switch {
				case (arrivedCs.Origin > cs.Origin && arrivedCs.SeqNum == cs.SeqNum) || arrivedCs.SeqNum > cs.SeqNum:
					cs = arrivedCs
					cs.Ackmap[id] = Acked
					cs.makeLostPeersUnavailable(peers)

				case arrivedCs.fullyAcked(id):
					cs = arrivedCs
					confirmedCommonstateC <- cs
					switch {
					case cs.Origin != id && stashType != None:
						cs.prepNewCs(id)

						switch stashType {
						case Add:
							cs.addOrder(addStash, id)
							cs.Ackmap[id] = Acked

						case Remove:
							cs.removeOrder(removeStash, id)
							cs.Ackmap[id] = Acked

						case State:
							cs.updateLocalState(stateStash, id)
							cs.Ackmap[id] = Acked
						}

					case cs.Origin == id && stashType != None:
						stashType = None
						idle = true

					default:
						idle = true
					}

				case cs.equals(arrivedCs):
					cs = arrivedCs
					cs.Ackmap[id] = Acked
					cs.makeLostPeersUnavailable(peers)

				default:
				}
			default:
			}
		}
	}
}
