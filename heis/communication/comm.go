package communication

import (
	"heis/config"
	"heis/network/bcast"
	"heis/network/peers"
	"heis/worldview"
	"time"
)

//DATATYPER SOM IKKE EIES AV COMMUNICATION - MÅ HENTES ANDRE STEDER SENERE

type NetMsg struct {
	FromID          string
	Local           worldview.ElevState
	HallRequests    [config.N_FLOORS][2]worldview.OrderState
	BackupPeerState map[string]worldview.ElevState
}

//DATATYPER SOM EIES AV COMMUNICATION (network/peers)

// SKIFT NAVN, SAMME NAVN SOM PeerUpdate i network/peers
type PeerUpdate struct {
	ID    string
	Local worldview.ElevState
	Hall  [config.N_FLOORS][2]worldview.OrderState
}

// data som trengs for å tolke hvem som er tilgjengelig av andre peers enn den selv

//Hele denne structen er unødvendig
// (AbleToServiceRequests ligger i tilstanden som broadcastes)
// (PeerUpdates i network/peers inneholder tapte noder og nye noder, bruk denne!)
//ID er en del av melding som mottas/sendes

type PeerStatus struct {
	ID                    string
	AbleToServiceRequests bool
	SeenAt                time.Time
}

// definere ikke vei kanaler går, siden vi skal kunne bruke funksjoenr for å sende til og i communication bruke smame kanal for receive
type Communication struct {
	myID        string
	port        int
	bcastPeriod time.Duration

	//communcation har eierskap på. Lar retning være åpen for å la andre moduler nå verdier som sendes på.

	peerUpdateCh chan PeerUpdate //output opdateringer til peers på nett
	// (ENDRE NAVN!Samme navn som kanalen som oppdaterer hvilke peers er på nettet i network/peers )

	peerStatusCh     chan PeerStatus //output status til faulhandler
	transmitToNetCh  chan NetMsg     //write only
	receiveFromNetCh chan NetMsg     //read only

	localWorldviewCh <-chan worldview.ElevState                      //read local state
	hallStateCh      <-chan [config.N_FLOORS][2]worldview.OrderState //read only

	// maps broadcast peer ID to index in peerStateChs
	peerIDIndex map[string]int
	// one channel per other elevator, forwarded to worldview
	peerStateChs    []chan worldview.ElevState
	peerDiscoveryCh chan peers.PeerUpdate
}

func NewCommunicationModule(
	id string,
	port int,
	localStateCh <-chan worldview.ElevState,
	hallStateCh <-chan [config.N_FLOORS][2]worldview.OrderState,
	peerStateChs []chan worldview.ElevState,
) *Communication {

	peerDiscoveryCh := make(chan peers.PeerUpdate, 16)
	transmitEnable := make(chan bool)

	c := &Communication{
		myID:             id,
		port:             port,
		localWorldviewCh: localStateCh,
		hallStateCh:      hallStateCh,
		bcastPeriod:      1 * time.Second,
		peerUpdateCh:     make(chan PeerUpdate, 16),
		peerStatusCh:     make(chan PeerStatus, 16),
		transmitToNetCh:  make(chan NetMsg, 16),
		receiveFromNetCh: make(chan NetMsg, 16),
		peerIDIndex:      make(map[string]int),
		peerStateChs:     peerStateChs,
		peerDiscoveryCh:  peerDiscoveryCh,
	}

	go bcast.Transmitter(port, c.transmitToNetCh)
	go bcast.Receiver(port, c.receiveFromNetCh)
	go peers.Transmitter(config.PeersPort, id, transmitEnable)
	go peers.Receiver(config.PeersPort, peerDiscoveryCh)
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
	for {
		select {
		case localStateUpd := <-c.localWorldviewCh:
			outMsg.Local = localStateUpd

		case hallUpd := <-c.hallStateCh:
			outMsg.HallRequests = hallUpd

		case msg := <-c.receiveFromNetCh:
			//Ikke gå videre ved ygilige IDer
			if msg.FromID == "" || msg.FromID == c.myID {
				continue
			}

			//status sendes hver gang melding kommer inn, som heartbeat
			c.peerStatusCh <- PeerStatus{
				ID:                    msg.FromID,
				AbleToServiceRequests: msg.Local.AbleToServiceRequests,
				SeenAt:                time.Now(),
			}
			//Ved ny info om peer fra nettet, send PeerUpdate på peerUpdateCh
			if !isSameAsPrevious(lastPeerMsg, msg) {
				c.peerUpdateCh <- PeerUpdate{
					ID:    msg.FromID,
					Local: msg.Local,
					Hall:  msg.HallRequests,
				}
				// forward state to worldview on the correct peer channel
				if idx := c.getPeerIndex(msg.FromID); idx >= 0 {
					select {
					case c.peerStateChs[idx] <- msg.Local:
					default:
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
	return prev.Local == msg.Local && prev.HallRequests == msg.HallRequests
}

func (c *Communication) GetPeerUpdateChannel() <-chan PeerUpdate {
	return c.peerUpdateCh
}

func (c *Communication) GetPeerStatusChannel() <-chan PeerStatus {
	return c.peerStatusCh
}
