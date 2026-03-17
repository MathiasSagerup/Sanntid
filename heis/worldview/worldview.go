package worldview

import (
	//"fmt"
	//"fmt"
	"fmt"
	"heis/config"
	"heis/driver"

	//	"heis/hallCallsAssigner"

	//	"heis/hallCallsAssigner"
	"heis/localElevator"
	//	"strconv"
	"time"
	//	"strconv"
	"time"
)

type ElevatorBehaviour int

const (
	idle     = 0
	moving   = 1
	doorOpen = 2
)

type elevatorID int

type HallCallWithConfirmation struct {
	state OrderState
	confirmation [config.N_ELEVATORS - 1]bool
type HallCallWithConfirmation struct {
	state OrderState
	confirmation [config.N_ELEVATORS - 1]bool
}

type OrderState int

const (
	NoOrder OrderState = iota
	Unconfirmed
	Confirmed
	Completed
)

type PeerState struct {
	LocalElevState localElevator.ElevState
	HallCalls [config.N_FLOORS][2]OrderState	
}

type HRAInput struct {
	HallRequests  	[config.N_FLOORS][2]bool
	thisElevState 	HRAElevStateInput
	otherElevStates []HRAElevStateInput 
}

type HRAElevStateInput struct{
	Floor                 int
	Dirn                  driver.MotorDirection
	Behaviour             localElevator.ElevatorBehaviour
	CabRequests           [config.N_FLOORS]bool
}

type PeerState struct {
	LocalElevState localElevator.ElevState
	HallCalls [config.N_FLOORS][2]OrderState	
}

type HRAInput struct {
	HallRequests  	[config.N_FLOORS][2]bool
	thisElevState 	HRAElevStateInput
	otherElevStates []HRAElevStateInput 
}

type HRAElevStateInput struct{
	Floor                 int
	Dirn                  driver.MotorDirection
	Behaviour             localElevator.ElevatorBehaviour
	CabRequests           [config.N_FLOORS]bool
}

// World View Decider module ------------------------------------------------------------------------------------------

type WorldViewDecider struct {
	localID string //TODO: Vurder om denne er nødvendig

	//System state
	hallCalls [config.N_FLOORS][2]HallCallWithConfirmation
	thisElevState localElevator.ElevState
	otherElevStates [config.N_ELEVATORS-1]localElevator.ElevState 	//Index corresponds to ElevID and are kept concistent
	connectedElevators [config.N_ELEVATORS - 1]bool 		//Index corresponds to ElevID and are kept concistent
	
	//Channels
	messageFromLocalElevChannel <-chan localElevator.ElevState
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan PeerState //Index corresponds to ElevID and are kept concistent
	hallCallButtonChan <-chan driver.ButtonEvent
	hallCallAssignerChan chan HRAInput
	toCommCh chan PeerState
	connectedElevatorsCh <-chan [config.N_ELEVATORS - 1]bool
	localID string //TODO: Vurder om denne er nødvendig

	//System state
	hallCalls [config.N_FLOORS][2]HallCallWithConfirmation
	thisElevState localElevator.ElevState
	otherElevStates [config.N_ELEVATORS-1]localElevator.ElevState 	//Index corresponds to ElevID and are kept concistent
	connectedElevators [config.N_ELEVATORS - 1]bool 		//Index corresponds to ElevID and are kept concistent
	
	//Channels
	messageFromLocalElevChannel <-chan localElevator.ElevState
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan PeerState //Index corresponds to ElevID and are kept concistent
	hallCallButtonChan <-chan driver.ButtonEvent
	hallCallAssignerChan chan HRAInput
	toCommCh chan PeerState
	connectedElevatorsCh <-chan [config.N_ELEVATORS - 1]bool
}

func NewWorldViewModule(
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan PeerState,
	hallCallAssignerChan chan HRAInput,
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan PeerState,
	hallCallAssignerChan chan HRAInput,
	driverToWorldviewChan <-chan driver.ButtonEvent,
	toCommCh chan PeerState,
	toCommCh chan PeerState,
	localElevCh <-chan localElevator.ElevState,
	localID string,
	connectedElevatorsCh <-chan [config.N_ELEVATORS - 1]bool,

	connectedElevatorsCh <-chan [config.N_ELEVATORS - 1]bool,

) *WorldViewDecider {

	w := &WorldViewDecider{
		messageFromOtherElevChannels: messageFromOtherElevChannels,
		hallCallAssignerChan:         hallCallAssignerChan,
		hallCallButtonChan:           driverToWorldviewChan,
		toCommCh:                     toCommCh,
		messageFromLocalElevChannel:  localElevCh,
		connectedElevatorsCh:		  connectedElevatorsCh,

		localID:                      localID, //TODO: Vurder om nødvendig
		thisElevState:                localElevator.ElevState{},
		otherElevStates:              [config.N_ELEVATORS - 1]localElevator.ElevState{},
		connectedElevators:			  [config.N_ELEVATORS - 1]bool{},
		connectedElevatorsCh:		  connectedElevatorsCh,

		localID:                      localID, //TODO: Vurder om nødvendig
		thisElevState:                localElevator.ElevState{},
		otherElevStates:              [config.N_ELEVATORS - 1]localElevator.ElevState{},
		connectedElevators:			  [config.N_ELEVATORS - 1]bool{},
	}

	go w.loop()

	return w
}

func (w *WorldViewDecider) loop() {
	checkMessagesFromOtherElevChannels := time.NewTicker(time.Millisecond * 15)

	checkMessagesFromOtherElevChannels := time.NewTicker(time.Millisecond * 15)

	for {
		select {
		case newElevState := <-w.messageFromLocalElevChannel:
			w.thisElevState = newElevState
			w.sendUpdatedInformationToHallCallAssigner()
			w.sendStateUpdateToCommunication()

		case hallButtonPressed := <-w.hallCallButtonChan:
			hallCallsBeforeCheck := w.hallCalls

			fmt.Println("[worldview] HallCallButton registered")
			if hallButtonPressed.Button != driver.BT_Cab { //TODO: Sjekk med Jens hva denne gjør
				if w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state == NoOrder {
					if w.getNumberOfConnectedPeers() == 0{
						w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state = Confirmed
						driver.SetButtonLamp(hallButtonPressed.Button, hallButtonPressed.Floor, true)	
					} else {
						w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state = Unconfirmed
					}
				}
			}

			if hallCallsBeforeCheck != w.hallCalls {
				fmt.Print("Update passed")
				w.sendUpdatedInformationToHallCallAssigner()
				w.sendStateUpdateToCommunication()
			}


		case <-checkMessagesFromOtherElevChannels.C:

			//Check each channel that corresponds to an elevator that is currently connected
			for elevID := 0; elevID < len(w.messageFromOtherElevChannels); elevID++ {
				if w.connectedElevators[elevID] == true{
					select {
					case newPeerState := <-w.messageFromOtherElevChannels[elevID]:
						
						//Check state transition with new hallcalls
						hallCallsBeforeCheck := w.hallCalls
						w.updateHallCallsAndLights(newPeerState.HallCalls, elevID)
						if hallCallsBeforeCheck != w.hallCalls {
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendStateUpdateToCommunication()
						}
						
						//Check local elev state transition from sender
						if newPeerState.LocalElevState != w.otherElevStates[elevID] {
							w.otherElevStates[elevID] = newPeerState.LocalElevState
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendStateUpdateToCommunication()
						}
					
					default:
						//No update from peer with this elevID
					}
				}
			}
		
		case newConnectedElevators := <- w.connectedElevatorsCh:
			w.connectedElevators = newConnectedElevators
		
		}
	}
			w.thisElevState = newElevState
			w.sendUpdatedInformationToHallCallAssigner()
			w.sendStateUpdateToCommunication()

		case hallButtonPressed := <-w.hallCallButtonChan:
			hallCallsBeforeCheck := w.hallCalls

			fmt.Println("[worldview] HallCallButton registered")
			if hallButtonPressed.Button != driver.BT_Cab { //TODO: Sjekk med Jens hva denne gjør
				if w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state == NoOrder {
					if w.getNumberOfConnectedPeers() == 0{
						w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state = Confirmed
						driver.SetButtonLamp(hallButtonPressed.Button, hallButtonPressed.Floor, true)	
					} else {
						w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state = Unconfirmed
					}
				}
			}

			if hallCallsBeforeCheck != w.hallCalls {
				w.sendUpdatedInformationToHallCallAssigner()
				w.sendStateUpdateToCommunication()
			}


		case <-checkMessagesFromOtherElevChannels.C:
			//fmt.Println("NewMessagRegistered")
			//Check each channel that corresponds to an elevator that is currently connected
			for elevID := 0; elevID < len(w.messageFromOtherElevChannels); elevID++ {
				if w.connectedElevators[elevID] == true{
					select {
					case newPeerState := <-w.messageFromOtherElevChannels[elevID]:
						
						//Check state transition with new hallcalls
						fmt.Println("HallCallsAreBeingChecked", newPeerState.HallCalls)
						hallCallsBeforeCheck := w.hallCalls
						w.updateHallCallsAndLights(newPeerState.HallCalls, elevID)
						if hallCallsBeforeCheck != w.hallCalls {
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendStateUpdateToCommunication()
						}
						
						//Check local elev state transition from sender
						if newPeerState.LocalElevState != w.otherElevStates[elevID] {
							fmt.Println("HallCallsWereChanged")
							w.otherElevStates[elevID] = newPeerState.LocalElevState
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendStateUpdateToCommunication()
						}
					
					default:
						//No update from peer with this elevID
					}
				}
			}
		
		case newConnectedElevators := <- w.connectedElevatorsCh:

			w.connectedElevators = newConnectedElevators
			fmt.Println("[Worldview] Connected elevators updated", w.connectedElevators)
		}
	}
}

