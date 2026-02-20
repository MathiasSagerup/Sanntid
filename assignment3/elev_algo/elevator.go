package elevator

import (
	"fmt"
	"yourproject/hardware"
)

func BehaviorToString(eb ElevatorBehaviour) string {
	switch eb {
	case EBIdle:
		return "EB_Idle"
	case EBDoorOpen:
		return "EB_DoorOpen"
	case EBMoving:
		return "EB_Moving"
	default:
		return "EB_UNDEFINED"
	}
}

func DirnToString(d Dirn) string {
	switch d {
	case DUp:
		return "D_Up"
	case DDown:
		return "D_Down"
	case DStop:
		return "D_Stop"
	default:
		return "D_UNDEFINED"
	}
}

func ButtonToString(b Button) string {
	switch b {
	case BHallUp:
		return "B_HallUp"
	case BHallDown:
		return "B_HallDown"
	case BCab:
		return "B_Cab"
	default:
		return "B_UNDEFINED"
	}
}

func Print(es Elevator) {
	fmt.Println("  +--------------------+")
	fmt.Printf(
		"  |floor = %-2d          |\n"+
			"  |dirn  = %-12.12s|\n"+
			"  |behav = %-12.12s|\n",
		es.Floor,
		DirnToString(es.Dirn),
		BehaviorToString(es.Behaviour),
	)
	fmt.Println("  +--------------------+")
	fmt.Println("  |  | up  | dn  | cab |")

	for f := NFloors - 1; f >= 0; f-- {
		fmt.Printf("  | %d", f)

		for btn := 0; btn < NButtons; btn++ {

			if (f == NFloors-1 && btn == int(BHallUp)) ||
				(f == 0 && btn == int(BHallDown)) {

				fmt.Print("|     ")

			} else {
				if es.Requests[f][btn] == 1 {
					fmt.Print("|  #  ")
				} else {
					fmt.Print("|  -  ")
				}
			}
		}
		fmt.Println("|")
	}

	fmt.Println("  +--------------------+")
}

func Uninitialized() Elevator {
	hardware.Init()

	return Elevator{
		Floor:     -1,
		Dirn:      DStop,
		Behaviour: EBIdle,
		Config: Config{
			DoorOpenDurationS: 3.0,
		},
	}
}

func FloorSensor() int {
	return hardware.GetFloorSensorSignal()
}

func RequestButton(f int, b Button) int {
	return hardware.GetButtonSignal(hardware.ButtonType(b), f)
}

func StopButton() int {
	return hardware.GetStopSignal()
}

func Obstruction() int {
	return hardware.GetObstructionSignal()
}

func FloorIndicator(f int) {
	hardware.SetFloorIndicator(f)
}

func RequestButtonLight(f int, b Button, v int) {
	hardware.SetButtonLamp(hardware.ButtonType(b), f, v)
}

func DoorLight(v int) {
	hardware.SetDoorOpenLamp(v)
}

func StopButtonLight(v int) {
	hardware.SetStopLamp(v)
}

func MotorDirection(d Dirn) {
	hardware.SetMotorDirection(hardware.MotorDirection(d))
}