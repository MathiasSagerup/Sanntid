package worldview

import (
	//"fmt"
	"fmt"
	"heis/config"
	"heis/driver"

	//	"heis/hallCallsAssigner"
	"heis/localElevator"
	"heis/model"//KODEKVALITET
	//	"strconv"
	"time"
)

type ElevatorBehaviour int

const (
	idle     = 0 //bruke iota som før? 
	moving   = 1
	doorOpen = 2
)

type elevatorID int

//KODEKVALITET: modell ref.
type HallCallWithConfirmation struct {//hallCallWithConfirmation siden den er lokal? 
	state model.OrderState
	confirmation [config.N_ELEVATORS - 1]bool
}
//-----------------

//KODEKVALITET: denne passer inn i model 
//type OrderState int
//
//const (
//	NoOrder OrderState = iota
//	Unconfirmed
//	Confirmed
//	Completed
//)
//--------------

//KODEKVALITET: flytter til model 

//type PeerState struct {
//	LocalElevState localElevator.ElevState
//	HallCalls [config.N_FLOORS][2]OrderState	
//}

//-------------

//hva skiller HRA input fra HRAElevStateInput? Gjør tydlieger  
type HRAInput struct {
	HallRequests  	[config.N_FLOORS][2]bool
	thisElevState 	HRAElevStateInput //ThisElevator
	otherElevStates []HRAElevStateInput //OtherElevators, []HRAElevState
}

type HRAElevStateInput struct{//HRAElevStateInput sier lite om rollen. HRAElevatorState
	Floor                 int
	Dirn                  driver.MotorDirection //svak forkortelse 
	Behaviour             localElevator.ElevatorBehaviour
	CabRequests           [config.N_FLOORS]bool
}


//denne er lokal? sett liten bokstav. Hv gjør den? 
//den holder worldwiew, mottar input fra andre moduler, sender output vider 
type WorldViewDecider struct { //wordwiew er bedre navn? 
	localID string 
	hallCalls [config.N_FLOORS][2]HallCallWithConfirmation
	//KODEKVALITET: endrer eierskap for model typer - kontrakt 

	//var
	thisElevState model.ElevatorState //thisElevator
	otherElevStates [config.N_ELEVATORS-1]model.ElevatorState //otherElevator	
	connectedElevators [config.N_ELEVATORS - 1]bool //connectedElevators		

	//channels
	messageFromLocalElevChannel <-chan model.ElevatorState //vanskleig å tolke. Hva med localElevatorUpdates 
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan model.PeerState //otherElevatorUpdates
	hallCallButtonChan <-chan model.HallCallEvent//hallCallEvents siden det er noe diskret og uventet. 
	hallCallAssignerChan chan HRAInput //hallCallAssignerInput
	toCommCh chan model.PeerState //peerStateUpdates 
	connectedElevatorsCh <-chan [config.N_ELEVATORS - 1]bool //connectedElevatorUpdates
	//---------------------
}

