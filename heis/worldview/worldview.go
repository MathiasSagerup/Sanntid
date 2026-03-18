package worldview

import (
	//"fmt"
	"fmt"
	"heis/config"
	"heis/driver"
	"heis/hallCallsAssigner"

	//	"heis/hallCallsAssigner"
	"heis/localElevator"
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
	hallCallAssignerChan chan hallCallsAssigner.HRAInput
	toCommCh chan PeerState
	connectedElevatorsCh <-chan [config.N_ELEVATORS - 1]bool
	completedHallCallsCh <-chan [config.N_FLOORS][2]bool
}

func NewWorldViewModule(
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan PeerState,
	hallCallAssignerChan chan hallCallsAssigner.HRAInput,
	driverToWorldviewChan <-chan driver.ButtonEvent,
	toCommCh chan PeerState,
	localElevCh <-chan localElevator.ElevState,
	localID string,
	connectedElevatorsCh <-chan [config.N_ELEVATORS - 1]bool,
	completedHallCallsCh <-chan [config.N_FLOORS][2]bool,
) *WorldViewDecider {

	w := &WorldViewDecider{
		messageFromOtherElevChannels: messageFromOtherElevChannels,
		hallCallAssignerChan:         hallCallAssignerChan,
		hallCallButtonChan:           driverToWorldviewChan,
		toCommCh:                     toCommCh,
		messageFromLocalElevChannel:  localElevCh,
		connectedElevatorsCh:		  connectedElevatorsCh,
		completedHallCallsCh:	      completedHallCallsCh,

		localID:                      localID, //TODO: Vurder om nødvendig
		thisElevState:                localElevator.ElevState{},
		otherElevStates:              [config.N_ELEVATORS - 1]localElevator.ElevState{},
		connectedElevators:			  [config.N_ELEVATORS - 1]bool{},
	}

	w.setHallCallLightsOff()

	go w.loop()

	return w
}

func (w *WorldViewDecider) loop() {
	checkMessagesFromOtherElevChannels := time.NewTicker(time.Millisecond * 5)

	for {
		select {
		case newElevState := <-w.messageFromLocalElevChannel:
			if w.thisElevState != newElevState {
				w.thisElevState = newElevState
				w.sendUpdatedInformationToHallCallAssigner()
				w.sendUpdatedInformationToCommunication()
			}

		case hallButtonPressed := <-w.hallCallButtonChan:
			hallCallsBeforeCheck := w.hallCalls

			fmt.Println("[worldview] HallCallButton registered")
			if hallButtonPressed.Button != driver.BT_Cab { //TODO: Sjekk med Jens hva denne gjør
				if w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state == NoOrder {
					if w.getNumberOfConnectedPeers() == 0 {
						w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state = Confirmed
						driver.SetButtonLamp(hallButtonPressed.Button, hallButtonPressed.Floor, true)	
					} else {
						w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state = Unconfirmed
					}
				}
			}

			if hallCallsBeforeCheck != w.hallCalls {
				w.sendUpdatedInformationToHallCallAssigner()
				w.sendUpdatedInformationToCommunication()
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
							fmt.Println("[worldview] recieved hallorders:", w.getHallCallsWithoutConfirmation())
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendUpdatedInformationToCommunication()
						}
						
						//Check local elev state transition from sender
						if newPeerState.LocalElevState != w.otherElevStates[elevID] {
							w.otherElevStates[elevID] = newPeerState.LocalElevState
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendUpdatedInformationToCommunication() //TODO: Vurder om vi kan fjerne denne
						}
					
					default:
						//No update from peer with this elevID
					}
				}
			}
		
		case newConnectedElevators := <- w.connectedElevatorsCh:
			w.connectedElevators = newConnectedElevators
		

		case newCompletedHallCalls := <- w.completedHallCallsCh:
			for floor := 0; floor < config.N_FLOORS; floor++ {
				for btnType := 0; btnType < 2; btnType++ {
					if (newCompletedHallCalls[floor][btnType]) && (w.hallCalls[floor][btnType].state == Confirmed){
						if w.getNumberOfConnectedPeers() == 0 {
							w.hallCalls[floor][btnType].state = NoOrder
							driver.SetButtonLamp(driver.ButtonType(btnType), floor, false)
						} else {
							w.hallCalls[floor][btnType].state = Completed
						}

						w.sendUpdatedInformationToCommunication()
						w.sendUpdatedInformationToHallCallAssigner()
					}
				}
			}
		}
	}
}

func (w *WorldViewDecider) updateHallCallsAndLights(incomingHallCalls [config.N_FLOORS][2]OrderState, senderElevID int) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btnType := 0; btnType < 2; btnType++ {
			w.updateSpecifiedHallCallAndLight(incomingHallCalls[floor][btnType], floor, driver.ButtonType(btnType), senderElevID)
		}
	}
}

