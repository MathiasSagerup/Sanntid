package hallCallsAssigner

//This module utilizes the HallRequestAssigner which was handed out.
//It takes in inputs given with the HRAIput type, and passes its output as assigned hallcalls, at the given channels at initialization.

import (
	"encoding/json"
	"fmt"
	"heis/config"
	"heis/localElevator"
	"os/exec"
	
)


type hraFormattedElevState struct {
	Behavior    string                `json:"behaviour"`
	Floor       int                   `json:"floor"`
	Direction   string                `json:"direction"`
	CabRequests [config.N_FLOORS]bool `json:"cabRequests"`
}

type hraFormattedInput struct {
	HallRequests [config.N_FLOORS][config.N_TRAVEL_DIRN]bool        `json:"hallRequests"`
	States       map[string]hraFormattedElevState 					`json:"states"`
}

type HRAInput struct {
	HallRequests    [config.N_FLOORS][config.N_TRAVEL_DIRN]bool
	ThisElevState   localElevator.ElevState
	OtherElevStates []localElevator.ElevState
}

type HallCallAssigner struct {
	//Channels
	HRAInputChan chan HRAInput
	HRAOutputChan chan [config.N_FLOORS][config.N_TRAVEL_DIRN]bool

	//internalStates
	thisElevID    string
	hraExecutable string
}

func NewHallCallAssigner(InputChan chan HRAInput, OutputChan chan [config.N_FLOORS][config.N_TRAVEL_DIRN]bool) *HallCallAssigner {

	h := &HallCallAssigner{
		HRAInputChan:  InputChan,
		HRAOutputChan: OutputChan,
		thisElevID:    "elev0",
		hraExecutable: config.HallCallAssignerExec,
	}

	go h.run()
	return h
}

func (h *HallCallAssigner) run() {

	for {
		select {
		case hraInput := <-h.HRAInputChan:

			inputMap := map[string]hraFormattedElevState{
				h.thisElevID: {
					Behavior:    hraInput.ThisElevState.Behaviour.String(),
					Floor:       hraInput.ThisElevState.Floor,
					Direction:   hraInput.ThisElevState.Dirn.String(),
					CabRequests: hraInput.ThisElevState.CabRequests,
				},
			}

			for otherElevIndex := 0; otherElevIndex < len(hraInput.OtherElevStates); otherElevIndex++ {
				inputMap["elev"+fmt.Sprint(otherElevIndex+1)] = hraFormattedElevState{
					Behavior:    hraInput.OtherElevStates[otherElevIndex].Behaviour.String(),
					Floor:       hraInput.OtherElevStates[otherElevIndex].Floor,
					Direction:   hraInput.OtherElevStates[otherElevIndex].Dirn.String(),
					CabRequests: hraInput.OtherElevStates[otherElevIndex].CabRequests,
				}
			}

			input := hraFormattedInput{
				HallRequests: hraInput.HallRequests,
				States:       inputMap,
			}

			jsonBytes, err := json.Marshal(input)
			if err != nil {
				fmt.Println("json.Marshal error: ", err)
				return
			}

			ret, err := exec.Command(h.hraExecutable, "-i", string(jsonBytes)).CombinedOutput()
			if err != nil {
				fmt.Println("exec.Command error: ", err)
				fmt.Println(string(ret))
				return
			}

			output := new(map[string][config.N_FLOORS][2]bool)
			err = json.Unmarshal(ret, &output)
			if err != nil {
				fmt.Println("json.Unmarshal error: ", err)
				return
			}

			h.sendAssignedHallcalls((*output)[h.thisElevID])
		}
	}
}

func (h *HallCallAssigner) sendAssignedHallcalls(hallCalls [config.N_FLOORS][config.N_TRAVEL_DIRN]bool) {
	select {
	case h.HRAOutputChan <- hallCalls:
	default:
		<-h.HRAOutputChan
		h.HRAOutputChan <- hallCalls
	}
}
