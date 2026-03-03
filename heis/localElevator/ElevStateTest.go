package localElevator

import (
	"fmt"
	"time"
)

func TestSingleElevator(e *Elevator) {

	fmt.Println("========== STARTING SINGLE ELEVATOR TEST ==========")

	// ----------------------------
	// TEST 1: Basic Upward Request
	// ----------------------------
	fmt.Println("TEST 1: Press cab button at top floor")

	e.requests[3][0] = true // simulate button press

	time.Sleep(500 * time.Millisecond)

	if e.behaviour != moving {
		fmt.Println("❌ FAIL: Elevator did not start moving")
	} else {
		fmt.Println("✅ PASS: Elevator started moving")
	}

	// Wait until it reaches floor 3
	timeout := time.After(10 * time.Second)

WAIT_LOOP:
	for {
		select {
		case <-timeout:
			fmt.Println("❌ FAIL: Timeout waiting for arrival")
			break WAIT_LOOP
		default:
			if e.floor == 3 && e.behaviour == doorOpen {
				fmt.Println("✅ PASS: Elevator opened door at correct floor")
				break WAIT_LOOP
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	// ----------------------------
	// TEST 2: Door closes after 3 seconds
	// ----------------------------
	fmt.Println("TEST 2: Door timing")

	time.Sleep(4 * time.Second)

	if e.behaviour == doorOpen {
		fmt.Println("❌ FAIL: Door did not close after 3 seconds")
	} else {
		fmt.Println("✅ PASS: Door closed correctly")
	}

	// ----------------------------
	// TEST 3: Obstruction
	// ----------------------------
	fmt.Println("TEST 3: Obstruction handling")

	e.requests[2][0] = true
	time.Sleep(500 * time.Millisecond)

	// wait until door open at floor 2
	for e.floor != 2 || e.behaviour != doorOpen {
		time.Sleep(100 * time.Millisecond)
	}

	e.obstruction = true
	fmt.Println("Obstruction activated")

	time.Sleep(4 * time.Second)

	if e.behaviour != doorOpen {
		fmt.Println("❌ FAIL: Door closed while obstructed")
	} else {
		fmt.Println("✅ PASS: Door remained open while obstructed")
	}

	e.obstruction = false
	fmt.Println("Obstruction cleared")

	time.Sleep(4 * time.Second)

	if e.behaviour == doorOpen {
		fmt.Println("❌ FAIL: Door did not close after obstruction cleared")
	} else {
		fmt.Println("✅ PASS: Door closed after obstruction cleared")
	}

	fmt.Println("========== TEST COMPLETE ==========")
}
