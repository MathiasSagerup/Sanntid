package localElevator

import (
	"fmt"
	Driver "heis/driver"
)

//modulen håndterer bestillinger for lokale heisen

func requests_above(e Elevator) bool {
	for f := e.floor + 1; f < N_FLOORS; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requests_below(e Elevator) bool {
	for f := 0; f < e.floor; f++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			if e.requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requests_here(e Elevator) bool {
	for btn := 0; btn < N_BUTTONS; btn++ {
		if e.requests[e.floor][btn] {
			return true
		}
	}
	return false
}

func requests_chooseDirection(e Elevator) DirnBehaviourPair {
	switch e.dirn {

	case Driver.MD_Up:
		if requests_above(e) {
			return DirnBehaviourPair{Driver.MD_Up, moving}
		}

		if requests_here(e) {
			return DirnBehaviourPair{Driver.MD_Down, doorOpen}
		}

		if requests_below(e) {
			return DirnBehaviourPair{Driver.MD_Down, moving}
		}

		return DirnBehaviourPair{Driver.MD_Stop, idle}

	case Driver.MD_Down:
		if requests_below(e) {
			return DirnBehaviourPair{Driver.MD_Down, moving}
		}

		if requests_here(e) {
			return DirnBehaviourPair{Driver.MD_Up, doorOpen}
		}

		if requests_above(e) {
			return DirnBehaviourPair{Driver.MD_Up, moving}
		}
		return DirnBehaviourPair{Driver.MD_Stop, idle}

	case Driver.MD_Stop:
		if requests_here(e) {
			return DirnBehaviourPair{Driver.MD_Stop, doorOpen}
		}

		if requests_above(e) {
			return DirnBehaviourPair{Driver.MD_Up, moving}
		}

		if requests_below(e) {
			return DirnBehaviourPair{Driver.MD_Down, moving}
		}

		return DirnBehaviourPair{Driver.MD_Stop, idle}

	default:
		return DirnBehaviourPair{Driver.MD_Stop, idle}
	}
}

func requests_shouldStop(e Elevator) bool {
	switch e.dirn {

	case Driver.MD_Down:
		if e.requests[e.floor][Driver.BT_HallDown] || e.requests[e.floor][Driver.BT_Cab] || !requests_above(e) {
			return true
		}
		return false

	case Driver.MD_Up:
		if e.requests[e.floor][Driver.BT_HallUp] || e.requests[e.floor][Driver.BT_Cab] || !requests_above(e) {
			return true
		}
		return false

	case Driver.MD_Stop:
		return true

	default:
		fmt.Println("requests_ShouldStop reached an undefined state in the elevator MotorDirection")
		return true
	}
}

func requests_shouldClearImmdeiately(e Elevator, btnFloor int, btnType Driver.ButtonType) bool {

	if e.floor == btnFloor {

		if e.dirn == Driver.MD_Up && btnType == Driver.BT_HallUp {
			return true

		} else if e.dirn == Driver.MD_Down && btnType == Driver.BT_HallDown {
			return true

		} else if e.dirn == Driver.MD_Stop {
			return true

		} else if e.dirn == Driver.MD_Stop {
			return true

		} else {
			return false
		}

	} else {
		return false
	}

}

func requests_clearAtCurrentFloor(e Elevator) Elevator {

	e.requests[e.floor][Driver.BT_Cab] = false

	switch e.dirn {

	case Driver.MD_Up:
		if !requests_above(e) && !e.requests[e.floor][Driver.BT_HallUp] {
			e.requests[e.floor][Driver.BT_HallDown] = false
		}

		e.requests[e.floor][Driver.BT_HallUp] = false
		break

	case Driver.MD_Down:
		if !requests_below(e) && !e.requests[e.floor][Driver.BT_HallDown] {
			e.requests[e.floor][Driver.BT_HallUp] = false
		}

		e.requests[e.floor][Driver.BT_HallDown] = false
		break

	case Driver.MD_Stop:
		e.requests[e.floor][Driver.BT_HallDown] = false
		e.requests[e.floor][Driver.BT_HallUp] = false
		break

	default:
		fmt.Println("requests_clearAtCurrentFloor reached an unconsistent MotorDir state")
	}
	return e
}
