package worldview

import (
	"fmt"
	"heis/config"
	"heis/driver"
	"heis/hallCallsAssigner"
	"heis/localElevator"
	"strconv"
	"time"
)

type ElevatorBehaviour int

const (
	idle     = 0
	moving   = 1
	doorOpen = 2
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

type HRAInput struct {
	HallRequests  	[config.N_FLOORS][2]bool
	thisElevState 	HRAELevStateInput
	otherElevStates []HRAELevStateInput 
}

type HRAELevStateInput struct{
	Floor                 int
	Dirn                  driver.MotorDirection
	Behaviour             localElevator.ElevatorBehaviour
	CabRequests           [config.N_FLOORS]bool
}

// World View Decider module ------------------------------------------------------------------------------------------

type WorldViewDecider struct {
	localID string //TODO: Vurder om denne er nødvendig

	//Elevator states
	thisElevState   ElevState
	otherElevStates [config.N_ELEVATORS-1]ElevState 	//Index corresponds to ElevID and are kept concistent
	connectedElevators [config.N_ELEVATORS - 1]bool 		//Index corresponds to ElevID and are kept concistent
	
	//Channels
	messageFromLocalElevChannel  <-chan ElevState
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan ElevState //Index corresponds to ElevID and are kept concistent
	hallCallButtonChan           <-chan driver.ButtonEvent
	hallCallAssignerChan chan HRAInput
	toCommCh chan ElevState
	connectedElevatorsCh chan [config.N_ELEVATORS - 1]bool
}

func NewWorldViewModule(
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan ElevState,
	hallCallAssignerChan chan HRAInput,
	driverToWorldviewChan <-chan driver.ButtonEvent,
	toCommCh chan ElevState,
	localElevCh <-chan ElevState,
	localID string,
	connectedElevatorsCh chan [config.N_ELEVATORS - 1]bool,

) *WorldViewDecider {

	w := &WorldViewDecider{
		messageFromOtherElevChannels: messageFromOtherElevChannels,
		hallCallAssignerChan:         hallCallAssignerChan,
		hallCallButtonChan:           driverToWorldviewChan,
		toCommCh:                     toCommCh,
		messageFromLocalElevChannel:  localElevCh,
		connectedElevatorsCh:		  connectedElevatorsCh,

		localID:                      localID, //TODO: Vurder om nødvendig
		thisElevState:                ElevState{},
		otherElevStates:              [config.N_ELEVATORS - 1]ElevState{},
		connectedElevators:			  [config.N_ELEVATORS - 1]bool{},
	}

	go w.loop()

	return w
}

func (w *WorldViewDecider) loop() {
	var hallCallsHasChanged bool
	var elevatorStateHasChanged bool
	checkMessagesFromOtherElevChannels := time.NewTicker(time.Millisecond * 15)

	for {
		hallCallsHasChanged = false
		elevatorStateHasChanged = false

		//Check for message from the local elevator
		select {
		case newElevState := <-w.messageFromLocalElevChannel:
			w.thisElevState = newElevState
			w.sendUpdatedInformationToHallCallAssigner()
			w.sendUpdatedInformationToCommunication()


		//TODO: Sjekk om den logikken er korrekt.
		case hallButtonPressed := <-w.hallCallButtonChan:
			if hallButtonPressed.Button != driver.BT_Cab {
				//fmt.Println("received hallCallButtn press")
				if w.thisElevState.HallCalls[hallButtonPressed.Floor][hallButtonPressed.Button] == NoOrder {
					w.updateSpecifiedHallCallAndLight(Unconfirmed,
						hallButtonPressed.Floor, hallButtonPressed.Button)
					hallCallsHasChanged = true
				}
				//fmt.Println("processed hallbtnpress")
				//fmt.Println(w.thisElevState.HallCalls)
			}
			//TODO? check if the orderState has changed. if yes set hallCallsHasChanged variable to true

		case <-checkMessagesFromOtherElevChannels.C:
			for elevID := 0; elevID < len(w.messageFromOtherElevChannels); elevID++ {
				if w.connectedElevators[elevID] == true{
					select {
					case newElevState := <-w.messageFromOtherElevChannels[elevID]:
						fmt.Println("got message from other elevator")
						if newElevState != w.otherElevStates[elevID] {
							w.otherElevStates[elevID] = newElevState
							elevatorStateHasChanged = true
							w.updateHallCallsAndLights(newElevState.HallCalls)
						}
					default:
					}
				}
			}
		
		case newConnectedElevators := <- w.connectedElevatorsCh:
			w.connectedElevators = newConnectedElevators

		default:
		}


		if elevatorStateHasChanged || hallCallsHasChanged {
			//fmt.Println("sending updated info to HallCallAssigner")
			//fmt.Println(w.thisElevState.HallCalls)
			w.sendUpdatedInformationToHallCallAssigner()
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
		w.updateHallCallsAndLights(incomingElevState.HallCalls)
		hallCallsHasChanged = true
	}
}

func (w *WorldViewDecider) updateHallCallsAndLights(incomingHallCalls [config.N_FLOORS][2]OrderState) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btnType := 0; btnType < 2; btnType++ {
			w.updateSpecifiedHallCallAndLight(incomingHallCalls[floor][btnType], floor, driver.ButtonType(btnType))
		}
	}
}

