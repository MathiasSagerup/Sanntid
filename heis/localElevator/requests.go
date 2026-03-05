package localElevator

import (
	"fmt"
	"heis/driver"
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

func requests_chooseDirection(e Elevator) dirnBehaviourPair {
	switch e.dirn {

	case driver.MD_Up:
		if requests_above(e) {
			return dirnBehaviourPair{driver.MD_Up, moving}
		}

		if requests_here(e) {
			return dirnBehaviourPair{driver.MD_Down, doorOpen}
		}

		if requests_below(e) {
			return dirnBehaviourPair{driver.MD_Down, moving}
		}

		return dirnBehaviourPair{driver.MD_Stop, idle}

	case driver.MD_Down:
		if requests_below(e) {
			return dirnBehaviourPair{driver.MD_Down, moving}
		}

		if requests_here(e) {
			return dirnBehaviourPair{driver.MD_Up, doorOpen}
		}

		if requests_above(e) {
			return dirnBehaviourPair{driver.MD_Up, moving}
		}
		return dirnBehaviourPair{driver.MD_Stop, idle}

	case driver.MD_Stop:
		if requests_here(e) {
			return dirnBehaviourPair{driver.MD_Stop, doorOpen}
		}

		if requests_above(e) {
			return dirnBehaviourPair{driver.MD_Up, moving}
		}

		if requests_below(e) {
			return dirnBehaviourPair{driver.MD_Down, moving}
		}

		return dirnBehaviourPair{driver.MD_Stop, idle}

	default:
		return dirnBehaviourPair{driver.MD_Stop, idle}
	}
}

func requests_shouldStop(e Elevator) bool {
	switch e.dirn {

	case driver.MD_Down:
		if e.requests[e.floor][driver.BT_HallDown] || e.requests[e.floor][driver.BT_Cab] || !requests_above(e) {
			return true
		}
		return false

	case driver.MD_Up:
		if e.requests[e.floor][driver.BT_HallUp] || e.requests[e.floor][driver.BT_Cab] || !requests_above(e) {
			return true
		}
		return false

	case driver.MD_Stop:
		return true

	default:
		fmt.Println("requests_ShouldStop reached an undefined state in the elevator MotorDirection")
		return true
	}
}

func requests_shouldClearImmediately(e Elevator, btnFloor int, btnType driver.ButtonType) bool {

	if e.floor == btnFloor {

		if e.dirn == driver.MD_Up && btnType == driver.BT_HallUp {
			return true

		} else if e.dirn == driver.MD_Down && btnType == driver.BT_HallDown {
			return true

		} else if e.dirn == driver.MD_Stop {
			return true

		} else if e.dirn == driver.MD_Stop {
			return true

		} else {
			return false
		}

	} else {
		return false
	}

}

func requests_clearAtCurrentFloor(e Elevator) Elevator {

	e.requests[e.floor][driver.BT_Cab] = false

	switch e.dirn {

	case driver.MD_Up:
		if !requests_above(e) && !e.requests[e.floor][driver.BT_HallUp] {
			e.requests[e.floor][driver.BT_HallDown] = false
		}

		e.requests[e.floor][driver.BT_HallUp] = false
		break

	case driver.MD_Down:
		if !requests_below(e) && !e.requests[e.floor][driver.BT_HallDown] {
			e.requests[e.floor][driver.BT_HallUp] = false
		}

		e.requests[e.floor][driver.BT_HallDown] = false
		break

	case driver.MD_Stop:
		e.requests[e.floor][driver.BT_HallDown] = false
		e.requests[e.floor][driver.BT_HallUp] = false
		break

	default:
		fmt.Println("requests_clearAtCurrentFloor reached an unconsistent MotorDir state")
	}
	return e
}
