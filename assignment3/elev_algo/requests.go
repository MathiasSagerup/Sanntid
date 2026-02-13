package main

type DirnBehaviourPair struct {
	Dirn dirn;
	ElevatorBehaviour behaviour;
} 


//Jeg tror følgende funksjoner er pure. Dvs at vi ikke trenger å tenke på concurrency her. Samme input -> samme output hver gang. 
//FSM vil ta i bruk select og message passing og concurrency etc. variabler som flere rutiner brukere må oversettes til servere. 


func requests_Above(e Elevator) bool {
    for f := e.floor + 1; f < N_FLOORS; f++ {
        for btn := 0; btn < N_BUTTONS; btn++ {
            if e.requests[f][btn] {
                return true;Elevator requests_clearAtCurrentFloor(Elevator e){
        
    e.requests[e.floor][B_Cab] = 0;
    switch(e.dirn){
    case D_Up:
        if(!requests_above(e) && !e.requests[e.floor][B_HallUp]){
            e.requests[e.floor][B_HallDown] = 0;
        }
        e.requests[e.floor][B_HallUp] = 0;
        break;
        
    case D_Down:
        if(!requests_below(e) && !e.requests[e.floor][B_HallDown]){
            e.requests[e.floor][B_HallUp] = 0;
        }
        e.requests[e.floor][B_HallDown] = 0;
        break;
        
    case D_Stop:
    default:
        e.requests[e.floor][B_HallUp] = 0;
        e.requests[e.floor][B_HallDown] = 0;
        break;
    }
    return e;
}
            }
        }
    }
    return false;
}

func requests_Below(e Elevator) bool {
	for f:=0; f< e.floor; f++ {
		for btn:= 0; btn < N_BUTTONS; btn++ {
			if(e.requests[f][btn]){
				return true;
			}
		}
	}
	return false;
}


func requests_Here(Elevator e) bool {
	for btn:= 0; btn < N_BUTTONS; btn++ {
		if e.requests[e.floor][btn] {
			return true;
		}
	}
	return false;
}

func requests_ChooseDirection(e Elevator) DirnBehaviourPair {
    switch e.dirn {

    case D_Up:
        if requests_Above(e) {
            return DirnBehaviourPair{D_Up, EB_Moving}
        }
        if requests_Here(e) {
            return DirnBehaviourPair{D_Down, EB_DoorOpen}
        }
        if requests_Below(e) {
            return DirnBehaviourPair{D_Down, EB_Moving}
        }

    case D_Down:
        if requests_Below(e) {
            return DirnBehaviourPair{D_Down, EB_Moving}
        }
        if requests_Here(e) {
            return DirnBehaviourPair{D_Up, EB_DoorOpen}
        }
        if requests_Above(e) {
            return DirnBehaviourPair{D_Up, EB_Moving}
        }

    case D_Stop:
        if requests_Here(e) {
            return DirnBehaviourPair{D_Stop, EB_DoorOpen}
        }
        if requests_Above(e) {
            return DirnBehaviourPair{D_Up, EB_Moving}
        }
        if requests_Below(e) {
            return DirnBehaviourPair{D_Down, EB_Moving}
        }
    }

    return DirnBehaviourPair{D_Stop, EB_Idle}
}


func requests_ShouldStop(e Elevator) bool {
    switch e.dirn {

    case D_Down:
        return e.requests[e.floor][B_HallDown] ||
               e.requests[e.floor][B_Cab] ||
               !requests_Below(e)

    case D_Up:
        return e.requests[e.floor][B_HallUp] ||
               e.requests[e.floor][B_Cab] ||
               !requests_Above(e)

    case D_Stop:
        return true
    
	default:
        return true
    }
}

func requests_ShouldClearImmediately(e Elevator, btnFloor int, btnType Button) bool {
    return e.floor == btnFloor && (
        (e.dirn == D_Up && btnType == B_HallUp) ||
        (e.dirn == D_Down && btnType == B_HallDown) ||
        e.dirn == D_Stop ||
        btnType == B_Cab )
}

func requests_ClearAtCurrentFloor(e Elevator) Elevator {
    // Clear the cab request at the current floor
    e.requests[e.floor][B_Cab] = false

    switch e.dirn {

    case D_Up:
        if !requests_Above(e) && !e.requests[e.floor][B_HallUp] {
            e.requests[e.floor][B_HallDown] = false
        }
        e.requests[e.floor][B_HallUp] = false

    case D_Down:
        if !requests_Below(e) && !e.requests[e.floor][B_HallDown] {
            e.requests[e.floor][B_HallUp] = false
        }
        e.requests[e.floor][B_HallDown] = false

    case D_Stop:
        fallthrough
    default:
        e.requests[e.floor][B_HallUp] = false
        e.requests[e.floor][B_HallDown] = false
    }

    return e
}
