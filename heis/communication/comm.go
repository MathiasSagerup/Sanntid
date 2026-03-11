package communication

import (
	"heis/config"
	"heis/network/bcast"
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

//DATATYPER SOM EIES AV COMMUNICATION

type PeerUpdate struct {
	ID    string
	Local worldview.ElevState
	Hall  [config.N_FLOORS][2]worldview.OrderState
}

// data som trengs for å tolke hvem som er tilgjengelig av andre peers enn den selv
type PeerStatus struct {
	ID                    string
	AbleToServiceRequests bool
	SeenAt                time.Time
}

// definere ikke vei kanaler går, siden vi skal kunne bruke funksjoenr for å sende til og i communication bruke smame kanal for receive
type Communication struct {
	myID        string
	port        int
	bcastPeriod time.Duration //hyppighet for bcfast

	//communcation har eierskap på. Lar retning være åpen for å la andre moduler nå verdier som sendes på.

	peerUpdateCh     chan PeerUpdate //output opdateringer til peers på nett
	peerStatusCh     chan PeerStatus //output status til faulhandler
	transmitToNetCh  chan NetMsg     //write only
	receiveFromNetCh chan NetMsg     //read only

	//Communcation sine input channels (read only) og ouput (write only) som den ikke har eierksp på
	//her definerer man om communication skal lese eller skrive fra channelse

	localStateCh <-chan worldview.ElevState                      //read local state
	hallStateCh  <-chan [config.N_FLOORS][2]worldview.OrderState //read only
}

//for å bruke communcation må du si hvilken heis du er, og porten som communkikasjonsmodulen skal kommunisere med nette tpå

func NewCommunicationModule(
	//send inn parametre fra main, siden main kobler oss ti landre modulers eierskap
	id string,
	port int,
	localStateCh <-chan worldview.ElevState,
	hallStateCh <-chan [config.N_FLOORS][2]worldview.OrderState,

) *Communication {

	//opprette instans med kanaler uten retning. Vi definerer i loop om vi skriver/leser fra kanlaer, og deifnere i funksjoner hvordan andre kanaler kan hente ut data vi finner
	c := &Communication{
		myID: id,
		port: port,
		//Hent ut det vi trenger fra andre cahnnels:
		localStateCh: localStateCh,
		hallStateCh:  hallStateCh,
		bcastPeriod:  1 * time.Second,
		//opprett cahnnels for private eierskap
		peerUpdateCh:     make(chan PeerUpdate, 16),
		peerStatusCh:     make(chan PeerStatus, 16),
		transmitToNetCh:  make(chan NetMsg, 16), //communction eier nettverks channesl
		receiveFromNetCh: make(chan NetMsg, 16),
	}

	//Det er kun communication som må vite noe om network
	go bcast.Transmitter(port, c.transmitToNetCh) // bcast.Transmitter leser fra c.transmitToNetCh, og bcaster på port
	go bcast.Receiver(port, c.receiveFromNetCh)   //bcast.Receiver lytter på port og legger på c.receiveFromNetCh
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
		case localStateUpd := <-c.localStateCh:
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
				lastPeerMsg[msg.FromID] = msg
			}

		case <-bcastTicker.C: //utløses hver bcastPeriode
			c.transmitToNetCh <- outMsg
		}
	}

}

func isSameAsPrevious(last map[string]NetMsg, msg NetMsg) bool {
	prev, ok := last[msg.FromID]
	if !ok {
		return false
	}
	return prev.Local == msg.Local && prev.HallRequests == msg.HallRequests
}

//funksjoner tilhører Communication typen brukes for å returrnere kanaler som tilhører
//funksjoner lar andre moduler skrive til communcation og lese fra kanaler ut fra communication
//da gir vil kun tilgang til den kanalene som skal brukes, ikke alle kanaler

//peer eier PeerUpdate og PeerStatus'//VURDER Å SENDE PEKER TILBAKE ISTENEF RDETTE?

//GetPeerUpdateChannel

func (c *Communication) GetPeerUpdateChannel() <-chan PeerUpdate {
	return c.peerUpdateCh
}

func (c *Communication) GetPeerStatusChannel() <-chan PeerStatus {
	return c.peerStatusCh
}
