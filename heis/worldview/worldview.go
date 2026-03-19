package worldview

//The worldview package is responsible for maintaining an updated view of the system state.
//This includes the state of all the elevators, the state of all hallcalls and the connectivity status of each peer. As this module determines hallcall states, it also controls the hallcall lights.
//It takes in this information from the local elevator and all peers on the network through the given channels on initialization of WorldViewDecider
//The module passes on relevant information to the HallRequestAssigner and Communication module through the given channels on initialization of WorldViewDecider

import (
	"fmt"
	"heis/config"
	"heis/driver"
	"heis/hallCallsAssigner"
	"heis/localElevator"
	"time"
)

// World View Decider module ------------------------------------------------------------------------------------------

type WorldView struct {
	//System state
	hallCalls 				[config.N_FLOORS][2]HallCallWithConfirmation
	thisElevState 			localElevator.ElevState
	otherElevStates 		[config.N_OTHER_ELEVATORS]localElevator.ElevState 			//Index corresponds to ElevID and are kept concistent
	connectedElevators		[config.N_OTHER_ELEVATORS]bool 						//Index corresponds to ElevID and are kept concistent
	
	//Channels
	localElevCh 			<-chan localElevator.ElevState
	otherElevChs 			[config.N_OTHER_ELEVATORS]<-chan PeerState 	//Index corresponds to ElevID and are kept concistent
	hallCallButtonCh 		<-chan driver.ButtonEvent
	HCAInputCh 				chan hallCallsAssigner.HRAInput
	localPeerStateCh 		chan PeerState
	connectedElevatorsCh 	<-chan [config.N_OTHER_ELEVATORS]bool
	completedHallCallsCh 	<-chan [config.N_FLOORS][2]bool
}

func NewWorldViewModule(
	otherElevChs 			[config.N_OTHER_ELEVATORS]<-chan PeerState,
	HCAInputCh 				chan hallCallsAssigner.HRAInput,
	driverToWorldviewChan 	<-chan driver.ButtonEvent,
	localPeerStateCh 		chan PeerState,
	localElevCh 			<-chan localElevator.ElevState,
	localID 				string,
	connectedElevatorsCh 	<-chan [config.N_OTHER_ELEVATORS]bool,
	completedHallCallsCh 	<-chan [config.N_FLOORS][2]bool,
) *WorldView {

	w := &WorldView{
		otherElevChs:			otherElevChs,
		HCAInputCh:				HCAInputCh,
		hallCallButtonCh:		driverToWorldviewChan,
		localPeerStateCh:		localPeerStateCh,
		localElevCh:			localElevCh,
		connectedElevatorsCh:	connectedElevatorsCh,
		completedHallCallsCh:	completedHallCallsCh,
		thisElevState:			localElevator.ElevState{},
		otherElevStates:		[config.N_OTHER_ELEVATORS]localElevator.ElevState{},
		connectedElevators:		[config.N_OTHER_ELEVATORS]bool{},
	}

	w.setHallCallLightsOff()

	go w.loop()

	return w
}

func (w *WorldView) loop() {
	checkMessagesFromOtherElevChannelsTick := time.NewTicker(time.Millisecond * 10)

	for {
		select {
		case newElevState := <-w.localElevCh:
			if w.thisElevState != newElevState {
				w.thisElevState = newElevState
				w.sendUpdatedHallcallassignerInput()
				w.sendUpdatedLocalPeerState()
			}

		case hallButtonPressed := <-w.hallCallButtonCh:
			hallCallsBeforeCheck := w.hallCalls

			if (hallButtonPressed.Button == driver.BT_HallDown)|| (hallButtonPressed.Button == driver.BT_HallUp) {
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
				w.sendUpdatedHallcallassignerInput()
				w.sendUpdatedLocalPeerState()
			}


		case <-checkMessagesFromOtherElevChannelsTick.C:

			//Check each channel that corresponds to an elevator that is currently connected
			for elevID := 0; elevID < len(w.otherElevChs); elevID++ {
				if w.connectedElevators[elevID] == true{
					select {
					case newPeerState := <-w.otherElevChs[elevID]:
						
						//Compare incoming hallcalls with current hallcalls and update if needed
						hallCallsBeforeCheck := w.hallCalls
						w.updateHallCallsAndLights(newPeerState.HallCalls, elevID)
						if hallCallsBeforeCheck != w.hallCalls {
							fmt.Println("[worldview] recieved hallorders:", w.getHallCallsWithoutConfirmation())
							w.sendUpdatedHallcallassignerInput()
							w.sendUpdatedLocalPeerState()
						}
						
						//Check local elev state transition from sender
						if newPeerState.LocalElevState != w.otherElevStates[elevID] {
							w.otherElevStates[elevID] = newPeerState.LocalElevState
							w.sendUpdatedHallcallassignerInput()
							w.sendUpdatedLocalPeerState()
						}
					
					default:
						//No update from peer with this elevID
					}
				}
			}
		
		case newConnectedElevators := <- w.connectedElevatorsCh:
			w.connectedElevators = newConnectedElevators
			w.sendUpdatedHallcallassignerInput()
		

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

						w.sendUpdatedLocalPeerState()
						w.sendUpdatedHallcallassignerInput()
					}
				}
			}
		}
	}
}

func (w *WorldView) sendUpdatedLocalPeerState() {
	fmt.Printf("[worldview] Current hallcalls are: %v\n", w.hallCalls)
	input := PeerState{w.thisElevState, w.getHallCallsWithoutConfirmation()}
	select{	
		case w.localPeerStateCh <- input:
		default:
			<- w.localPeerStateCh
			w.localPeerStateCh <- input
	}
}

func (w *WorldView) sendUpdatedHallcallassignerInput() {
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
		case w.HCAInputCh <- input:
		default:
			<- w.HCAInputCh
			w.HCAInputCh <- input
	}
}