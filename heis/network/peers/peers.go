package peers

import (
	"fmt"
	"heis/network/conn"
	"net"
	"sort"
	"time"
)

type PeerUpdate struct {
	Peers []string //aktive noder akkurat nå
	New   string   //en node som akkurat dukket opp
	Lost  []string //noder som har dødd
}

const interval = 15 * time.Millisecond //hyppighet periodisk boradcast
const timeout = 500 * time.Millisecond //tidskrav boradcast

//oppg: peers.Transmitter sende ut ID periodisk

func Transmitter(port int, id string, transmitEnable <-chan bool) { //kanal med bool verdier leses i denne funksjonen

	conn := conn.DialBroadcastUDP(port)
	addr, _ := net.ResolveUDPAddr("udp4", fmt.Sprintf("255.255.255.255:%d", port))

	enable := true
	for {
		select {
		case enable = <-transmitEnable:
		case <-time.After(interval):
		}
		if enable {
			conn.WriteTo([]byte(id), addr) //skriver ID til adresse
		}
	}
}

//peers.receiver  lytter på port og lager oversikt over PeerUpdate

func Receiver(port int, peerUpdateCh chan<- PeerUpdate) {

	var buf [1024]byte
	var p PeerUpdate
	lastSeen := make(map[string]time.Time)

	conn := conn.DialBroadcastUDP(port)

	for {
		updated := false

		conn.SetReadDeadline(time.Now().Add(interval))
		n, _, _ := conn.ReadFrom(buf[0:])

		id := string(buf[:n])

		// Adding new connection
		p.New = ""
		if id != "" {
			if _, idExists := lastSeen[id]; !idExists {
				p.New = id
				updated = true
			}

			lastSeen[id] = time.Now()
		}

		// Removing dead connection
		p.Lost = make([]string, 0)
		for k, v := range lastSeen {
			if time.Now().Sub(v) > timeout {
				updated = true
				p.Lost = append(p.Lost, k)
				delete(lastSeen, k)
			}
		}

		// Sending update
		if updated {
			p.Peers = make([]string, 0, len(lastSeen))

			for k, _ := range lastSeen {
				p.Peers = append(p.Peers, k)
			}

			sort.Strings(p.Peers)
			sort.Strings(p.Lost)
			peerUpdateCh <- p
		}
	}
}