//TODO: En så avansert sjekk er vel ikke nødvendig for tastetrykk. De må vel bare kunne gå fra no_order til unconfirmed
func (w *WorldViewDecider) updateSpecifiedHallCallAndLight(incomingHallCall OrderState, floor int, hallBtn driver.ButtonType, senderElevID int) {
	switch w.hallCalls[floor][hallBtn].state {
	case NoOrder:
		switch incomingHallCall {
		case NoOrder:
			//Do nothing, is expected when new elevators first connect after initialization
		case Unconfirmed:
			w.hallCalls[floor][hallBtn].state = Unconfirmed
		case Confirmed:
			w.hallCalls[floor][hallBtn].state = Confirmed
			driver.SetButtonLamp(hallBtn,floor,true)
		default:
			fmt.Print("[worldview] Unexpected transition, current hallcall is: ", w.hallCalls[floor][hallBtn].state, "but received: ", incomingHallCall, "floor: ", floor, "direction: ", hallBtn)
		}

	case Unconfirmed:
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
			}

		case Confirmed:
			w.hallCalls[floor][hallBtn].state = Confirmed
			w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
			driver.SetButtonLamp(hallBtn,floor,true)
		
		case Completed:
			w.hallCalls[floor][hallBtn].state = Completed
			w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
			driver.SetButtonLamp(hallBtn,floor,true)
		default:
			fmt.Print("[worldview] Unexpected transition, current hallcall is: ", w.hallCalls[floor][hallBtn].state, "but received: ", incomingHallCall, "floor: ", floor, "direction: ", hallBtn)
		}

	case Confirmed:
		switch incomingHallCall {
		case Confirmed:
			//Do nothing, is expected when an peer corractly changes to confirmes as well
		case Completed:
			w.hallCalls[floor][hallBtn].state = Completed
		default:
			fmt.Print("[worldview] Unexpected transition, current hallcall is: ", w.hallCalls[floor][hallBtn].state, "but received: ", incomingHallCall, "floor: ", floor, "direction: ", hallBtn)
		}

	case Completed:
		switch incomingHallCall {
		case Completed:
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
			}

		case NoOrder:
			w.hallCalls[floor][hallBtn].state = NoOrder
			w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
			driver.SetButtonLamp(hallBtn,floor,false)
		
		default:
			fmt.Print("[worldview] Unexpected transition, current hallcall is: ", w.hallCalls[floor][hallBtn].state, "but received: ", incomingHallCall, "floor: ", floor, "direction: ", hallBtn)
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
		}
	}
	return connectedPeers
}

func (w *WorldViewDecider) sendUpdatedInformationToCommunication() {
	fmt.Printf("[worldview] Current hallcalls are: %v\n", w.hallCalls)
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
			}
		}
	}

	//Make HRAElevStates from current ElevStates
	thisElevInput := localElevator.ElevState{
	    Floor: 			w.thisElevState.Floor,
		Dirn: 			w.thisElevState.Dirn,
		Behaviour: 		w.thisElevState.Behaviour,
		CabRequests: 	w.thisElevState.CabRequests,
	}

	otherElevStatesInput := []localElevator.ElevState{}
	for elevIndex := 0; elevIndex < config.N_ELEVATORS-1; elevIndex++ {
		//We only pass on the elevator if it considers itself able to take orders, and is connected to network, and unobstructed
		if w.otherElevStates[elevIndex].AbleToServiceRequests && w.connectedElevators[elevIndex] && !w.otherElevStates[elevIndex].Obstruction {
			elevatorHRAState := localElevator.ElevState {
				Floor: 			w.otherElevStates[elevIndex].Floor,
				Dirn: 			w.otherElevStates[elevIndex].Dirn,
				Behaviour: 		w.otherElevStates[elevIndex].Behaviour,
				CabRequests: 	w.otherElevStates[elevIndex].CabRequests,
			}
			otherElevStatesInput = append(otherElevStatesInput, elevatorHRAState)
		}
	}

	//Collect and pass input to HallRequestAssigner channel
	input := hallCallsAssigner.HRAInput{
		HallRequests: hallRequestsInput,
		ThisElevState: thisElevInput,
		OtherElevStates: otherElevStatesInput,	
	}

	select{	
		case w.hallCallAssignerChan <- input:
		default:
			<- w.hallCallAssignerChan
			w.hallCallAssignerChan <- input
	}
}

func (w *WorldViewDecider) setHallCallLightsOff() {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if btn == driver.BT_HallDown || btn == driver.BT_HallUp {
				driver.SetButtonLamp(driver.ButtonType(btn), floor, false)
			}
		}
	}
}