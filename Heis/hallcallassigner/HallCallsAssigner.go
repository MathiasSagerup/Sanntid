/*
This module is made to handle hall calls data. It has some main responsibilities:

- Store and update the hall calls state machine based on:
	- new hall calls from local driver
	- external hall call structs from communication module

- Pass assigned hall calls to a channel upon a request
	- The internal elevator state is sent together with the request
	- The elevator alive struct is considered to assign the orders correctly

- Ensure that the most recently updated hall calls struct is stored on a channel
*/

package hallcallassigner

import "github.com/daixiang0/gci/pkg/config"

//------------------------------------------------------------------------------------
//Constants
const NUM_FLOORS int = 4

//------------------------------------------------------------------------------------
//Datatype definitions
type HallCallState int

const (
	NoOrder HallCallState = iota
    Unconfirmed
    Confirmed
    Completed
)

type AssignedHallCalls [NUM_FLOORS][2]bool 				// Up/Down request for each floor
type HallCallStates [NUM_FLOORS][2]HallCallState		// Up/Down request for each floor

type elevatorState struct{
	//Behavior
	//Floor
	//Direction
	//CabRequests
}

type assignedHallCallRequest struct {
	State ElevatorState
	ResponseChan chan<- AssignedHallCalls
}

type internalState struct {
	hallCallsArray [NUM_FLOORS][2]HallCallState
	elevator_other1 ElevatorState
	elevator_other2 ElevatorState
	elevator_local ElevatorState
}

//------------------------------------------------------------------------------------
//HallCallAssigner module

type HallCallAssigner struct {
    // Pointers to other modules
    driver *Driver
    network *Communication
	elevator *Elevator

	IntarfaceChannels {
		chan AssignedHallcalls
	}
    
    // Request channels for this module
    assignedHallCallRequestChan chan assignedHallCallRequest
}

func InitializeHallCallAssigner(driver *Driver, network *Communication) *HallCallAssigner {
    h := &HallCallAssigner{
        driver:    driver,
        network:   network,
		assignedHallCallRequestChan: 	make(chan assignedHallCallRequest),
    }

    go h.runUpdateLoop()

    return h
}

func (h *HallCallAssigner) RequestAssignedHallCalls(state ElevatorState) AssignedHallCalls {
    respChan := make(chan AssignedHallCalls)

    h.assignedHallCallRequestChan <- assignedHallCallRequest{
        State:        state,
        ResponseChan: respChan,
    }

    return <-respChan
}

func (h *HallCallAssigner) computeAssignedHallCalls(state *internalState) AssignedHallCalls{

	return AssignedHallCalls{} //Placeholder
}

func (h *HallCallAssigner) runUpdateLoop() {
    state := internalState{
        hallCallsArray: [NUM_FLOORS][2]HallCallState{},
		elevator_other1: ElevatorState{},
		elevator_other2: ElevatorState{},
		elevator_local: ElevatorState{},
    }

    for {
        select {
			case request := <-h.assignedHallCallRequestChan:
				// Got a hall call request
				state.elevator_local = request.State
				assignedHallCalls := h.computeAssignedHallCalls(&state)
				request.ResponseChan <- assignedHallCalls
			

			default:
				//Nothing add time delay
		}
    }
	elevator.assignedHallcalls <- assignedHallcalls

	
	//read new events from I/O and update hallCallsArray

	//set new output on communication module channel (continous update)
}


// + requestAssigneHallcalls(state, return channel)
// - readDriverEvents()
// - writeToChannelsCommunicaion()




locelElevFSM(){
	elevator Elevator{}
	for{
		elevator = updateStateBasedOnDriverEvents(elevator)

		elevator.assignedHallCalls = hallcallassigner.getAssignedHallcalls(elevator)

		setOuputs(elevator)

		//All koden du har laget
	}
}

updateStateBasedOnDriverEvents(elev, driverChannels){
	select:
		case: newfloor := <-channelFloor
			elevator.floor = newfloor
		case: newObstruction := <-channelObstruction
			elevator.obstruction = newObstruction
		case: newfloor := <-channelFloor
			elevator.floor = newfloor
		defult:
			//ingenting
	
	return elevator
	}

//Dette skal bort

type FaultHandler struct{
	config
	communciationChannels{}
	MyID
}

InitializeFaultHanlder(inputChannels, ID, config){
	faultHandler := FaultHandler{communciationChannels, ID, config}
	go faultHandler.runLoop()
}

(f* faultHandler) runLoop(){
	for{

	}
}(f* faultHandler)





UpdateFromHallcallChannels(Hallcalls, hallCallChannels){
	select{
		case hallcallsA hallCallChannels.elevatorA{
			Hallcalls = compareHallCalls(Hallcalls, hallcallsA)
		}
		case hallcallsB hallCallChannels.elevatorA{
			Hallcalls = compareHallCalls(Hallcalls, hallcallsB)
		}
	}
	return Hallcalls
}

UpdateFromHDriver(Hallcalls, hallCallChannels){
	select{
		case hallcallsA hallCallChannels.elevatorA{
			Hallcalls = compareHallCalls(Hallcalls, hallcallsA)
		}
		case hallcallsB hallCallChannels.elevatorA{
			Hallcalls = compareHallCalls(Hallcalls, hallcallsB)
		}
	}
	return Hallcalls
}