func (w *WorldViewDecider) updateHallCallsAndLights(incomingHallCalls [config.N_FLOORS][2]OrderState, senderElevID int) {
func (w *WorldViewDecider) updateHallCallsAndLights(incomingHallCalls [config.N_FLOORS][2]OrderState, senderElevID int) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btnType := 0; btnType < 2; btnType++ {
			w.updateSpecifiedHallCallAndLight(incomingHallCalls[floor][btnType], floor, driver.ButtonType(btnType), senderElevID)
			w.updateSpecifiedHallCallAndLight(incomingHallCalls[floor][btnType], floor, driver.ButtonType(btnType), senderElevID)
		}
	}
}

//TODO: En så avansert sjekk er vel ikke nødvendig for tastetrykk. De må vel bare kunne gå fra no_order til unconfirmed
func (w *WorldViewDecider) updateSpecifiedHallCallAndLight(incomingHallCall OrderState, floor int, hallBtn driver.ButtonType, senderElevID int) {
	switch w.hallCalls[floor][hallBtn].state {
//TODO: En så avansert sjekk er vel ikke nødvendig for tastetrykk. De må vel bare kunne gå fra no_order til unconfirmed
func (w *WorldViewDecider) updateSpecifiedHallCallAndLight(incomingHallCall OrderState, floor int, hallBtn driver.ButtonType, senderElevID int) {
	switch w.hallCalls[floor][hallBtn].state {
	case NoOrder:
		switch incomingHallCall {
		switch incomingHallCall {
		case Unconfirmed:
			w.hallCalls[floor][hallBtn].state = Unconfirmed
			w.hallCalls[floor][hallBtn].state = Unconfirmed
		case Confirmed:
			w.hallCalls[floor][hallBtn].state = Confirmed
			w.hallCalls[floor][hallBtn].state = Confirmed
		default:
			//TODO: Legg til warnings her
			//TODO: Legg til warnings her
		}

	case Unconfirmed:
		switch incomingHallCall {
		switch incomingHallCall {
		case Unconfirmed:
			w.hallCalls[floor][hallBtn].confirmation[senderElevID] = true

			//Check if all connected elevators now have confirmed for state transition
			allConnectedElevatorsHaveConfirmed := true
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
				if (w.connectedElevators[elevID] == true) && (w.hallCalls[floor][hallBtn].confirmation[elevID] == false){
					allConnectedElevatorsHaveConfirmed = false
				}
			}

			if allConnectedElevatorsHaveConfirmed {
				w.hallCalls[floor][hallBtn].state = Confirmed
				w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
				driver.SetButtonLamp(hallBtn,floor,true)				
			w.hallCalls[floor][hallBtn].confirmation[senderElevID] = true

			//Check if all connected elevators now have confirmed for state transition
			allConnectedElevatorsHaveConfirmed := true
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
				if (w.connectedElevators[elevID] == true) && (w.hallCalls[floor][hallBtn].confirmation[elevID] == false){
					allConnectedElevatorsHaveConfirmed = false
				}
			}

			if allConnectedElevatorsHaveConfirmed {
				w.hallCalls[floor][hallBtn].state = Confirmed
				w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
				driver.SetButtonLamp(hallBtn,floor,true)				
			}


		case Confirmed:
			w.hallCalls[floor][hallBtn].state = Confirmed
			w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
			w.hallCalls[floor][hallBtn].state = Confirmed
			w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
			driver.SetButtonLamp(hallBtn,floor,true)
		default:
			//TODO: Legg til warnings her
			//TODO: Legg til warnings her
		}

	case Confirmed:
		switch incomingHallCall {
		switch incomingHallCall {
		case Completed:
			w.hallCalls[floor][hallBtn].state = Completed
			w.hallCalls[floor][hallBtn].state = Completed
		default:
			//TODO: Legg til warnings her
			//TODO: Legg til warnings her
		}


	case Completed:
		switch incomingHallCall {
		switch incomingHallCall {
		case Completed:
			w.hallCalls[floor][hallBtn].confirmation[senderElevID] = true

			//Check if all connected elevators now have confirmed for state transition
			allConnectedElevatorsHaveConfirmed := true
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
				if (w.connectedElevators[elevID] == true) && (w.hallCalls[floor][hallBtn].confirmation[elevID] == false){
					allConnectedElevatorsHaveConfirmed = false
			w.hallCalls[floor][hallBtn].confirmation[senderElevID] = true

			//Check if all connected elevators now have confirmed for state transition
			allConnectedElevatorsHaveConfirmed := true
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
				if (w.connectedElevators[elevID] == true) && (w.hallCalls[floor][hallBtn].confirmation[elevID] == false){
					allConnectedElevatorsHaveConfirmed = false
				}
			}

			if allConnectedElevatorsHaveConfirmed {
				w.hallCalls[floor][hallBtn].state = NoOrder
				w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
				driver.SetButtonLamp(hallBtn,floor,false)				

			if allConnectedElevatorsHaveConfirmed {
				w.hallCalls[floor][hallBtn].state = NoOrder
				w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
				driver.SetButtonLamp(hallBtn,floor,false)				
			}


		case NoOrder:
			w.hallCalls[floor][hallBtn].state = NoOrder
			w.hallCalls[floor][hallBtn].state = NoOrder
			driver.SetButtonLamp(hallBtn,floor,false)
		}
	}
}

func (w *WorldViewDecider) getHallCallsWithoutConfirmation() [config.N_FLOORS][2]OrderState {
	hallCallsWithoutConfirmation := [config.N_FLOORS][2]OrderState{}
	for floor:=0; floor<config.N_FLOORS; floor++{
		hallCallsWithoutConfirmation[floor][0] = w.hallCalls[floor][0].state
		hallCallsWithoutConfirmation[floor][1] = w.hallCalls[floor][1].state
	}
	return hallCallsWithoutConfirmation
}

func (w *WorldViewDecider) getNumberOfConnectedPeers() int{
	connectedPeers := 0
	for elevID := 0; elevID < len(w.connectedElevators); elevID++{
		if w.connectedElevators[elevID] == true {
			connectedPeers++
func (w *WorldViewDecider) getHallCallsWithoutConfirmation() [config.N_FLOORS][2]OrderState {
	hallCallsWithoutConfirmation := [config.N_FLOORS][2]OrderState{}
	for floor:=0; floor<config.N_FLOORS; floor++{
		hallCallsWithoutConfirmation[floor][0] = w.hallCalls[floor][0].state
		hallCallsWithoutConfirmation[floor][1] = w.hallCalls[floor][1].state
	}
	return hallCallsWithoutConfirmation
}

func (w *WorldViewDecider) getNumberOfConnectedPeers() int{
	connectedPeers := 0
	for elevID := 0; elevID < len(w.connectedElevators); elevID++{
		if w.connectedElevators[elevID] == true {
			connectedPeers++
		}
	}
	return connectedPeers
	return connectedPeers
}

func (w *WorldViewDecider) sendStateUpdateToCommunication() {
	input := PeerState{w.thisElevState, w.getHallCallsWithoutConfirmation()}
	select{	
		case w.toCommCh <- input:
		default:
			<- w.toCommCh
			w.toCommCh <- input
	}
}

//TODO: Sjekk at denne gir mening med nye structs
func (w *WorldViewDecider) sendUpdatedInformationToHallCallAssigner() {
	//Transform hallRequestStates to bools
	hallRequestsInput := [config.N_FLOORS][2]bool{}
func (w *WorldViewDecider) sendStateUpdateToCommunication() {
	input := PeerState{w.thisElevState, w.getHallCallsWithoutConfirmation()}
	select{	
		case w.toCommCh <- input:
		default:
			<- w.toCommCh
			w.toCommCh <- input
	}
}

//TODO: Sjekk at denne gir mening med nye structs
func (w *WorldViewDecider) sendUpdatedInformationToHallCallAssigner() {
	//Transform hallRequestStates to bools
	hallRequestsInput := [config.N_FLOORS][2]bool{}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for dir := 0; dir < 2; dir++ {
			if w.hallCalls[floor][dir].state == Confirmed {
				hallRequestsInput[floor][dir] = true
			if w.hallCalls[floor][dir].state == Confirmed {
				hallRequestsInput[floor][dir] = true
			}
		}
	}

	//Make HRAElevStates from current ElevStates
	thisElevInput := HRAElevStateInput{
	    Floor: 			w.thisElevState.Floor,
		Dirn: 			w.thisElevState.Dirn,
		Behaviour: 		w.thisElevState.Behaviour,
		CabRequests: 	w.thisElevState.CabRequests,
	}

	otherElevStatesInput := []HRAElevStateInput{}
	for elevIndex := 0; elevIndex < config.N_ELEVATORS-1; elevIndex++ {
		//We only pass on the elevator if it considers itself able to take orders, and is connected to network, and unobstructed
		if w.otherElevStates[elevIndex].AbleToServiceRequests && w.connectedElevators[elevIndex] && !w.otherElevStates[elevIndex].Obstruction {
			elevatorHRAState := HRAElevStateInput {
				Floor: 			w.otherElevStates[elevIndex].Floor,
				Dirn: 			w.otherElevStates[elevIndex].Dirn,
				Behaviour: 		w.otherElevStates[elevIndex].Behaviour,
				CabRequests: 	w.otherElevStates[elevIndex].CabRequests,
			}
			otherElevStatesInput = append(otherElevStatesInput, elevatorHRAState)
	//Make HRAElevStates from current ElevStates
	thisElevInput := HRAElevStateInput{
	    Floor: 			w.thisElevState.Floor,
		Dirn: 			w.thisElevState.Dirn,
		Behaviour: 		w.thisElevState.Behaviour,
		CabRequests: 	w.thisElevState.CabRequests,
	}

	otherElevStatesInput := []HRAElevStateInput{}
	for elevIndex := 0; elevIndex < config.N_ELEVATORS-1; elevIndex++ {
		//We only pass on the elevator if it considers itself able to take orders, and is connected to network, and unobstructed
		if w.otherElevStates[elevIndex].AbleToServiceRequests && w.connectedElevators[elevIndex] && !w.otherElevStates[elevIndex].Obstruction {
			elevatorHRAState := HRAElevStateInput {
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
}


