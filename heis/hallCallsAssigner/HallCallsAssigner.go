package hallCallsAssigner

import (
	"encoding/json"
	"fmt"
	"heis/config"
	"os/exec"
	"runtime"
)

//chan for elevStateA
//chan for elevStateB
//for c

type HRAElevState struct {
	Behaviour   string `json:"behaviour"`
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"`
	CabRequests []bool `json:"cabRequests"`
}

type HRAInput struct {
	HallRequests [][2]bool                     `json:"hallRequests"`
	States       map[string]HRAElevState       `json:"states"`
}

type HRAOutput struct {
	HallRequests [][2]bool 
}

type HallCallAssigner struct {

	//inptu channel from worldview
	HRAInputChan chan HRAInput

	//output channel to localElevator
	HRAOutputChan chan HRAOutput
	
	//internalStates
	input HRAInput
	ouutput HRAOutput
}

func newHallCallAssigner (HRAInputChan chan HRAInput, HallRequestsToLocalElevator chan HRAOutput) *HallCallAssigner {
	
	h:= &HallCallAssigner{
		HRAInputChan: HRAInputChan,
		HRAOutputChan: HallRequestsToLocalElevator,
	}
	
	go h.run(HRAInputChan,HallRequestsToLocalElevator)
	return h
}


func (h *HallCallAssigner) run(HRAInputChan chan HRAInput, HallRequestToLocalElevator chan HRAOutput) {

	for {
		select{

		case input := <-HRAInputChan:
			Assign 

		}
	}

}
// Assign calls the hall_request_assigner binary and returns assigned hall requests per elevator.
// Returns map[elevatorID] -> [floor][2]bool (up, down per floor)

func Assign(hallRequests [config.N_Floors][2]bool, states map[string]HRAElevState) (map[string][config.N_Floors][2]bool, error) {
	executable := config.HallCallAssignerExec
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}

	hallReqs := make([][2]bool, config.N_Floors)
	for f := 0; f < config.N_Floors; f++ {
		hallReqs[f] = hallRequests[f]
	}

	input := HRAInput{
		HallRequests: hallReqs,
		States:       states,
	}

	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("assigner: marshal: %v", err)
	}

	out, err := exec.Command(executable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("assigner: exec: %v\noutput: %s", err, string(out))
	}

	var rawOutput map[string][][2]bool
	if err := json.Unmarshal(out, &rawOutput); err != nil {
		return nil, fmt.Errorf("assigner: unmarshal: %v", err)
	}

	result := make(map[string][config.N_Floors][2]bool)
	for id, floors := range rawOutput {
		var arr [config.N_Floors][2]bool
		for f := 0; f < config.N_Floors && f < len(floors); f++ {
			arr[f] = floors[f]
		}
		result[id] = arr
	}
	return result, nil
}
