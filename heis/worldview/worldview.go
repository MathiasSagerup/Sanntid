package worldview

import (
	"fmt"
	"heis/config"
	"heis/driver"
	"heis/hallCallsAssigner"
	"heis/localElevator"
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
	state        OrderState
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
	HallCalls      [config.N_FLOORS][2]OrderState
}

// World View Decider module ------------------------------------------------------------------------------------------

type WorldViewDecider struct {
	localID string //TODO: Vurder om denne er nødvendig

	//System state
	hallCalls          [config.N_FLOORS][2]HallCallWithConfirmation
	thisElevState      localElevator.ElevState
	otherElevStates    [config.N_ELEVATORS - 1]localElevator.ElevState //Index corresponds to ElevID and are kept concistent
	connectedElevators [config.N_ELEVATORS - 1]bool                    //Index corresponds to ElevID and are kept concistent

	//Channels
	messageFromLocalElevChannel  <-chan localElevator.ElevState
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan PeerState //Index corresponds to ElevID and are kept concistent
	hallCallButtonChan           <-chan driver.ButtonEvent
	hallCallAssignerChan         chan hallCallsAssigner.HRAInput
	toCommCh                     chan PeerState
	connectedElevatorsCh         <-chan [config.N_ELEVATORS - 1]bool
}

func NewWorldViewModule(
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan PeerState,
	hallCallAssignerChan chan hallCallsAssigner.HRAInput,
	driverToWorldviewChan <-chan driver.ButtonEvent,
	toCommCh chan PeerState,
	localElevCh <-chan localElevator.ElevState,
	localID string,
	connectedElevatorsCh <-chan [config.N_ELEVATORS - 1]bool,
) *WorldViewDecider {

	w := &WorldViewDecider{
		messageFromOtherElevChannels: messageFromOtherElevChannels,
		hallCallAssignerChan:         hallCallAssignerChan,
		hallCallButtonChan:           driverToWorldviewChan,
		toCommCh:                     toCommCh,
		messageFromLocalElevChannel:  localElevCh,
		connectedElevatorsCh:         connectedElevatorsCh,
		localID:                      localID, //TODO: Vurder om nødvendig
		thisElevState:                localElevator.ElevState{},
		otherElevStates:              [config.N_ELEVATORS - 1]localElevator.ElevState{},
		connectedElevators:           [config.N_ELEVATORS - 1]bool{},
	}

	go w.loop()

	return w
}

func (w *WorldViewDecider) loop() {
	checkMessagesFromOtherElevChannels := time.NewTicker(time.Millisecond * 15)

	for {
		select {
		case newElevState := <-w.messageFromLocalElevChannel:
			prevBehaviour := w.thisElevState.Behaviour
			w.thisElevState = newElevState

			// When door opens, mark Confirmed hall calls at this floor as Completed
			if newElevState.Behaviour == localElevator.ElevatorBehaviour(doorOpen) &&
				prevBehaviour != localElevator.ElevatorBehaviour(doorOpen) {
				hallCallsChanged := false
				for btn := 0; btn < 2; btn++ {
					if w.hallCalls[newElevState.Floor][btn].state == Confirmed {
						w.hallCalls[newElevState.Floor][btn].state = Completed
						hallCallsChanged = true
					}
				}
				if hallCallsChanged {
					w.sendUpdatedInformationToHallCallAssigner()
					w.sendStateUpdateToCommunication()
				}
			}

			w.sendUpdatedInformationToHallCallAssigner()
			w.sendStateUpdateToCommunication()

		case hallButtonPressed := <-w.hallCallButtonChan:
			hallCallsBeforeCheck := w.hallCalls

			fmt.Println("[worldview] HallCallButton registered")
			if hallButtonPressed.Button != driver.BT_Cab {
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
				fmt.Print("Update passed")
				w.sendUpdatedInformationToHallCallAssigner()
				w.sendStateUpdateToCommunication()
			}

		case <-checkMessagesFromOtherElevChannels.C:
			for elevID := 0; elevID < len(w.messageFromOtherElevChannels); elevID++ {
				if w.connectedElevators[elevID] == true {
					select {
					case newPeerState := <-w.messageFromOtherElevChannels[elevID]:
						fmt.Println("HallCallsAreBeingChecked", newPeerState.HallCalls)
						hallCallsBeforeCheck := w.hallCalls
						w.updateHallCallsAndLights(newPeerState.HallCalls, elevID)
						if hallCallsBeforeCheck != w.hallCalls {
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendStateUpdateToCommunication()
						}

						if newPeerState.LocalElevState != w.otherElevStates[elevID] {
							fmt.Println("HallCallsWereChanged")
							w.otherElevStates[elevID] = newPeerState.LocalElevState
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendStateUpdateToCommunication()
						}

					default:
					}
				}
			}

		case newConnectedElevators := <-w.connectedElevatorsCh:
			w.connectedElevators = newConnectedElevators
			fmt.Println("[Worldview] Connected elevators updated", w.connectedElevators)
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
		case Unconfirmed:
			w.hallCalls[floor][hallBtn].state = Unconfirmed
		case Confirmed:
			w.hallCalls[floor][hallBtn].state = Confirmed
		default:
			//TODO: Legg til warnings her
		}

	case Unconfirmed:
		switch incomingHallCall {
		case Unconfirmed:
			w.hallCalls[floor][hallBtn].confirmation[senderElevID] = true

			allConnectedElevatorsHaveConfirmed := true
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
				if (w.connectedElevators[elevID] == true) && (w.hallCalls[floor][hallBtn].confirmation[elevID] == false) {
					allConnectedElevatorsHaveConfirmed = false
				}
			}

			if allConnectedElevatorsHaveConfirmed {
				w.hallCalls[floor][hallBtn].state = Confirmed
				w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{}
				driver.SetButtonLamp(hallBtn, floor, true)
			}

		case Confirmed:
			w.hallCalls[floor][hallBtn].state = Confirmed
			w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{}
			driver.SetButtonLamp(hallBtn, floor, true)
		default:
			//TODO: Legg til warnings her
		}

	case Confirmed:
		switch incomingHallCall {
		case Completed:
			w.hallCalls[floor][hallBtn].state = Completed
		default:
			//TODO: Legg til warnings her
		}

	case Completed:
		switch incomingHallCall {
		case Completed:
			w.hallCalls[floor][hallBtn].confirmation[senderElevID] = true

			allConnectedElevatorsHaveConfirmed := true
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
				if (w.connectedElevators[elevID] == true) && (w.hallCalls[floor][hallBtn].confirmation[elevID] == false) {
					allConnectedElevatorsHaveConfirmed = false
				}
			}

			if allConnectedElevatorsHaveConfirmed {
				w.hallCalls[floor][hallBtn].state = NoOrder
				w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{}
				driver.SetButtonLamp(hallBtn, floor, false)
			}

		case NoOrder:
			w.hallCalls[floor][hallBtn].state = NoOrder
			driver.SetButtonLamp(hallBtn, floor, false)
		}
	}
}

