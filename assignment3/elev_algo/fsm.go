package main

import (
	"fmt"
	"elevator"
	"requests"
	"timer"
)

func setAllLights(es elevator.Elevator) {
	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			elevator.RequestButtonLight(
				floor,
				elevator.Button(btn),
				es.requests[floor][btn] == 1,
			)
		}
	}
}

func FsmOnInitBetweenFloors(e *elevator.Elevator) {
	elevator.MotorDirection(elevator.DDown)
	e.Dirn = elevator.DDown
	e.Behaviour = elevator.EBMoving
}

func FsmOnRequestButtonPress(e *elevator.Elevator, btnFloor int, btnType elevator.Button) {

	fmt.Printf("\n\nFsmOnRequestButtonPress(%d, %s)\n",
		btnFloor,
		elevator.ButtonToString(btnType),
	)

	elevator.Print(*e)

	switch e.Behaviour {

	case elevator.EBDoorOpen:
		if requests.ShouldClearImmediately(*e, btnFloor, btnType) {
			timer.Start(e.Config.DoorOpenDurationS)
		} else {
			e.Requests[btnFloor][btnType] = 1
		}

	case elevator.EBMoving:
		e.Requests[btnFloor][btnType] = 1

	case elevator.EBIdle:
		e.Requests[btnFloor][btnType] = 1

		pair := requests.ChooseDirection(*e)
		e.Dirn = pair.Dirn
		e.Behaviour = pair.Behaviour

		switch pair.Behaviour {

		case elevator.EBDoorOpen:
			elevator.DoorLight(true)
			timer.Start(e.Config.DoorOpenDurationS)

			updated := requests.ClearAtCurrentFloor(*e)
			*e = updated

		case elevator.EBMoving:
			elevator.MotorDirection(e.Dirn)

		case elevator.EBIdle:
			// nothing
		}
	}

	setAllLights(*e)

	fmt.Println("\nNew state:")
	elevator.Print(*e)
}

func FsmOnFloorArrival(e *elevator.Elevator, newFloor int) {

	fmt.Printf("\n\nFsmOnFloorArrival(%d)\n", newFloor)
	elevator.Print(*e)

	e.Floor = newFloor
	elevator.FloorIndicator(e.Floor)

	switch e.Behaviour {

	case elevator.EBMoving:
		if requests.ShouldStop(*e) {

			elevator.MotorDirection(elevator.DStop)
			elevator.DoorLight(true)

			updated := requests.ClearAtCurrentFloor(*e)
			*e = updated

			timer.Start(e.Config.DoorOpenDurationS)
			setAllLights(*e)

			e.Behaviour = elevator.EBDoorOpen
		}
	}

	fmt.Println("\nNew state:")
	elevator.Print(*e)
}

func FsmOnDoorTimeout(e *elevator.Elevator) {

	fmt.Println("\n\nFsmOnDoorTimeout()")
	elevator.Print(*e)

	switch e.Behaviour {

	case elevator.EBDoorOpen:

		pair := requests.ChooseDirection(*e)
		e.Dirn = pair.Dirn
		e.Behaviour = pair.Behaviour

		switch e.Behaviour {

		case elevator.EBDoorOpen:
			timer.Start(e.Config.DoorOpenDurationS)

			updated := requests.ClearAtCurrentFloor(*e)
			*e = updated

			setAllLights(*e)

		case elevator.EBMoving, elevator.EBIdle:
			elevator.DoorLight(false)
			elevator.MotorDirection(e.Dirn)
		}
	}

	fmt.Println("\nNew state:")
	elevator.Print(*e)
}