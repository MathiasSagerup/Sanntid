package worldview

import (
	"fmt"
	"heis/config"
	"heis/driver"
	"heis/hallCallsAssigner"
	"heis/localElevator"
	"strconv"
)

type elevatorID int

type ElevState struct {
	Floor                 int
	Dirn                  driver.MotorDirection
	Behaviour             localElevator.ElevatorBehaviour
	CabRequests           [config.N_FLOORS]bool
	Obstruction           bool
	AbleToServiceRequests bool
	HallCalls             [config.N_FLOORS][2]OrderState
}

type OrderState int

const (
	NoOrder OrderState = iota
	Unconfirmed
	Confirmed
	Completed
)

// World View Decider module ------------------------------------------------------------------------------------------

type WorldViewDecider struct {
	thisElevState   ElevState
	otherElevStates []ElevState //Index corresponds to ElevID

	messageFromLocalElevChannel  <-chan localElevator.ElevState
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan ElevState //Index in array corresponds to ElevID
	hallCallButtonChan           <-chan driver.ButtonEvent

	//channel to HCA
	hallCallAssignerChan chan hallCallsAssigner.HRAInput

	//channel to communication module
	toCommCh chan<- ElevState

	localID string

	//ID av heiser ved bruk av indeksering, potensielt problematisk mtp at heiser vil da endre IDen sin ila. programmets levetid når heiser
	//mister nett og blir med tilbake. Blir kaos i sjekkingen under if newWorldView == oldWorldView.
	// Tror det er bedre å bruke IDen som blir sent med broadcast meldingene.
}

func NewWorldViewModule(
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan ElevState,
	hallCallAssignerChan chan hallCallsAssigner.HRAInput,
	driverToWorldviewChan <-chan driver.ButtonEvent,
	toCommCh chan<- ElevState,
	localElevCh <-chan localElevator.ElevState,
	localID string,
	initialLocalElevState ElevState,
	initialOtherElevStates []ElevState,
) *WorldViewDecider {

	w := &WorldViewDecider{
		messageFromOtherElevChannels: messageFromOtherElevChannels,
		hallCallAssignerChan:         hallCallAssignerChan,
		hallCallButtonChan:           driverToWorldviewChan,
		toCommCh:                     toCommCh,
		messageFromLocalElevChannel:  localElevCh,
		localID:                      localID,
		thisElevState:                initialLocalElevState,
		otherElevStates:              initialOtherElevStates,
	}

	go w.loop()

	return w
}

func (w *WorldViewDecider) loop() {
	var hallCallsHasChanged bool
	var elevatorStateHasChanged bool
	for {
		hallCallsHasChanged = false
		elevatorStateHasChanged = false

		//Check for message from the local elevator
		select {
		case newElevState := <-w.messageFromLocalElevChannel:
			converted := localElevStateToWorldviewState(newElevState, w.thisElevState.HallCalls)
			if converted != w.thisElevState {
				w.thisElevState = converted
				elevatorStateHasChanged = true
				//fmt.Println("received new localELevState")
			}

		//TODO: Sjekk om den logikken er korrekt.
		case hallButtonPressed := <-w.hallCallButtonChan:
			if hallButtonPressed.Button != driver.BT_Cab {
				//fmt.Println("received hallCallButtn press")
				if w.thisElevState.HallCalls[hallButtonPressed.Floor][hallButtonPressed.Button] == NoOrder {
					w.compareSingleIncomingHallCall(Unconfirmed,
						hallButtonPressed.Floor, hallButtonPressed.Button)
					hallCallsHasChanged = true
				}
				//fmt.Println("processed hallbtnpress")
				//fmt.Println(w.thisElevState.HallCalls)
			}
			//TODO? check if the orderState has changed. if yes set hallCallsHasChanged variable to true

		default:
		}

		//Check for message from all other elevators
		for elevID := 0; elevID < len(w.messageFromOtherElevChannels); elevID++ {
			select {
			case newElevState := <-w.messageFromOtherElevChannels[elevID]:
				if newElevState != w.otherElevStates[elevID] {
					w.otherElevStates[elevID] = newElevState
					elevatorStateHasChanged = true
					w.compareIncomingHallCalls(newElevState.HallCalls)
				}
			default:
			}
		}

		if elevatorStateHasChanged || hallCallsHasChanged {
			//fmt.Println("sending updated info to HallCallAssigner")
			//fmt.Println(w.thisElevState.HallCalls)
			w.hallCallAssignerChan <- w.sendUpdatedInformationToHallCallAssigner()
			w.toCommCh <- w.thisElevState

		}
	}
}

func localElevStateToWorldviewState(s localElevator.ElevState, existingHallCalls [config.N_FLOORS][2]OrderState) ElevState {
	return ElevState{
		Floor:                 s.Floor,
		Dirn:                  s.Dirn,
		Behaviour:             s.Behaviour,
		CabRequests:           s.CabRequests,
		Obstruction:           s.Obstruction,
		AbleToServiceRequests: s.AbleToServiceRequests,
		HallCalls:             existingHallCalls,
	}
}

