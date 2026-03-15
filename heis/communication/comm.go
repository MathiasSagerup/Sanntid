package communication

import (
	"fmt"
	"heis/config"
	"heis/network/bcast"
	"heis/network/peers"
	"heis/worldview"
	"time"
)

//DATATYPER SOM IKKE EIES AV COMMUNICATION - MÅ HENTES ANDRE STEDER SENERE

type NetMsg struct {
	FromID         string
	LocalElevState worldview.ElevState
	//HallRequests    [config.N_FLOORS][2]worldview.OrderState
	BackupPeerState map[string]worldview.ElevState
}

//DATATYPER SOM EIES AV COMMUNICATION (network/peers)

// SKIFT NAVN, SAMME NAVN SOM PeerUpdate i network/peers
type PeerUpdate struct {
	ID    string
	Local worldview.ElevState
	Hall  [config.N_FLOORS][2]worldview.OrderState
}

type Communication struct {
	myID        string
	port        int 		//Distinasjonsport for broadcasting?
	bcastPeriod time.Duration

	peerUpdateCh chan PeerUpdate //output opdateringer til peers på nett
	// (ENDRE NAVN!Samme navn som kanalen som oppdaterer hvilke peers er på nettet i network/peers )

	transmitToNetCh  chan NetMsg     //write only
	receiveFromNetCh chan NetMsg     //read only

	localWorldviewCh <-chan worldview.ElevState //read local state

	// maps broadcast peer ID to index in peerStateChs
	peerIDIndex map[string]int
	// one channel per other elevator, forwarded to worldview
	peerStateChs    []chan worldview.ElevState
	peerDiscoveryCh chan peers.PeerUpdate
}

func NewCommunicationModule(
	id string,
	port int,
	worldviewToCommCh <-chan worldview.ElevState,
	peerStateChs []chan worldview.ElevState,
) *Communication {

	peerDiscoveryCh := make(chan peers.PeerUpdate, 16)
	transmitEnable := make(chan bool)

	c := &Communication{
		myID:             id,
		port:             port,
		localWorldviewCh: worldviewToCommCh,
		bcastPeriod:      15 * time.Millisecond,
		peerUpdateCh:     make(chan PeerUpdate, 16),
		transmitToNetCh:  make(chan NetMsg, 16),
		receiveFromNetCh: make(chan NetMsg, 16),
		peerIDIndex:      make(map[string]int),
		peerStateChs:     peerStateChs,
		peerDiscoveryCh:  peerDiscoveryCh,
	}

	go bcast.Transmitter(port, c.transmitToNetCh)
	go bcast.Receiver(port, c.receiveFromNetCh)
	go c.run()

	return c
}

//loop tilhører instand

func (c *Communication) run() {

	bcastTicker := time.NewTicker(c.bcastPeriod) //bcastTicker inneholder struct med en kanal C
	defer bcastTicker.Stop()

	//oppretter outMsg og LastPeerMsg

	outMsg := NetMsg{FromID: c.myID}       //melding ut er fra ID
	lastPeerMsg := make(map[string]NetMsg) //map av nøkkelpar ID til NetMsg

	//vi leser fra og setter data på kanaler.
	i := 0
	for {
		select {

		case updatedWorldview := <-c.localWorldviewCh:
			outMsg.LocalElevState = updatedWorldview
			//fmt.Println("updated worldview received")

		case msg := <-c.receiveFromNetCh:
			//Ikke gå videre ved ygilige IDer
			if msg.FromID == "" || msg.FromID == c.myID {
				continue
			}

			//Ved ny info om peer fra nettet, send PeerUpdate på peerUpdateCh
			if !isSameAsPrevious(lastPeerMsg, msg) {
				c.peerUpdateCh <- PeerUpdate{
					ID:    msg.FromID,
					Local: msg.LocalElevState,
				}
				// forward state to worldview on the correct peer channel
				if idx := c.getPeerIndex(msg.FromID); idx >= 0 {
					fmt.Printf("[comm] forwarding state from %s (idx=%d) HallCalls=%v\n", msg.FromID, idx, msg.LocalElevState.HallCalls)
					select {
					case c.peerStateChs[idx] <- msg.LocalElevState:
					default:
						fmt.Printf("[comm] WARNING: dropped state from %s (channel full)\n", msg.FromID)
					}
				}
				lastPeerMsg[msg.FromID] = msg
			}

		case peerUpdate := <-c.peerDiscoveryCh:
			if peerUpdate.New != "" {
				// index is kept permanently so the same elevator
				// gets the same index if it reconnects
				c.getPeerIndex(peerUpdate.New)
			}

		case <-bcastTicker.C: //utløses hver bcastPeriode
			if i == 0 {
				fmt.Println("broadcasting msg")
				i++
			}
			c.transmitToNetCh <- outMsg
		}
	}

}

// getPeerIndex returns the channel index for a peer ID, assigning a new slot if unseen.
// Returns -1 if no slots are available.
func (c *Communication) getPeerIndex(id string) int {
	if idx, ok := c.peerIDIndex[id]; ok {
		return idx
	}
	nextIdx := len(c.peerIDIndex)
	if nextIdx >= len(c.peerStateChs) {
		return -1
	}
	c.peerIDIndex[id] = nextIdx
	return nextIdx
}

func isSameAsPrevious(last map[string]NetMsg, msg NetMsg) bool {
	prev, ok := last[msg.FromID]
	if !ok {
		return false
	}
	return prev.LocalElevState == msg.LocalElevState
}

func (c *Communication) GetPeerUpdateChannel() <-chan PeerUpdate {
	return c.peerUpdateCh
}
