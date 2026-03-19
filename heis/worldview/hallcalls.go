package worldview

import (
	"heis/config"
	"heis/driver"
	"fmt"
)

type HallCallWithConfirmation struct {
	state OrderState
	confirmation [config.N_OTHER_ELEVATORS]bool
}

type OrderState int

const (
	NoOrder OrderState = iota
	Unconfirmed
	Confirmed
	Completed
)

func (w *WorldView) getHallCallsWithoutConfirmation() [config.N_FLOORS][config.N_TRAVEL_DIRN]OrderState {
	hallCallsWithoutConfirmation := [config.N_FLOORS][config.N_TRAVEL_DIRN]OrderState{}
	for floor:=0; floor<config.N_FLOORS; floor++{
		hallCallsWithoutConfirmation[floor][driver.BT_HallUp] = w.hallCalls[floor][driver.BT_HallUp].state
		hallCallsWithoutConfirmation[floor][driver.BT_HallDown] = w.hallCalls[floor][driver.BT_HallDown].state
	}
	return hallCallsWithoutConfirmation
}

func (w *WorldView) setHallCallLightsOff() {
	for floor := 0; floor < config.N_FLOORS; floor++ {
			for dirn:= 0; dirn < config.N_TRAVEL_DIRN; dirn++ {
			driver.SetButtonLamp(driver.ButtonType(dirn), floor, false)
		}
	}
}

func (w *WorldView) updateHallCallsAndLights(incomingHallCalls [config.N_FLOORS][N_TRAVEL_DIRN]OrderState, senderElevID int) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for dirn := 0; dirn < config.N_TRAVEL_DIRN; dirn++ {
			w.updateHallCallAndLight(incomingHallCalls[floor][dirn], floor, driver.ButtonType(dirn), senderElevID)
		}
	}
}

func (w *WorldView) updateHallCallAndLight(incomingHallCall OrderState, floor int, hallBtn driver.ButtonType, senderElevID int) {
	localHallCall := w.hallCalls[floor][hallBtn]

	switch localHallCall.state {
	case NoOrder:
		switch incomingHallCall {
		case NoOrder:
			//Do nothing, is expected when new elevators first connect after initialization
		case Unconfirmed:
			localHallCall.state = Unconfirmed
		case Confirmed:
			localHallCall.state = Confirmed
			driver.SetButtonLamp(hallBtn,floor,true)
		default:
			fmt.Print("[worldview] Unexpected transition, current hallcall is: ", localHallCall.state, "but received: ", incomingHallCall, "floor: ", floor, "direction: ", hallBtn)
		}

	case Unconfirmed:
		switch incomingHallCall {
		case Unconfirmed:
			localHallCall.confirmation[senderElevID] = true

			//Check if all connected elevators now have confirmed for state transition
			allConnectedElevatorsHaveConfirmed := true
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
				if (w.connectedElevators[elevID]) && (!localHallCall.confirmation[elevID]){
					allConnectedElevatorsHaveConfirmed = false
				}
			}

			if allConnectedElevatorsHaveConfirmed {
				localHallCall.state = Confirmed
				localHallCall.confirmation = [config.N_OTHER_ELEVATORS]bool{} //reset all confirmations to false after transition
				driver.SetButtonLamp(hallBtn,floor,true)				
			}

		case Confirmed:
			localHallCall.state = Confirmed
			localHallCall.confirmation = [config.N_OTHER_ELEVATORS]bool{} //reset all confirmations to false after transition
			driver.SetButtonLamp(hallBtn,floor,true)
		
		case Completed:
			localHallCall.state = Completed
			localHallCall.confirmation = [config.N_OTHER_ELEVATORS]bool{} //reset all confirmations to false after transition
			driver.SetButtonLamp(hallBtn,floor,true)
		default:
			fmt.Print("[worldview] Unexpected transition, current hallcall is: ", localHallCall.state, "but received: ", incomingHallCall, "floor: ", floor, "direction: ", hallBtn)
		}

	case Confirmed:
		switch incomingHallCall {
		case Confirmed:
			//Do nothing, is expected when an peer corractly changes to confirmes as well
		case Completed:
			localHallCall.state = Completed
		default:
			fmt.Print("[worldview] Unexpected transition, current hallcall is: ", localHallCall.state, "but received: ", incomingHallCall, "floor: ", floor, "direction: ", hallBtn)
		}

	case Completed:
		switch incomingHallCall {
		case Completed:
			localHallCall.confirmation[senderElevID] = true

			//Check if all connected elevators now have confirmed for state transition
			allConnectedElevatorsHaveConfirmed := true
			for elevID := 0; elevID < len(w.connectedElevators); elevID++ {
				if (w.connectedElevators[elevID]) && (!localHallCall.confirmation[elevID]){
					allConnectedElevatorsHaveConfirmed = false
				}
			}

			if allConnectedElevatorsHaveConfirmed {
				localHallCall.state = NoOrder
				localHallCall.confirmation = [config.N_OTHER_ELEVATORS]bool{} //reset all confirmations to false after transition
				driver.SetButtonLamp(hallBtn,floor,false)				
			}

		case NoOrder:
			localHallCall.state = NoOrder
			localHallCall.confirmation = [config.N_OTHER_ELEVATORS]bool{} //reset all confirmations to false after transition
			driver.SetButtonLamp(hallBtn,floor,false)
		
		default:
			fmt.Print("[worldview] Unexpected transition, current hallcall is: ", localHallCall.state, "but received: ", incomingHallCall, "floor: ", floor, "direction: ", hallBtn)
		}
	}
}
