package worldview

import (
	"heis/types"
)

// ---- Internal request/update types ----

type worldviewRequest struct {
	responseChan chan types.Worldview
}

type elevatorStateUpdate struct {
	id    string
	state types.ElevatorStateMsg
}

type hallRequestUpdate struct {
	hallRequests [][2]bool
}

type peerUpdate struct {
	update types.PeerUpdate
}

type hallRequestSet struct {
	floor  int
	btn    int
	active bool
}

// ---- Struct ----

type Worldview struct {
	// Internal state
	elevatorStates map[string]types.ElevatorStateMsg
	hallRequests   [][2]bool
	peers          []string

	// Outbound notification channel - worldview pushes here when peers are lost
	PeerLostChan chan string

	// Outbound notification channel - worldview pushes here when a new peer joins
	PeerJoinedChan chan string

	// Internal request/update channels
	worldviewRequestChan    chan worldviewRequest
	elevatorStateUpdateChan chan elevatorStateUpdate
	hallRequestUpdateChan   chan hallRequestUpdate
	peerUpdateChan          chan peerUpdate
	hallRequestSetChan      chan hallRequestSet
}

// ---- Constructor ----

func NewWorldview() *Worldview {
	w := &Worldview{
		elevatorStates:          make(map[string]types.ElevatorStateMsg),
		hallRequests:            make([][2]bool, types.N_FLOORS),
		peers:                   make([]string, 0),
		PeerLostChan:            make(chan string, 10),
		PeerJoinedChan:          make(chan string, 10),
		worldviewRequestChan:    make(chan worldviewRequest),
		elevatorStateUpdateChan: make(chan elevatorStateUpdate),
		hallRequestUpdateChan:   make(chan hallRequestUpdate),
		peerUpdateChan:          make(chan peerUpdate),
		hallRequestSetChan:      make(chan hallRequestSet),
	}
	go w.run()
	return w
}

// ---- Internal run loop ----

func (w *Worldview) run() {
	for {
		select {

		// Someone requesting the full worldview
		case req := <-w.worldviewRequestChan:
			req.responseChan <- w.copyWorldview()

		// New elevator state received from network
		case update := <-w.elevatorStateUpdateChan:
			w.elevatorStates[update.id] = update.state

		// New hall requests received from network
		case update := <-w.hallRequestUpdateChan:
			w.hallRequests = update.hallRequests

		// Single hall request button set (e.g. from local button press)
		case update := <-w.hallRequestSetChan:
			w.hallRequests[update.floor][update.btn] = update.active

		// Peer update from communication module
		case update := <-w.peerUpdateChan:
			w.peers = update.update.Peers

			// Notify about new peer
			if update.update.New != "" {
				select {
				case w.PeerJoinedChan <- update.update.New:
				default:
				}
			}

			// Notify about lost peers and remove their state
			for _, lostID := range update.update.Lost {
				delete(w.elevatorStates, lostID)
				select {
				case w.PeerLostChan <- lostID:
				default:
				}
			}
		}
	}
}

// ---- Helper: returns a deep copy of worldview state ----

func (w *Worldview) copyWorldview() types.Worldview {
	// Deep copy elevator states
	statesCopy := make(map[string]types.ElevatorStateMsg, len(w.elevatorStates))
	for id, state := range w.elevatorStates {
		statesCopy[id] = state
	}

	// Deep copy hall requests
	hallCopy := make([][2]bool, len(w.hallRequests))
	copy(hallCopy, w.hallRequests)

	// Deep copy peers
	peersCopy := make([]string, len(w.peers))
	copy(peersCopy, w.peers)

	return types.Worldview{
		ElevatorStates: statesCopy,
		HallRequests:   hallCopy,
		Peers:          peersCopy,
	}
}

// ---- Public methods ----

// Called by HallCallAssigner to get full current worldview
func (w *Worldview) GetWorldview() types.Worldview {
	respChan := make(chan types.Worldview)
	w.worldviewRequestChan <- worldviewRequest{
		responseChan: respChan,
	}
	return <-respChan
}

// Called by Communication when a new elevator state arrives from network
func (w *Worldview) UpdateElevatorState(id string, state types.ElevatorStateMsg) {
	w.elevatorStateUpdateChan <- elevatorStateUpdate{
		id:    id,
		state: state,
	}
}

// Called by Communication when hall requests are received from network
func (w *Worldview) UpdateHallRequests(hallRequests [][2]bool) {
	w.hallRequestUpdateChan <- hallRequestUpdate{
		hallRequests: hallRequests,
	}
}

// Called by LocalElevator when a hall button is pressed locally
func (w *Worldview) SetHallRequest(floor int, btn int, active bool) {
	w.hallRequestSetChan <- hallRequestSet{
		floor:  floor,
		btn:    btn,
		active: active,
	}
}

// Called by Communication when peer list changes
func (w *Worldview) UpdatePeers(update types.PeerUpdate) {
	w.peerUpdateChan <- peerUpdate{update: update}
}
