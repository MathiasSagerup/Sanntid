package worldview

import (
	"heis/config"
	"heis/driver"
	"heis/hallCallsAssigner"
	"heis/localElevator"
	"strconv"
	"fmt"
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

	messageFromLocalElevChannel  <-chan ElevState
	messageFromOtherElevChannels []<-chan ElevState //Index in array corresponds to ElevID
	hallCallButtonChan <-chan driver.ButtonEvent

	//channel to HCA
	hallCallAssignerChan chan hallCallsAssigner.HRAInput

	//LES !!

	//ID av heiser ved bruk av indeksering, potensielt problematisk mtp at heiser vil da endre IDen sin ila. programmets levetid når heiser
	//mister nett og blir med tilbake. Blir kaos i sjekkingen under if newWorldView == oldWorldView.
	// Tror det er bedre å bruke IDen som blir sent med broadcast meldingene.
}

func NewWorldViewModule(
	messageFromOtherElevChannels []<-chan ElevState,
	hallCallAssignerChan chan hallCallsAssigner.HRAInput,
	driverToWorldviewChan <-chan driver.ButtonEvent,
	initialLocalElevState ElevState,
	initialOtherElevStates []ElevState,
) *WorldViewDecider {

	//Assert correct length of array to a contsistent amount of elevators

	w := &WorldViewDecider{
		messageFromOtherElevChannels: messageFromOtherElevChannels,
		hallCallAssignerChan:         hallCallAssignerChan,
		hallCallButtonChan:           driverToWorldviewChan,
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
			if newElevState != w.thisElevState {
				w.thisElevState = newElevState
				elevatorStateHasChanged = true
				fmt.Println("received new localELevState")
			}

		//TODO: Sjekk om den logikken er korrekt. 
		case hallButtonPressed := <-w.hallCallButtonChan:
			if hallButtonPressed.Button != driver.BT_Cab {
				fmt.Println("received hallCallButtn press")
				if w.thisElevState.HallCalls[hallButtonPressed.Floor][hallButtonPressed.Button] == NoOrder {
            		w.compareSingleIncomingHallCall(Unconfirmed,
                	hallButtonPressed.Floor, hallButtonPressed.Button)
            		hallCallsHasChanged = true
        			}
					fmt.Println("processed hallbtnpress")
					fmt.Println(w.thisElevState.HallCalls)
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
			fmt.Println("sending updated info to HallCallAssigner")
			w.hallCallAssignerChan <- w.sendUpdatedInformationToHallCallAssigner()
		}
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
	allElevs := append([]ElevState{w.thisElevState}, w.otherElevStates...)

	for i, elev := range allElevs {
		cabReqs := make([]bool, config.N_FLOORS)
		for f := 0; f < config.N_FLOORS; f++ {
			cabReqs[f] = elev.CabRequests[f]
		}

		states[strconv.Itoa(i)] = hallCallsAssigner.HRAElevState{
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
