package hallCallsAssigner

import (
	"encoding/json"
	"fmt"
	"heis/config"
	"heis/driver"
	"heis/localElevator"
	"os/exec"
	"runtime"
	"strconv"
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

type HRAFormattedInput struct {
	HallRequests [config.N_FLOORS][2]bool `json:"hallRequests"`
	States       map[string]HRAElevState  `json:"states"`
}

type HRAInput struct {
	HallRequests    [config.N_FLOORS][2]bool
	ThisElevState   HRAElevStateInput
	OtherElevStates []HRAElevStateInput
}

type HRAElevStateInput struct {
	Behaviour   localElevator.ElevatorBehaviour
	Floor       int
	Dirn        driver.MotorDirection
	CabRequests [config.N_FLOORS]bool
}

type HallCallAssigner struct {
	//inptu channel from worldview
	HRAInputChan chan HRAInput

	//output channel to localElevator
	HRAOutputChan chan [config.N_FLOORS][2]bool

	//internalStates
	input       HRAInput
	output      [config.N_FLOORS][2]bool
	localElevID string
}

func NewHallCallAssigner(InputChan chan HRAInput, OutputChan chan [config.N_FLOORS][2]bool, ID string) *HallCallAssigner {

	h := &HallCallAssigner{
		HRAInputChan:  InputChan,
		HRAOutputChan: OutputChan,
		localElevID:   ID,
	}

	go h.run(InputChan, OutputChan, h.localElevID)
	return h
}

func (h *HallCallAssigner) run(InputChan chan HRAInput, OutputChan chan [config.N_FLOORS][2]bool, ID string) {
	for input := range InputChan { //siden det per nå er kun et case i en select case for loop bruker vi for range loop isteden.
		//fmt.Println(("received info from worldview"))
		AssignedHallCalls, _ := assign(input, ID)
		OutputChan <- AssignedHallCalls[ID]
	}
}

// Assign calls the hall_request_assigner binary and returns assigned hall requests per elevator.
// Returns map[elevatorID] -> [floor][2]bool (up, down per floor)
func assign(Input HRAInput, ID string) (map[string][config.N_FLOORS][2]bool, error) {
	executable := config.HallCallAssignerExec
	if runtime.GOOS == "windows" {
		executable += ".exe"
	}

	states := make(map[string]HRAElevState)

	localCabReqs := make([]bool, config.N_FLOORS)
	for f := 0; f < config.N_FLOORS; f++ {
		localCabReqs[f] = Input.ThisElevState.CabRequests[f]
	}
	states[ID] = HRAElevState{
		Behaviour:   Input.ThisElevState.Behaviour.String(),
		Floor:       Input.ThisElevState.Floor,
		Direction:   Input.ThisElevState.Dirn.String(),
		CabRequests: localCabReqs,
	}

	for i, elev := range Input.OtherElevStates {
		cabReqs := make([]bool, config.N_FLOORS)
		for f := 0; f < config.N_FLOORS; f++ {
			cabReqs[f] = elev.CabRequests[f]
		}
		states["peer-"+strconv.Itoa(i)] = HRAElevState{
			Behaviour:   elev.Behaviour.String(),
			Floor:       elev.Floor,
			Direction:   elev.Dirn.String(),
			CabRequests: cabReqs,
		}
	}

	formattedInput := HRAFormattedInput{
		HallRequests: Input.HallRequests,
		States:       states,
	}

	jsonBytes, err := json.Marshal(formattedInput)

	if err != nil {
		return nil, fmt.Errorf("assigner: marshal: %v", err)
	}
	//fmt.Println(Input)

	out, err := exec.Command(executable, "-i", string(jsonBytes)).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("assigner: exec: %v\noutput: %s", err, string(out))
	}

	var assignedHallRequests map[string][config.N_FLOORS][2]bool
	if err := json.Unmarshal(out, &assignedHallRequests); err != nil {
		return nil, fmt.Errorf("assigner: unmarshal: %v", err)
	}

	return assignedHallRequests, err
}