//KODEKVALITET : ref til moduler 
func NewWorldViewModule(
	messageFromOtherElevChannels [config.N_ELEVATORS - 1]<-chan model.PeerState,//peerStateInputs 
	hallCallAssignerChan chan HRAInput, //hallCallAssignerInputs
	driverToWorldviewChan <-chan driver.ButtonEvent, //hallButtonEvents er det den mottar. Endre driver impl. 
	toCommCh chan model.PeerState,//kryptisk. peerStateUpdates 
	localElevCh <-chan model.ElevatorState,//localElevatorStates
	localID string,
	connectedElevatorsCh <-chan [config.N_ELEVATORS - 1]bool, //connectedPeers 
//--------------------------

) *WorldViewDecider {


	//SETT OPP I REKKEEFØLGE TYPEN ER DEKALERERT

	w := &WorldViewDecider{
		messageFromOtherElevChannels: messageFromOtherElevChannels, //
		hallCallAssignerChan:         hallCallAssignerChan, //hallCallAssignerChan:         hallCallAssignerInput,
		hallCallButtonChan:           driverToWorldviewChan,
		toCommCh:                     toCommCh,
		messageFromLocalElevChannel:  localElevCh,
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
	checkMessagesFromOtherElevChannels := time.NewTicker(time.Millisecond * 15) //peerPollTicker

	for {
		select {
		case newElevState := <-w.messageFromLocalElevChannel: //navn må endres 
			w.thisElevState = newElevState
			//DENNE DELEN GJETAS MYE. lag funksjon som publishUpdates() som oppdater HCA og C. De funk under. Bidrar til lesbarhet 
			w.sendUpdatedInformationToHallCallAssigner()
			w.sendUpdatedInformationToCommunication()


		//forslag for det over
		//case localElevatorState := <-w.localElevatorUpdates:
		//	w.localElevatorState = localElevatorState
		//	w.publishUpdates()

		//KODEKVALITET: nå får hall calls kun hall calls, så alt av cab calls filtrering forsvinner: 

		case hallButtonPressed := <-w.hallCallButtonChan: //hallCall siden den holder èn verdi, og hallCallEvents siden det kommer inn fler de over tid
			hallCallsBeforeCheck := w.hallCalls //previousHallCall

			fmt.Println("[worldview] HallCallButton registered")
			//if hallButtonPressed.Button != driver.BT_Cab { //DENNE DELEN FORSVINNER 

			//denne delen er rotete. hva med å trekke ut loakele variable slik:  
			//floor := hallCallEvent.Floor
			//button := hallCallEvent.Button
			//currentCall := &w.hallCalls[floor][button]	

				if w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state == NoOrder {
					if w.getNumberOfConnectedPeers() == 0{
						w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state = Confirmed
						driver.SetButtonLamp(hallButtonPressed.Button, hallButtonPressed.Floor, true)	
					} else {
						w.hallCalls[hallButtonPressed.Floor][hallButtonPressed.Button].state = Unconfirmed
					}
				}
			//}

			if hallCallsBeforeCheck != w.hallCalls {
				w.sendUpdatedInformationToHallCallAssigner()
				w.sendUpdatedInformationToCommunication()
			}
		//---------------------------------


		case <-checkMessagesFromOtherElevChannels.C: //case <-peerPollTicker.C: er bedre 

			
			for elevID := 0; elevID < len(w.messageFromOtherElevChannels); elevID++ { //hva med for elevID := range w.otherElevUpdates {
				if w.connectedElevators[elevID] == true{ //if w.connectedElevators[elevID] holder
					select {
					case newPeerState := <-w.messageFromOtherElevChannels[elevID]: //otherELevatorUpdate := <-w.otherElevators[elevID]:
					
						hallCallsBeforeCheck := w.hallCalls
						w.updateHallCallsAndLights(newPeerState.HallCalls, elevID)
						if hallCallsBeforeCheck != w.hallCalls {
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendUpdatedInformationToCommunication()
						}
						
						
						if newPeerState.LocalElevState != w.otherElevStates[elevID] {
							w.otherElevStates[elevID] = newPeerState.LocalElevState
							w.sendUpdatedInformationToHallCallAssigner()
							w.sendUpdatedInformationToCommunication() 
						}
					
					default:
						
					}
				}
			}
		
		case newConnectedElevators := <- w.connectedElevatorsCh:
			w.connectedElevators = newConnectedElevators
		
		}
	}
}

//navnet er langt og det er ikke tudelig at det gjelder andre peers enn oss selv. Hva med applyIncomingHallCalls. 
//denne itererer over alle inkommende hallcalls og sender inn hver av de inn i funksjon
//applyIncomingHallCalls
func (w *WorldViewDecider) updateHallCallsAndLights(incomingHallCalls [config.N_FLOORS][2]OrderState, senderElevID int) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btnType := 0; btnType < 2; btnType++ { //hallcall er bedre navn
			w.updateSpecifiedHallCallAndLight(incomingHallCalls[floor][btnType], floor, driver.ButtonType(btnType), senderElevID)
		}
	}
}

