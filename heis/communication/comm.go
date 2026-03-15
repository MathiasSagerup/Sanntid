package communication

import (
	"fmt"
//	"heis/config"
	"heis/network/bcast"
//	"heis/network/peers"
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
//	Hall  [config.N_FLOORS][2]worldview.OrderState
}

type Communication struct {
	myID        string
	bcastPeriod time.Duration
	transmitToNetCh  chan NetMsg    
	receiveFromNetCh chan NetMsg 
	localWorldviewCh <-chan worldview.ElevState //read local state
	peerIDIndex map[string]int 		// maps broadcast peer ID to index in peerStateChs
	peerStateChs    []chan worldview.ElevState // one channel per other elevator, forwarded to worldview
	recoveredLocalStateCh chan worldview.ElevState
}

func NewCommunicationModule(
	id string,
	broadcastPort int,
	worldviewToCommCh <-chan worldview.ElevState,
	peerStateChs []chan worldview.ElevState,
	recoveredLocalStateCh chan worldview.ElevState,
) *Communication {
	//transmitEnable := make(chan bool)

	c := &Communication{
		myID:             		id,
		localWorldviewCh: 		worldviewToCommCh,
		bcastPeriod:      		15 * time.Millisecond,

		transmitToNetCh:  		make(chan NetMsg, 1),
		receiveFromNetCh: 		make(chan NetMsg, 1),
		peerIDIndex:      		make(map[string]int),
		peerStateChs:     		peerStateChs,
		recoveredLocalStateCh: 	recoveredLocalStateCh,
	}

	go bcast.Transmitter(broadcastPort, c.transmitToNetCh)
	go bcast.Receiver(broadcastPort, c.receiveFromNetCh)
	go c.run()

	return c
}

func (c *Communication) run() {

	bcastTicker := time.NewTicker(c.bcastPeriod)
	defer bcastTicker.Stop()

	outMsg := NetMsg{
		FromID: c.myID, 
		LocalElevState: worldview.ElevState{}, 
		BackupPeerState: make(map[string]worldview.ElevState),
	}

	lastPeerMsg := make(map[string]NetMsg) //map av nøkkelpar ID til NetMsg

	for {
		select {

		case updatedWorldview := <-c.localWorldviewCh:
			outMsg.LocalElevState = updatedWorldview
			//fmt.Println("updated worldview received")

		case msg := <-c.receiveFromNetCh:

			//Ikke gå videre med manglede ID eller meldinger fra oss selv
			if msg.FromID == "" || msg.FromID == c.myID {
				continue
			}

			//Vi ønsker ikke å behandle meldinger som er identiske med den siste mottatte meldingen fra samme peer
			if !isSameAsPrevious(lastPeerMsg, msg) {
				// Saving the latest message from this peer for future comparison and backup state recovery
				lastPeerMsg[msg.FromID] = msg
				outMsg.BackupPeerState[msg.FromID] = msg.LocalElevState

				// Forward state to worldview on the correct peer channel
				idx := c.getCurrentOrAssignNewPeerIndex(msg.FromID)
				fmt.Printf("[comm] forwarding state from %s (idx=%d) HallCalls=%v\n", msg.FromID, idx, msg.LocalElevState.HallCalls)
				select {
				case c.peerStateChs[idx] <- msg.LocalElevState:
				default:
					<- c.peerStateChs[idx] 
					c.peerStateChs[idx] <- msg.LocalElevState
				}

				// Updating our recovery state with the latest from this peer
				//TODO: Kanskje vi bare trenger å lese 1 recovered state og godta den første som kommer?
				recoveredState, localStateWasRecovered := msg.BackupPeerState[c.myID]
				if localStateWasRecovered {
					fmt.Printf("[comm] Received recovery state from peer %s: %v\n", msg.FromID, recoveredState)
					select {
					case c.recoveredLocalStateCh <- recoveredState:
					default:
						<- c.recoveredLocalStateCh
						c.recoveredLocalStateCh <- recoveredState
					}	
				}
			}

		case <-bcastTicker.C:
			select {
			case c.transmitToNetCh <- outMsg:
			default:
				fmt.Printf("[comm] WARNING: dropped message to network (channel full)\n")
			}
		}
	}
}

// getPeerIndex returns the channel index for a peer ID, assigning a new slot if unseen.
func (c *Communication) getCurrentOrAssignNewPeerIndex(id string) int {
	if idx, ok := c.peerIDIndex[id]; ok {
		return idx
	}

	nextIdx := len(c.peerIDIndex)
	if nextIdx >= len(c.peerStateChs) {
		errMsg := fmt.Sprintf("[comm] ERROR: Too many peers! Discovered peer %s but only %d slots available. System misconfigured.", 
        id, len(c.peerStateChs))
    	fmt.Printf("%s\n", errMsg)
    	panic(errMsg)
	}

	c.peerIDIndex[id] = nextIdx
	fmt.Printf("[comm] New peer detected. Assigned peer %s to index %d\n", id, nextIdx)
	return nextIdx
}

func isSameAsPrevious(last map[string]NetMsg, msg NetMsg) bool {
	prev, ok := last[msg.FromID]
	if !ok {
		//If no previous message from this peer, it's not the same
		return false
	}
	return prev.LocalElevState == msg.LocalElevState
}
