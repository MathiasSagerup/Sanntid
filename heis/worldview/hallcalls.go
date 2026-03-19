package worldview

import (
	"heis/config"
	"heis/driver"
	"fmt"
)

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

func (w *WorldViewDecider) getHallCallsWithoutConfirmation() [config.N_FLOORS][2]OrderState {
	hallCallsWithoutConfirmation := [config.N_FLOORS][2]OrderState{}
	for floor:=0; floor<config.N_FLOORS; floor++{
		hallCallsWithoutConfirmation[floor][0] = w.hallCalls[floor][0].state
		hallCallsWithoutConfirmation[floor][1] = w.hallCalls[floor][1].state
	}
	return hallCallsWithoutConfirmation
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

func (w *WorldViewDecider) updateHallCallsAndLights(incomingHallCalls [config.N_FLOORS][2]OrderState, senderElevID int) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btnType := 0; btnType < 2; btnType++ {
			w.updateSpecifiedHallCallAndLight(incomingHallCalls[floor][btnType], floor, driver.ButtonType(btnType), senderElevID)
		}
	}
}

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
