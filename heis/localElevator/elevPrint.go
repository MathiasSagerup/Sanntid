package localElevator

import (
	"fmt"
	"heis/driver"
)

func (eb ElevatorBehaviour) String() string {
	switch eb {
	case idle:
		return "idle"
	case doorOpen:
		return "doorOpen"
	case moving:
		return "moving"
	default:
		return "UNDEFINED"
	}
}

func (e Elevator) Print() {
	fmt.Println("  +--------------------+")
	fmt.Printf(
		"  |floor = %-2d          |\n"+
			"  |dirn  = %-12s|\n"+
			"  |behav = %-12s|\n",
		e.floor,
		e.dirn,
		e.behaviour,
	)
	fmt.Println("  +--------------------+")
	fmt.Println("  |  | up  | dn  | cab |")

	for f := N_FLOORS - 1; f >= 0; f-- {
		fmt.Printf("  | %d", f)
		for btn := 0; btn < N_BUTTONS; btn++ {
			if (f == N_FLOORS-1 && btn == int(driver.BT_HallUp)) ||
				(f == 0 && btn == int(driver.BT_HallDown)) {
				fmt.Print("|     ")
			} else if e.requests[f][btn] {
				fmt.Print("|  #  ")
			} else {
				fmt.Print("|  -  ")
			}
		}
		fmt.Println("|")
	}
	fmt.Println("  +--------------------+")
}
