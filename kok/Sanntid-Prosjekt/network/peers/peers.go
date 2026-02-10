package peers

import (
	"fmt"
	"net"
	"root/config"
	"root/network/conn"
	"sort"
	"strconv"
	"time"
)

type PeerUpdate struct {
	Peers []int
	New   int
	Lost  []int
}

func Transmitter(port int, id int, transmitEnable <-chan bool) {

	conn := conn.DialBroadcastUDP(port)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))

	enable := true
	for {
		select {
		case enable = <-transmitEnable:
		case <-time.After(config.HeartbeatTime):
		}
		if enable {
			idStr := strconv.Itoa(id)
			conn.WriteTo([]byte(idStr), addr)
		}
	}
}

func Receiver(port int, peerUpdateCh chan<- PeerUpdate) {

	var buf [1024]byte
	var p PeerUpdate
	lastSeen := make(map[int]time.Time)
	p.Lost = make([]int, 0)

	conn := conn.DialBroadcastUDP(port)

	for {
		updated := false

		conn.SetReadDeadline(time.Now().Add(config.HeartbeatTime))
		n, _, _ := conn.ReadFrom(buf[0:])

		idStr := string(buf[:n])
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}

		p.New = 1000
		if _, idExists := lastSeen[id]; !idExists {
			p.New = id
			updated = true

			for i, lostPeer := range p.Lost {
				if lostPeer == id {
					p.Lost = append(p.Lost[:i], p.Lost[i+1:]...)
					break
				}
			}
		}

		lastSeen[id] = time.Now()

		for k, v := range lastSeen {
			if time.Now().Sub(v) > config.DisconnectTime {
				updated = true
				p.Lost = append(p.Lost, k)
				delete(lastSeen, k)
			}
		}

		if updated {
			p.Peers = make([]int, 0, len(lastSeen))

			for k := range lastSeen {
				p.Peers = append(p.Peers, k)
			}

			sort.Ints(p.Peers)
			sort.Ints(p.Lost)
			peerUpdateCh <- p
		}
	}
}