func (w *WorldViewDecider) updateSpecifiedHallCallAndLight(incomingHallCall OrderState, floor int, hallBtn driver.ButtonType) {
	switch w.thisElevState.HallCalls[floor][hallBtn] {
	case NoOrder:
		switch incomingHallCall {
		case Unconfirmed:
			w.thisElevState.HallCalls[floor][hallBtn] = Unconfirmed
		case Confirmed:
			w.thisElevState.HallCalls[floor][hallBtn] = Confirmed
		default:
		}

	case Unconfirmed:
		switch incomingHallCall {
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
				driver.SetButtonLamp(hallBtn,floor,true)
			}
		case Confirmed:
			w.thisElevState.HallCalls[floor][hallBtn] = Confirmed
			driver.SetButtonLamp(hallBtn,floor,true)
		default:
		}

	case Confirmed:
		switch incomingHallCall {
		case Completed:
			w.thisElevState.HallCalls[floor][hallBtn] = Completed
		default:
		}
	case Completed:
		switch incomingHallCall {
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
				driver.SetButtonLamp(hallBtn,floor,false)

			}
		case NoOrder:
			w.thisElevState.HallCalls[floor][hallBtn] = NoOrder
			driver.SetButtonLamp(hallBtn,floor,false)
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

func (w *WorldViewDecider) sendUpdatedInformationToCommunication() {
	select{	
		case w.toCommCh <- input:
		default:
			<- w.hallCallAssignerChan
			w.hallCallAssignerChan <- input
	}

}

func (w *WorldViewDecider) sendUpdatedInformationToHallCallAssigner() {
	//Transform hallRequestStates to bools
	hallRequestsInput := [config.N_FLOORS][2]bool{}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for dir := 0; dir < 2; dir++ {
			if w.thisElevState.HallCalls[floor][dir] == Confirmed {
				hallRequestsInput[floor][dir] = true
			}
		}
	}

	//Make HRAElevStates from current ElevStates
	thisElevInput := HRAELevStateInput{
	    Floor: 			w.thisElevState.Floor,
		Dirn: 			w.thisElevState.Dirn,
		Behaviour: 		w.thisElevState.Behaviour,
		CabRequests: 	w.thisElevState.CabRequests,
	}

	otherElevStatesInput := []HRAELevStateInput{}
	for elevIndex := 0; elevIndex < config.N_ELEVATORS; elevIndex++ {
		//We only pass on the elevator if it considers itself able to take orders, and is connected to network, and unobstructed
		if w.otherElevStates[elevIndex].AbleToServiceRequests && w.connectedElevators[elevIndex] && !w.otherElevStates[elevIndex].Obstruction {
			elevatorHRAState := HRAELevStateInput {
				Floor: 			w.otherElevStates[elevIndex].Floor,
				Dirn: 			w.otherElevStates[elevIndex].Dirn,
				Behaviour: 		w.otherElevStates[elevIndex].Behaviour,
				CabRequests: 	w.otherElevStates[elevIndex].CabRequests,
			}
			otherElevStatesInput = append(otherElevStatesInput, elevatorHRAState)
		}
	}

	//Collect and pass input to HallRequestAssigner channel
	input := HRAInput{
		HallRequests: hallRequestsInput,
		thisElevState: thisElevInput,
		otherElevStates: otherElevStatesInput,	
	}

	select{	
		case w.hallCallAssignerChan <- input:
		default:
			<- w.hallCallAssignerChan
			w.hallCallAssignerChan <- input
	}

	/*
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
		if !elev.AbleToServiceRequests {
			continue
		}

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
	*/
}

