package communication

//This module is responsible for creating and passing on masseges to our network module.
//Recieved messages are filtered, and forwarded to worldview. It also detects which peers are connected and forwards this information
//The module is also able to recover its previous state in case of a system restart

import (
	"fmt"
	"heis/config"
	"heis/network/bcast"
	"heis/worldview"
	"time"
)



type NetMsg struct {
	FromID         				string
	LocalState					worldview.PeerState
	BackupPeerState				map[string]worldview.PeerState  		//maps string ID of peer to their latest sent state
}

type Communication struct {
	myID                  string
	bcastPeriod           time.Duration
	timoutCheckPeriod     time.Duration
	transmitToNetCh       chan NetMsg
	receiveFromNetCh      chan NetMsg
	localPeerStateCh      <-chan worldview.PeerState
	peerIDIndex           map[string]int                                   	// maps peer ID to its correponding index used in arrays
	PeerStateChs          [config.N_ELEVATORS - 1]chan worldview.PeerState 	// one channel per other elevator
	peersConnectedCh      chan [config.N_ELEVATORS - 1]bool
	recoveredCabCallsCh   chan [config.N_FLOORS]bool						//Used during initialization to regain previous state from peers if we are recovering from a failure
}

func NewCommunicationModule(
	id string,
	broadcastPort int,
	worldviewToCommCh <-chan worldview.PeerState,
	PeerStateChs [config.N_ELEVATORS - 1]chan worldview.PeerState,
	recoveredCabCallsCh chan [config.N_FLOORS]bool,
	peersConnectedCh chan [config.N_ELEVATORS - 1]bool,
) *Communication {

	c := &Communication{
		myID:                  id,
		localPeerStateCh:      worldviewToCommCh,
		bcastPeriod:           15 * time.Millisecond,
		timoutCheckPeriod:     100 * time.Millisecond,
		transmitToNetCh:       make(chan NetMsg, 1),
		receiveFromNetCh:      make(chan NetMsg, 1),
		peerIDIndex:           make(map[string]int),
		PeerStateChs:		   PeerStateChs,
		peersConnectedCh:      peersConnectedCh,
		recoveredCabCallsCh:   recoveredCabCallsCh,
	}

	go bcast.Transmitter(broadcastPort, c.transmitToNetCh)
	go bcast.Receiver(broadcastPort, c.receiveFromNetCh)
	go c.run()

	return c
}

func (c *Communication) run() {

	bcastTicker := time.NewTicker(c.bcastPeriod)
	defer bcastTicker.Stop()

	checkElevatorTimeout := time.NewTicker(c.timoutCheckPeriod)
	defer checkElevatorTimeout.Stop()

	timeLastMessageRecieved := make(map[string]time.Time)
	connectedElevators := [config.N_ELEVATORS - 1]bool{}

	outMsg := NetMsg{
		FromID:        		c.myID,
		LocalState:  		worldview.PeerState{},
		BackupPeerState: 	make(map[string]worldview.PeerState),
	}

	lastPeerMsg := make(map[string]NetMsg)

	for {
		select {

		case updatedLocalPeerState := <-c.localPeerStateCh:
			outMsg.LocalState = updatedLocalPeerState

		case msg := <-c.receiveFromNetCh:

			if msg.FromID == "" || msg.FromID == c.myID {
				continue
			}

			//Register that we have recieved a message from this peer
			id_index := c.getCurrentOrAssignNewPeerIndex(msg.FromID)
			timeLastMessageRecieved[msg.FromID] = time.Now()

			if connectedElevators[id_index] == false {
				connectedElevators[id_index] = true
				c.sendToPeersConnectedCh(connectedElevators)
			}

			if !isSameAsPrevious(lastPeerMsg, msg) {
				lastPeerMsg[msg.FromID] = msg
				outMsg.BackupPeerState[msg.FromID] = msg.LocalState
				c.sendToPeerStateChs(id_index, msg)

				// Updating our recovered cab calls with the latest from this peer
				recoveredState, localStateWasRecovered := msg.BackupPeerState[c.myID]
				if localStateWasRecovered {
					select {
					case c.recoveredCabCallsCh <- recoveredState.LocalElevState.CabRequests:
					default:
						<-c.recoveredCabCallsCh
						c.recoveredCabCallsCh <- recoveredState.LocalElevState.CabRequests
					}
				}
			}

		case <-bcastTicker.C:
			select {
			case c.transmitToNetCh <- outMsg:
			default:
				fmt.Printf("[comm] WARNING: dropped message to network (channel full)\n")
			}

		case <-checkElevatorTimeout.C:
			connectedElevatorsBeforeCheck := connectedElevators

			// Check which peers are currently timed out
			now := time.Now()
			for peerID, lastTimeMessageRecieved := range timeLastMessageRecieved {
				id_index := c.getCurrentOrAssignNewPeerIndex(peerID)
				if now.Sub(lastTimeMessageRecieved) > config.PEER_TIMEOUT_DURATION_MS * time.Millisecond {
					connectedElevators[id_index] = false
				} else {
					connectedElevators[id_index] = true
				}
			}

			if connectedElevators != connectedElevatorsBeforeCheck {
				c.sendToPeersConnectedCh(connectedElevators)
			}
		}
	}
}

func (c *Communication) sendToPeersConnectedCh(connectedElevators [config.N_ELEVATORS - 1]bool) {
	fmt.Printf("[comm] Changed elevator connectivity status: %v\n", connectedElevators)
	select {
	case c.peersConnectedCh <- connectedElevators:
	default:
		<-c.peersConnectedCh
		c.peersConnectedCh <- connectedElevators
	}
}

func (c *Communication) sendToPeerStateChs(id_index int, msg NetMsg) {
	select {
	case c.PeerStateChs[id_index] <- msg.LocalState:
	default:
		<-c.PeerStateChs[id_index]
		c.PeerStateChs[id_index] <- msg.LocalState
	}
}

func (c *Communication) getCurrentOrAssignNewPeerIndex(id string) int {
	if idx, ok := c.peerIDIndex[id]; ok {
		return idx
	}

	nextIdx := len(c.peerIDIndex)
	if nextIdx >= len(c.PeerStateChs) {
		errMsg := fmt.Sprintf("[comm] ERROR: Too many peers! Discovered peer %s but only %d slots available. System misconfigured.", id, len(c.PeerStateChs))
		fmt.Printf("%s\n", errMsg)
		panic(errMsg)
	}

	c.peerIDIndex[id] = nextIdx
	fmt.Printf("[comm] New peer detected. Assigned peer %s to index %d\n", id, nextIdx)
	return nextIdx
}

func isSameAsPrevious(last map[string]NetMsg, msg NetMsg) bool {
	prev, previousMessageFound := last[msg.FromID]
	if !previousMessageFound {
		return false
	}
	return prev.LocalState == msg.LocalState
}
