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
	HallRequests [config.N_Floors][2]bool `json:"hallRequests"`
	States       map[string]HRAElevState  `json:"states"`
}

type HRAOutput struct {
	HallRequests [config.N_Floors][2]bool
}

type HallCallAssigner struct {

	//inptu channel from worldview
	HRAInputChan chan HRAInput

	//output channel to localElevator
	HRAOutputChan chan HRAOutput

	//internalStates
	input       HRAInput
	output      HRAOutput
	localElevID string
}

func newHallCallAssigner(InputChan chan HRAInput, OutputChan chan HRAOutput, ID string) *HallCallAssigner {

	h := &HallCallAssigner{
		HRAInputChan:  InputChan,
		HRAOutputChan: OutputChan,
		localElevID:   ID,
	}

	go h.run(InputChan, OutputChan, h.localElevID)
	return h
}

func (h *HallCallAssigner) run(InputChan chan HRAInput, OutputChan chan HRAOutput, ID string) {

	for {
		select {
		case input := <-InputChan:
			AssignedHallCalls, err := assign(input)
			if err != nil {
				fmt.Errorf("assigner: unmarshal: %v", err)
			}
			OutputChan <- AssignedHallCalls[ID]
		}
	}

}

// Assign calls the hall_request_assigner binary and returns assigned hall requests per elevator.
// Returns map[elevatorID] -> [floor][2]bool (up, down per floor)

func assign(Input HRAInput) (map[string][config.N_Floors][2]bool, error) {
	executable := config.HallCallAssignerExec
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}

	jsonBytes, err := json.Marshal(Input)
	if err != nil {
		return nil, fmt.Errorf("assigner: marshal: %v", err)
	}

	out, err := exec.Command(executable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("assigner: exec: %v\noutput: %s", err, string(out))
	}

	var assignedHallRequests map[string][config.N_Floors][2]bool
	if err := json.Unmarshal(out, &assignedHallRequests); err != nil {
		return nil, fmt.Errorf("assigner: unmarshal: %v", err)
	}

	return assignedHallRequests, err
}
