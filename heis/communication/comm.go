package communication

import (
	"fmt"
	"heis/config"
	"heis/network/bcast"
	"heis/worldview"
	"time"
)

type NetMsg struct {
	FromID         				string
	localState					worldview.PeerState
	Backupworldview				map[string]worldview.PeerState  		//maps string ID of peer to their latest sent state
}

type Communication struct {
	myID                  string
	bcastPeriod           time.Duration
	timoutCheckPeriod     time.Duration
	transmitToNetCh       chan NetMsg
	receiveFromNetCh      chan NetMsg
	localWorldViewCh      <-chan worldview.PeerState
	peerIDIndex           map[string]int                                   	// maps peer ID to its correponding index in worldview.PeerStateChs and peersConnectedCh
	PeerStateChs          [config.N_ELEVATORS - 1]chan worldview.PeerState 	// one channel per other elevator, forwarded to worldview
	localWorldViewCh      <-chan worldview.PeerState
	peerIDIndex           map[string]int                                   	// maps peer ID to its correponding index in worldview.PeerStateChs and peersConnectedCh
	PeerStateChs          [config.N_ELEVATORS - 1]chan worldview.PeerState 	// one channel per other elevator, forwarded to worldview
	peersConnectedCh      chan [config.N_ELEVATORS - 1]bool
	recoveredCabCallsCh   chan [config.N_FLOORS]bool							//Used during initialization to regain previous state from peers if we are recovering from a failure
}

func NewCommunicationModule(
	id string,
	broadcastPort int,
	worldviewToCommCh <-chan worldview.PeerState,
	PeerStateChs [config.N_ELEVATORS - 1]chan worldview.PeerState,
	recoveredCabCallsCh chan [config.N_FLOORS]bool,
	peersConnectedCh chan [config.N_ELEVATORS - 1]bool,
) *Communication {
	//transmitEnable := make(chan bool)

	c := &Communication{
		myID:                  id,
		localWorldViewCh:      worldviewToCommCh,
		bcastPeriod:           15 * time.Millisecond, //TODO: Føler ikke denne er på riktig plass, flytte til config?
		timoutCheckPeriod:     100 * time.Millisecond, //TODO: Føler ikke denne er på riktig plass, flytte til config?
		transmitToNetCh:       make(chan NetMsg, 1),
		receiveFromNetCh:      make(chan NetMsg, 1),
		peerIDIndex:           make(map[string]int),
		PeerStateChs:		   PeerStateChs,
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
		localState:  		worldview.PeerState{},
		Backupworldview: 	make(map[string]worldview.PeerState),
	}

	lastPeerMsg := make(map[string]NetMsg) //map av nøkkelpar ID til NetMsg

	for {
		select {

		case updatedWorldview := <-c.localWorldViewCh:
			fmt.Println("Update recieved with hallcalls", updatedWorldview.HallCalls)
			outMsg.localState = updatedWorldview

		case msg := <-c.receiveFromNetCh:

			//Ikke gå videre med manglede ID eller meldinger fra oss selv
			if msg.FromID == "" || msg.FromID == c.myID {
				continue
			}


			//Registrer at vi har mottatt en melding fra denne peer
			id_index := c.getCurrentOrAssignNewPeerIndex(msg.FromID)
			timeLastMessageRecieved[msg.FromID] = time.Now()
			if connectedElevators[id_index] == false {
				connectedElevators[id_index] = true
				c.sendToPeersConnectedCh(connectedElevators)
			}
			
			//Vi ønsker ikke å behandle meldinger som er identiske med den siste mottatte meldingen fra samme peer
			if !isSameAsPrevious(lastPeerMsg, msg) {
				fmt.Println("[comm] Videresender hallcalls", msg.LocalState.HallCalls)
				lastPeerMsg[msg.FromID] = msg
				outMsg.Backupworldview[msg.FromID] = msg.localState //Will be returned to the sending peer for backup
				c.sendToPeerStateChs(id_index, msg)

				// Updating our recovery state with the latest from this peer
				recoveredState, localStateWasRecovered := msg.Backupworldview[c.myID]
				recoveredState, localStateWasRecovered := msg.Backupworldview[c.myID]
				if localStateWasRecovered {
					select {
					case c.recoveredCabCallsCh <- recoveredState.LocalElevState.CabRequests:
					case c.recoveredCabCallsCh <- recoveredState.LocalElevState.CabRequests:
					default:
						<-c.recoveredCabCallsCh
						c.recoveredCabCallsCh <- recoveredState.LocalElevState.CabRequests
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
			connectedElevatorsFromLastCheck := connectedElevators

			// Check which peers are currently timed out
			now := time.Now()
			for peerID, lastTimeMessageRecieved := range timeLastMessageRecieved {
				id_index := c.getCurrentOrAssignNewPeerIndex(peerID)
				if now.Sub(lastTimeMessageRecieved) > config.PEER_TIMEOUT_DURATION*time.Millisecond {
					connectedElevators[id_index] = false
				} else {
					connectedElevators[id_index] = true
				}
			}

			// Only send an update if a new elevator has timed out
			if connectedElevators != connectedElevatorsFromLastCheck {
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
	fmt.Printf("[comm] forwarding state from %s (idx=%d) HallCalls=%v\n", msg.FromID, id_index, msg.localState.HallCalls)
	select {
	case c.PeerStateChs[id_index] <- msg.localState:
	default:
		<-c.PeerStateChs[id_index]
		c.PeerStateChs[id_index] <- msg.localState
	}
}

func (c *Communication) getCurrentOrAssignNewPeerIndex(id string) int {
	if idx, ok := c.peerIDIndex[id]; ok {
		return idx
	}

	nextIdx := len(c.peerIDIndex)
	if nextIdx >= len(c.PeerStateChs) {
	if nextIdx >= len(c.PeerStateChs) {
		errMsg := fmt.Sprintf("[comm] ERROR: Too many peers! Discovered peer %s but only %d slots available. System misconfigured.",
			id, len(c.PeerStateChs))
			id, len(c.PeerStateChs))
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
	return prev.localState == msg.localState
}
