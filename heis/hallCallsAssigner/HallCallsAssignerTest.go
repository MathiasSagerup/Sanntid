package hallCallsAssigner

import (
	"fmt"
	"heis/config"
)

// TestSingleElevatorHRA calls Assign() with a single hardcoded elevator and
// prints which hall calls were assigned to it.  Run from main() to verify
// that the hall_request_assigner binary works correctly.
func TestSingleElevatorHRA() {
	fmt.Println("========== HRA SINGLE-ELEVATOR TEST ==========")

	const elevID = "test-elevator"

	// Elevator starts idle at floor 0, no cab requests
	states := map[string]HRAElevState{
		elevID: {
			Behaviour:   "idle",
			Floor:       0,
			Direction:   "stop",
			CabRequests: []bool{false, false, false, false},
		},
	}

	// Hall requests: up at floor 1, up+down at floor 2, down at floor 3
	var hallRequests [config.N_FLOORS][2]bool
	hallRequests[1][0] = true // floor 1 up
	hallRequests[2][0] = true // floor 2 up
	hallRequests[2][1] = true // floor 2 down
	hallRequests[3][1] = true // floor 3 down

	input := HRAInput{
		HallRequests: hallRequests,
		States:       states,
	}

	fmt.Printf("Input hall requests:\n")
	for f := 0; f < config.N_FLOORS; f++ {
		fmt.Printf("  floor %d: up=%v down=%v\n", f, hallRequests[f][0], hallRequests[f][1])
	}

	result, err := assign(input)
	if err != nil {
		fmt.Printf("❌ FAIL: Assign() returned error: %v\n", err)
		return
	}

	assigned, ok := result[elevID]
	if !ok {
		fmt.Printf("❌ FAIL: elevator %q not found in result\n", elevID)
		return
	}

	fmt.Printf("Assigned hall calls for %q:\n", elevID)
	allCorrect := true
	for f := 0; f < config.N_FLOORS; f++ {
		fmt.Printf("  floor %d: up=%v down=%v\n", f, assigned[f][0], assigned[f][1])
		if assigned[f] != hallRequests[f] {
			allCorrect = false
		}
	}

	if allCorrect {
		fmt.Println("✅ PASS: All hall calls assigned to the single elevator")
	} else {
		fmt.Println("❌ FAIL: Assigned calls do not match expected (all calls should go to one elevator)")
	}

	fmt.Println("========== HRA TEST COMPLETE ==========")
}
