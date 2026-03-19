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
	hallCalls 				[config.N_FLOORS][config.N_TRAVEL_DIRN]HallCallWithConfirmation
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
	completedHallCallsCh 	<-chan [config.N_FLOORS][config.N_TRAVEL_DIRN]bool
}

func NewWorldViewModule(
	otherElevChs 			[config.N_OTHER_ELEVATORS]<-chan PeerState,
	HCAInputCh 				chan hallCallsAssigner.HRAInput,
	driverToWorldviewChan 	<-chan driver.ButtonEvent,
	localPeerStateCh 		chan PeerState,
	localElevCh 			<-chan localElevator.ElevState,
	localID 				string,
	connectedElevatorsCh 	<-chan [config.N_OTHER_ELEVATORS]bool,
	completedHallCallsCh 	<-chan [config.N_FLOORS][config.N_TRAVEL_DIRN]bool,
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
	peerPollTicker := time.NewTicker(time.Millisecond * 10)

	for {
		select {
		case newElevState := <-w.localElevCh:
			if w.thisElevState != newElevState {
				w.thisElevState = newElevState
				w.sendHallCallAssignerInput()
				w.sendLocalPeerState()
			}

		//JUSTER SÅNN AT DET KUN ER HALLCALLS OSM KOMMER PÅ HALLCALLSBUTTON CH - I DRIVER? 
		case hallButtonPressed := <-w.hallCallButtonCh:
			hallCallsBeforeCheck := w.hallCalls
			floor := hallButtonPressed.Floor
			button := hallButtonPressed.Button

			if (button == driver.BT_HallDown)||(button== driver.BT_HallUp) {

				if w.hallCalls[floor][button].state == NoOrder {
					if w.isAloneOnNetwork() {
						w.hallCalls[floor][button].state = Confirmed
						driver.SetButtonLamp(button, floor, true)	
					} else {
						w.hallCalls[floor][button].state = Unconfirmed
					}
				}

			}

			if hallCallsBeforeCheck != w.hallCalls {
				w.sendHallCallAssignerInput()
				w.sendLocalPeerState()
			}


		case <-peerPollTicker.C:

			//Check each channel that corresponds to an elevator that is currently connected
			for elevID := range w.otherElevChs {
				if w.connectedElevators[elevID]{
					select {
					case newPeerState := <-w.otherElevChs[elevID]:
						
						//Compare incoming hallcalls with current hallcalls and update if needed
						hallCallsBeforeCheck := w.hallCalls
						w.updateHallCallsAndLights(newPeerState.HallCalls, elevID)
						if hallCallsBeforeCheck != w.hallCalls {
							fmt.Println("[worldview] recieved hallorders:", w.getHallCallsWithoutConfirmation())
							w.sendHallCallAssignerInput()
							w.sendLocalPeerState()
						}
						
						//Check local elev state transition from sender
						if newPeerState.LocalElevState != w.otherElevStates[elevID] {
							w.otherElevStates[elevID] = newPeerState.LocalElevState
							w.sendHallCallAssignerInput()
							w.sendLocalPeerState()
						}
					
					default:
						//No update from peer with this elevID
					}
				}
			}
		
		case newConnectedElevators := <- w.connectedElevatorsCh:
			w.connectedElevators = newConnectedElevators
			w.sendHallCallAssignerInput()
		

		case newCompletedHallCalls := <- w.completedHallCallsCh:
			for floor := 0; floor < config.N_FLOORS; floor++ {
				for dirn := 0; dirn < config.N_TRAVEL_DIRN; dirn++ {

					if (newCompletedHallCalls[floor][dirn]) && (w.hallCalls[floor][dirn].state == Confirmed){
						if w.isAloneOnNetwork(){
							w.hallCalls[floor][dirn].state = NoOrder
							driver.SetButtonLamp(driver.ButtonType(dirn), floor, false)
						} else {
							w.hallCalls[floor][dirn].state = Completed
						}

						w.sendLocalPeerState()
						w.sendHallCallAssignerInput()
					}
				}
			}
		}
	}
}


func (w *WorldView) isAloneOnNetwork() bool {
	if w.getNumberOfConnectedPeers() == 0 {
		return true
	}
	return false
}

func (w *WorldView) sendLocalPeerState() {
	fmt.Printf("[worldview] Current hallcalls are: %v\n", w.hallCalls)
	input := PeerState{w.thisElevState, w.getHallCallsWithoutConfirmation()}
	select{	
		case w.localPeerStateCh <- input:
		default:
			<- w.localPeerStateCh
			w.localPeerStateCh <- input
	}
}

func (w *WorldView) sendHallCallAssignerInput() {
	//Transform hallRequestStates to bools
	hallRequestsInput := [config.N_FLOORS][2]bool{}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for dirn := 0; dirn < config.N_TRAVEL_DIRN; dirn++ {
			if w.hallCalls[floor][dirn].state == Confirmed {
				hallRequestsInput[floor][dirn] = true
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
	for elevIndex := 0; elevIndex < config.N_OTHER_ELEVATORS; elevIndex++ {
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