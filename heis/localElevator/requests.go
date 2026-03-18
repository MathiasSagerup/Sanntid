package localElevator

import (
	"fmt"
	"heis/config"
	"heis/driver"
)

//modulen håndterer bestillinger for lokale heisen

func requestsAbove(e localElevator) bool {
	for f := e.floor + 1; f < config.N_FLOORS; f++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if e.requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsBelow(e localElevator) bool {
	for f := 0; f < e.floor; f++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if e.requests[f][btn] {
				return true
			}
		}
	}
	return false
}

func requestsHere(e localElevator) bool {
	for btn := 0; btn < config.N_BUTTONS; btn++ {
		if e.requests[e.floor][btn] {
			return true
		}
	}
	return false
}

func requestsChooseDirection(e localElevator) dirnBehaviourPair {
	switch e.dirn {

	case driver.MD_Up:
		if requestsAbove(e) {
			return dirnBehaviourPair{driver.MD_Up, moving}
		}

		if requestsHere(e) {
			return dirnBehaviourPair{driver.MD_Down, doorOpen}
		}

		if requestsBelow(e) {
			return dirnBehaviourPair{driver.MD_Down, moving}
		}

		return dirnBehaviourPair{driver.MD_Stop, idle}

	case driver.MD_Down:
		if requestsBelow(e) {
			return dirnBehaviourPair{driver.MD_Down, moving}
		}

		if requestsHere(e) {
			return dirnBehaviourPair{driver.MD_Up, doorOpen}
		}

		if requestsAbove(e) {
			return dirnBehaviourPair{driver.MD_Up, moving}
		}
		return dirnBehaviourPair{driver.MD_Stop, idle}

	case driver.MD_Stop:
		if requestsHere(e) {
			return dirnBehaviourPair{driver.MD_Stop, doorOpen}
		}

		if requestsAbove(e) {
			return dirnBehaviourPair{driver.MD_Up, moving}
		}

		if requestsBelow(e) {
			return dirnBehaviourPair{driver.MD_Down, moving}
		}

		return dirnBehaviourPair{driver.MD_Stop, idle}

	default:
		return dirnBehaviourPair{driver.MD_Stop, idle}
	}
}

func requestsShouldStop(e localElevator) bool {
	switch e.dirn {

	case driver.MD_Down:
		if e.requests[e.floor][driver.BT_HallDown] || e.requests[e.floor][driver.BT_Cab] || !requestsBelow(e) {
			return true
		}
		return false

	case driver.MD_Up:
		if e.requests[e.floor][driver.BT_HallUp] || e.requests[e.floor][driver.BT_Cab] || !requestsAbove(e) {
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

func requestsShouldClearImmediately(e localElevator, btnFloor int, btnType driver.ButtonType) bool {

	if e.floor == btnFloor {

		if e.dirn == driver.MD_Up && btnType == driver.BT_HallUp {
			return true

		} else if e.dirn == driver.MD_Down && btnType == driver.BT_HallDown {
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

func requestsClearAtCurrentFloor(e *localElevator) {

	e.requests[e.floor][driver.BT_Cab] = false

	switch e.dirn {

	case driver.MD_Up:
		if !requestsAbove(*e) && !e.requests[e.floor][driver.BT_HallUp] {
			if e.requests[e.floor][driver.BT_HallDown] {
				e.requests[e.floor][driver.BT_HallDown] = false
				e.completedHallCalls[e.floor][driver.BT_HallDown] = true
				e.sendCompletedHallCallsToWorldView()
			}
		}

		if e.requests[e.floor][driver.BT_HallUp] {
			e.requests[e.floor][driver.BT_HallUp] = false
			e.completedHallCalls[e.floor][driver.BT_HallUp] = true
			e.sendCompletedHallCallsToWorldView()
		}
		break

	case driver.MD_Down:
		if !requestsBelow(*e) && !e.requests[e.floor][driver.BT_HallDown] {
			if e.requests[e.floor][driver.BT_HallUp] {
				e.requests[e.floor][driver.BT_HallUp] = false
				e.completedHallCalls[e.floor][driver.BT_HallUp] = true
				e.sendCompletedHallCallsToWorldView()
			}
		}

		if e.requests[e.floor][driver.BT_HallDown] {
			e.requests[e.floor][driver.BT_HallDown] = false
			e.completedHallCalls[e.floor][driver.BT_HallDown] = true
			e.sendCompletedHallCallsToWorldView()
		}
		break

	case driver.MD_Stop:
		if e.requests[e.floor][driver.BT_HallDown] {
			e.requests[e.floor][driver.BT_HallDown] = false
			e.completedHallCalls[e.floor][driver.BT_HallDown] = true
			e.sendCompletedHallCallsToWorldView()
		}

		if e.requests[e.floor][driver.BT_HallUp] {
			e.requests[e.floor][driver.BT_HallUp] = false
			e.completedHallCalls[e.floor][driver.BT_HallUp] = true
			e.sendCompletedHallCallsToWorldView()
		}
		break

	default:
		fmt.Println("requests_clearAtCurrentFloor reached an unconsistent MotorDir state")
	}
}