//hva med applyIncomingHallCall som navn 
func (w *WorldViewDecider) updateSpecifiedHallCallAndLight(incomingHallCall OrderState, floor int, hallBtn driver.ButtonType, senderElevID int) {
	
	//w.hallCalls[floor][hallBtn] nevnes mye og er ikke lesbart. trekk ut lokal variabel slik: 
	//localHallCall:= &w.hallCalls[floor][hallButton]. Da kan vi skile hallcall.state og hallcall.confirmation
	//
	
	switch w.hallCalls[floor][hallBtn].state {
	case NoOrder:
		switch incomingHallCall {
		case Unconfirmed:
			w.hallCalls[floor][hallBtn].state = Unconfirmed
		case Confirmed:
			w.hallCalls[floor][hallBtn].state = Confirmed
			driver.SetButtonLamp(hallBtn,floor,true)
		default:
			
		}

	case Unconfirmed: //altså at egen heis har uncifirmed 
		switch incomingHallCall {
		case Unconfirmed:
			w.hallCalls[floor][hallBtn].confirmation[senderElevID] = true

			
			allConnectedElevatorsHaveConfirmed := true //allConnectedConfirmed
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ { //samme for kom
				if (w.connectedElevators[elevID] == true) && (w.hallCalls[floor][hallBtn].confirmation[elevID] == false){//if (w.connectedElevators[elevID]) && (!localHallCall.confirmation[elevID])
					allConnectedElevatorsHaveConfirmed = false
				}
			}

			if allConnectedElevatorsHaveConfirmed { //navn over
				w.hallCalls[floor][hallBtn].state = Confirmed
				w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
				driver.SetButtonLamp(hallBtn,floor,true)				
			}

		case Confirmed: //hvis incoming er comfirmed setter vi den rett til det
			w.hallCalls[floor][hallBtn].state = Confirmed
			w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} 
			driver.SetButtonLamp(hallBtn,floor,true)
		default:
			
		}

	case Confirmed:
		switch incomingHallCall {
		case Completed:
			w.hallCalls[floor][hallBtn].state = Completed
		default:
			
		}

	case Completed:
		switch incomingHallCall {
		case Completed:
			w.hallCalls[floor][hallBtn].confirmation[senderElevID] = true

			
			allConnectedElevatorsHaveConfirmed := true
			//sette dette i en funksjon allConnectedHasConfirmed() for lesbarhet? Den skal kun sjekke èn ting. 
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
				if (w.connectedElevators[elevID] == true) && (w.hallCalls[floor][hallBtn].confirmation[elevID] == false){
					allConnectedElevatorsHaveConfirmed = false
				}
			}
			//-------------------------

			if allConnectedElevatorsHaveConfirmed {
				w.hallCalls[floor][hallBtn].state = NoOrder
				w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
				driver.SetButtonLamp(hallBtn,floor,false)				
			}

		case NoOrder:
			w.hallCalls[floor][hallBtn].state = NoOrder
			w.hallCalls[floor][hallBtn].confirmation = [config.N_ELEVATORS - 1]bool{} //reset all confirmations to false after transition
			driver.SetButtonLamp(hallBtn,floor,false)
		}
	}
}

//skjønner ikke hva denne gjør. 

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



//


//FORTSETT HERFRA 

//








//skille oppdatering og sending. 
func (w *WorldViewDecider) sendUpdatedInformationToCommunication() {
	//KODEKVALITET: endre ti modell ref. 
	//input := PeerState{w.thisElevState, w.getHallCallsWithoutConfirmation()}
	input := model.PeerState{
		LocalElevState: w.thisElevState,
		HallCalls:      w.getHallCallsWithoutConfirmation(),
	}
	//----------------
	select{	
		case w.toCommCh <- input:
		default:
			<- w.toCommCh
			w.toCommCh <- input
	}
}



//virker som denne gjør mer enn å bare sende. Den regner ut også. 
func (w *WorldViewDecider) sendUpdatedInformationToHallCallAssigner() {
	
	hallRequestsInput := [config.N_FLOORS][2]bool{}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for dir := 0; dir < 2; dir++ {
			if w.hallCalls[floor][dir].state == Confirmed {
				hallRequestsInput[floor][dir] = true
			}
		}
	}


	thisElevInput := HRAElevStateInput{
	    Floor: 			w.thisElevState.Floor,
		Dirn: 			w.thisElevState.Dirn,
		Behaviour: 		w.thisElevState.Behaviour,
		CabRequests: 	w.thisElevState.CabRequests,
	}

	otherElevStatesInput := []HRAElevStateInput{}
	for elevIndex := 0; elevIndex < config.N_ELEVATORS-1; elevIndex++ {
	
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


//Gustav la til dette: 

func (w *WorldViewDecider) publishUpdates() {
	w.sendUpdatedInformationToHallCallAssigner()
	w.sendUpdatedInformationToCommunication()
}

//.----------------