func (w *WorldViewDecider) getHallCallsWithoutConfirmation() [config.N_FLOORS][2]OrderState {
	hallCallsWithoutConfirmation := [config.N_FLOORS][2]OrderState{}
	for floor := 0; floor < config.N_FLOORS; floor++ {
		hallCallsWithoutConfirmation[floor][0] = w.hallCalls[floor][0].state
		hallCallsWithoutConfirmation[floor][1] = w.hallCalls[floor][1].state
	}
	return hallCallsWithoutConfirmation
}

func (w *WorldViewDecider) getNumberOfConnectedPeers() int {
	connectedPeers := 0
	for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
		if w.connectedElevators[elevID] == true {
			connectedPeers++
		}
	}
	return connectedPeers
}

func (w *WorldViewDecider) sendStateUpdateToCommunication() {
	input := PeerState{w.thisElevState, w.getHallCallsWithoutConfirmation()}
	select {
	case w.toCommCh <- input:
	default:
		<-w.toCommCh
		w.toCommCh <- input
	}
}

//TODO: Sjekk at denne gir mening med nye structs
func (w *WorldViewDecider) sendUpdatedInformationToHallCallAssigner() {
	hallRequestsInput := [config.N_FLOORS][2]bool{}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for dir := 0; dir < 2; dir++ {
			if w.hallCalls[floor][dir].state == Confirmed {
				hallRequestsInput[floor][dir] = true
			}
		}
	}

	thisElevInput := hallCallsAssigner.HRAElevStateInput{
		Floor:       w.thisElevState.Floor,
		Dirn:        w.thisElevState.Dirn,
		Behaviour:   w.thisElevState.Behaviour,
		CabRequests: w.thisElevState.CabRequests,
	}

	otherElevStatesInput := []hallCallsAssigner.HRAElevStateInput{}
	for elevIndex := 0; elevIndex < config.N_ELEVATORS-1; elevIndex++ {
		if w.otherElevStates[elevIndex].AbleToServiceRequests && w.connectedElevators[elevIndex] && !w.otherElevStates[elevIndex].Obstruction {
			elevatorHRAState := hallCallsAssigner.HRAElevStateInput{
				Floor:       w.otherElevStates[elevIndex].Floor,
				Dirn:        w.otherElevStates[elevIndex].Dirn,
				Behaviour:   w.otherElevStates[elevIndex].Behaviour,
				CabRequests: w.otherElevStates[elevIndex].CabRequests,
			}
			otherElevStatesInput = append(otherElevStatesInput, elevatorHRAState)
		}
	}

	input := hallCallsAssigner.HRAInput{
		HallRequests:    hallRequestsInput,
		ThisElevState:   thisElevInput,
		OtherElevStates: otherElevStatesInput,
	}

	select {
	case w.hallCallAssignerChan <- input:
	default:
		<-w.hallCallAssignerChan
		w.hallCallAssignerChan <- input
	}
}
