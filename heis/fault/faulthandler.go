package fault

import (
	"time"
)

type AvailableElevs struct {
	Available map[string]bool
}

type Config struct {
	PeerTimeout   time.Duration
	PublishPeriod time.Duration
	AllElevatorIDs []string
}

type PeerStatus struct {
	ID                string
	AbleToServiceHall bool
	SeenAt            time.Time
}

//endre disse lokalt i faulhandler 

func defaultConfig() Config {
	return Config{
		PeerTimeout:   3 * time.Second,
		PublishPeriod: 1 * time.Second,
		AllElevatorIDs: []string{"1", "2", "3"},
	}
}

//SETT OP STRUCT MED all informasjon som kommer fra hver modul, kanealer fora 

//type FaulHandler struct {
//	cfg Config
//}

//Definerer lokale variabler og initier kanalnavn og type data på kanaler 
type FaultHandler struct{
	myID 			string
	cfg				Config
	//eierskap av fh
	availableCh 	chan AvailableElevs //output 
	//cahnnels den ikke har eierskp på: 
	peerStatusCh 	<-chan PeerStatus //fh skal kun lese peerstatus 
	localStuckCh 	<-chan bool //input
}

func InitialzeFaultHandlerModule(
	id string,
	peerStatusCh <-chan PeerStatus,
	localStuckCh <-chan bool,
	) *FaultHandler {

	//lager instand av type Faulhandler 
	f := &FaultHandler{
		//hent ut fra parameterlisten 
		myID: 			id,
		peerStatusCh: 	peerStatusCh,
		localStuckCh: 	localStuckCh,
		//sett lokal informasjon
		cfg:			defaultConfig(), //default config values 
		availableCh:	make(chan AvailableElevs, 16), //sender ut tilgjegnelige heiser
	}
		
	
	go f.loop()

	return f
}

func (f *FaultHandler) loop() {
	//siste status for peer som faulhandler har. 
	lastStatusByPeer := make(map[string]PeerStatus)
	localIsStuck := false

	//ticker utløser sending av availableCh
	publishAvailableTicker := time.NewTicker(f.cfg.PublishPeriod)
	defer publishAvailableTicker.Stop()

	for {
		select {
		case status := <-f.peerStatusCh: //ny status oppdatering fra communcation
			//se kun på peers 
			if status.ID == "" || status.ID == f.myID {
				continue
			}
			lastStatusByPeer[status.ID] = status

		case stuck := <-f.localStuckCh: //ny status på stuck lokalt 
			localIsStuck = stuck

		case <-publishAvailableTicker.C: //gjør klart sending av availableCh
			now := time.Now()
			available := make(map[string]bool)

			//antar utilgjengelighet på hver heis som standard 
			for _, id := range f.cfg.AllElevatorIDs {
				available[id] = false
			}
		
			// Egen heis er alltid på nett, men ikke tilgjengelig hvis stuck
			available[f.myID] = !localIsStuck
		
			// Sett peers til true hvis de er på nett og kan ta hall calls
			for id, status := range lastStatusByPeer {
				if peerIsAlive(status.SeenAt, f.cfg.PeerTimeout, now) && status.AbleToServiceHall {
					available[id] = true
				}
			}

			//send availableCh
		
			f.availableCh <- AvailableElevs{
				Available: available,
			}
		}
	}
}

func peerIsAlive(lastSeen time.Time, timeout time.Duration, now time.Time) bool {
	return now.Sub(lastSeen) <= timeout
}


func (f *FaultHandler) GetAvailableElevsChannel() <-chan AvailableElevs {
	return f.availableCh
}