func (w *WorldViewDecider) recieveOtherElevMessage(incomingElevState ElevState, senderElevID elevatorID, hallCallsHasChanged bool) {
	if incomingElevState.HallCalls != w.thisElevState.HallCalls {
		w.otherElevStates[senderElevID] = incomingElevState
		w.compareIncomingHallCalls(incomingElevState.HallCalls)
		hallCallsHasChanged = true
	}
}

func (w *WorldViewDecider) compareIncomingHallCalls(incomingHallCalls [config.N_FLOORS][2]OrderState) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btnType := 0; btnType < 2; btnType++ {
			w.compareSingleIncomingHallCall(incomingHallCalls[floor][btnType], floor, driver.ButtonType(btnType))
		}
	}
}

func (w *WorldViewDecider) compareSingleIncomingHallCall(incomingHallCallActivation OrderState, floor int, hallBtn driver.ButtonType) {
	switch w.thisElevState.HallCalls[floor][hallBtn] {
	case NoOrder:
		switch incomingHallCallActivation {
		case Unconfirmed:
			w.thisElevState.HallCalls[floor][hallBtn] = Unconfirmed
		case Confirmed:
			w.thisElevState.HallCalls[floor][hallBtn] = Confirmed
		default:
		}

	case Unconfirmed:
		switch incomingHallCallActivation {
		case Unconfirmed:
			//If we recieve unconfirmed we must check if all elevators agree on unconfirmed and change to confirmed
			unconfirmedCounter := 0
			for elevID := 0; elevID < len(w.otherElevStates); elevID++ {
				if (w.otherElevStates[elevID].AbleToServiceRequests) && (w.otherElevStates[elevID].HallCalls[floor][hallBtn] == Unconfirmed) {
					unconfirmedCounter++
				}
			}
			fmt.Printf("[confirm check] floor=%d btn=%d unconfirmedCounter=%d aliveCount=%d\n",
				floor, hallBtn, unconfirmedCounter, w.getOtherElevsAliveCount())
			for i, s := range w.otherElevStates {
				fmt.Printf("  otherElev[%d]: AbleToServiceRequests=%v HallCall=%v\n",
					i, s.AbleToServiceRequests, s.HallCalls[floor][hallBtn])
			}
			if unconfirmedCounter == w.getOtherElevsAliveCount() {
				w.thisElevState.HallCalls[floor][hallBtn] = Confirmed
			}
		case Confirmed:
			w.thisElevState.HallCalls[floor][hallBtn] = Confirmed
		default:
		}

	case Confirmed:
		switch incomingHallCallActivation {
		case Completed:
			w.thisElevState.HallCalls[floor][hallBtn] = Completed
		default:
		}
	case Completed:
		switch incomingHallCallActivation {
		case Completed:
			//If we recieve completed we must check if all elevators agree on completed and if so, change to NoOrder
			completedCounter := 0
			for elevID := 0; elevID < len(w.otherElevStates); elevID++ {
				if (w.otherElevStates[elevID].AbleToServiceRequests) && (w.otherElevStates[elevID].HallCalls[floor][hallBtn] == Completed) {
					completedCounter++
				}
			}
			if completedCounter == w.getOtherElevsAliveCount() {
				w.thisElevState.HallCalls[floor][hallBtn] = NoOrder

			}
		case NoOrder:
			w.thisElevState.HallCalls[floor][hallBtn] = NoOrder
		}
	}
}

func (w *WorldViewDecider) getOtherElevsAliveCount() int {
	counter := 0
	for elevID := 0; elevID < len(w.otherElevStates); elevID++ {
		if w.otherElevStates[elevID].AbleToServiceRequests {
			counter++
		}
	}
	return counter
}

func (w *WorldViewDecider) sendUpdatedInformationToHallCallAssigner() hallCallsAssigner.HRAInput {
	var hallRequests [config.N_FLOORS][2]bool

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for dir := 0; dir < 2; dir++ {
			if w.thisElevState.HallCalls[floor][dir] == Confirmed {
				hallRequests[floor][dir] = true
			}
		}
	}

	states := make(map[string]hallCallsAssigner.HRAElevState)

	// local elevator uses its string ID so the HCA can look it up by ID
	localCabReqs := make([]bool, config.N_FLOORS)
	for f := 0; f < config.N_FLOORS; f++ {
		localCabReqs[f] = w.thisElevState.CabRequests[f]
	}
	states[w.localID] = hallCallsAssigner.HRAElevState{
		Behaviour:   w.thisElevState.Behaviour.String(),
		Floor:       w.thisElevState.Floor,
		Direction:   w.thisElevState.Dirn.String(),
		CabRequests: localCabReqs,
	}

	// other elevators use numeric keys (HCA doesn't need to look them up by name)
	for i, elev := range w.otherElevStates {
		cabReqs := make([]bool, config.N_FLOORS)
		for f := 0; f < config.N_FLOORS; f++ {
			cabReqs[f] = elev.CabRequests[f]
		}
		states["peer-"+strconv.Itoa(i)] = hallCallsAssigner.HRAElevState{
			Behaviour:   elev.Behaviour.String(),
			Floor:       elev.Floor,
			Direction:   elev.Dirn.String(),
			CabRequests: cabReqs,
		}
	}

	return hallCallsAssigner.HRAInput{
		HallRequests: hallRequests,
		States:       states,
	}
}
