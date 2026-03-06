package localElevator

import (
	"fmt"
	"heis/driver"
	"heis/timer"
	"time"
)

const N_BUTTONS = 3
const N_FLOORS = 4

type ElevatorBehaviour int

const (
	idle     = 0
	moving   = 1
	doorOpen = 2
	stopped  = 3
)

type dirnBehaviourPair struct {
	dirn      driver.MotorDirection
	behaviour ElevatorBehaviour
}

type Order int

const (
	No_Order          = 0
	Unconfirmed_Order = 1
	Confirmed_Order   = 2
)

type Elevator struct {
	floor                   int
	dirn                    driver.MotorDirection
	requests                [N_FLOORS][N_BUTTONS]bool
	behaviour               ElevatorBehaviour
	obstruction             bool
	ableToServiceRequests bool
	dirnBehaviourPair       dirnBehaviourPair
}

// å kjøre denne vil få heisen til å kjøre ned til en etasje, da vil localElevFSM
// overta styring.
func intializeLocalElev(localElev Elevator) Elevator {
	driver.SetMotorDirection(driver.MD_Down)
	localElev.dirn = driver.MD_Down
	localElev.behaviour = moving
	setAllLights(localElev)

	//opprett nødvendige kanaler til localELevFSM
	return localElev
}

func setAllLights(localElev Elevator) { //turn all request lights off
	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			driver.SetButtonLamp(driver.ButtonType(btn), floor, localElev.requests[floor][btn])
		}
	}
}

func combineHallCallsAndCabCalls(newHallCalls hallCalls, localElev Elevator) Elevator {

	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS-1; btn++ {
			localElev.requests[floor][btn] = newHallCalls[floor][btn]
		}
	}
	return localElev
}

// Husk å tømme kanaler før de oppdateres. 

//TODO: Read from driver, oppdatere tilstander, så funksjonskall til hallCallsAssigner
// så switch case logikk på tilstandsendringer heisoppførsel) 

//TODO Endre logikken slik at det ikke er event basert. 
//

func localElevFSM(buttonCh <-chan driver.ButtonEvent, floorCh <-chan int,
	stopCh <-chan bool, obstructionCh <-chan bool,
	assignedHallCallsCh <-chan [N_FLOORS][N_BUTTONS]bool, localStateCh chan<- Elevator) {

	doorTimer := time.NewTimer(0)
	<-doorTimer.C //drain initial tick

	localElev := intializeLocalElev(Elevator{})

	for {
		sele
			case btn


		}
	}


	for {

		select {

		case btn := <-buttonCh:

			if btn.Button == driver.BT_Cab {

				localElev.requests[btn.Floor][btn.Button] = true

				switch localElev.behaviour {

				case doorOpen:
					if requests_shouldClearImmediately(localElev, btn.Floor, btn.Button) {
						timer.ResetDoorTimer(doorTimer)
					}

				case moving:

				case idle:
					localElev.dirnBehaviourPair = requests_chooseDirection(localElev)
					localElev.dirn = localElev.dirnBehaviourPair.dirn
					localElev.behaviour = localElev.dirnBehaviourPair.behaviour

					switch localElev.dirnBehaviourPair.behaviour {
					case doorOpen:
						driver.SetDoorOpenLamp(true)
						timer.ResetDoorTimer(doorTimer)
						localElev = requests_clearAtCurrentFloor(localElev)

					case moving:
						driver.SetMotorDirection(localElev.dirn)

					case idle:
					}
				}

			}
			localStateCh <- localElev

		case assignedHallCalls:= <-hallCallAssigner.assignedHallCallsCh: //implementer fsm_onReqBtnPress for hallcalls

			
			localElev = combineHallCallsAndCabCalls(hallCalls, localElev)

			switch localElev.behaviour {

			case doorOpen:
				if requests_shouldClearImmediately(localElev, localElev.floor, driver.BT_HallDown) ||
					requests_shouldClearImmediately(localElev, localElev.floor, driver.BT_HallUp) {
					timer.ResetDoorTimer(doorTimer)
				}

			case moving:

			case idle:
				localElev.dirnBehaviourPair = requests_chooseDirection(localElev)
				localElev.dirn = localElev.dirnBehaviourPair.dirn
				localElev.behaviour = localElev.dirnBehaviourPair.behaviour

				switch localElev.dirnBehaviourPair.behaviour {
				case doorOpen:
					driver.SetDoorOpenLamp(true)
					timer.ResetDoorTimer(doorTimer)
					localElev = requests_clearAtCurrentFloor(localElev)

				case moving:
					driver.SetMotorDirection(localElev.dirn)

				case idle:
				}
			}
			localStateCh <- localElev

		//newFloorReached
		case floor := <-floorCh: //implementert fsm_onFloorArrival, all good

			localElev.floor = floor
			driver.SetFloorIndicator(floor)

			switch localElev.behaviour {

			case moving:
				if requests_shouldStop(localElev) {

					driver.SetMotorDirection(driver.MD_Stop)

					//door behaviour
					driver.SetDoorOpenLamp(true)
					localElev.behaviour = doorOpen
					localElev = requests_clearAtCurrentFloor(localElev)
					timer.ResetDoorTimer(doorTimer)
				}
			}
			localStateCh <- localElev

		/*
			case stop := <-stopCh:

				if stop { //stoppknapp aktiv
					driver.SetStopLamp(stop)
					driver.SetMotorDirection(driver.MD_Stop)
					localElev.behaviour = idle
					localElev.dirn = driver.MD_Stop
				}

				if stop == false && localElev.behaviour == idle {
					driver.SetStopLamp(stop)
					localElev.behaviour = idle //klar til å ta imot ordre igjen
				}
		*/

		case obstr := <-obstructionCh: //sender når det er endring i obstr knapp

			localElev.obstruction = obstr

			if localElev.behaviour == doorOpen {
				timer.ResetDoorTimer(doorTimer)
			}
			localStateCh <- localElev

		case <-doorTimer.C:
			switch localElev.behaviour {

			case doorOpen:
				localElev.dirnBehaviourPair = requests_chooseDirection(localElev)
				localElev.dirn = localElev.dirnBehaviourPair.dirn
				localElev.behaviour = localElev.dirnBehaviourPair.behaviour

				switch localElev.behaviour {
				case doorOpen:
					driver.SetDoorOpenLamp(true)
					timer.ResetDoorTimer(doorTimer)
					localElev = requests_clearAtCurrentFloor(localElev)
					setAllLights(localElev)

				case moving:

				case idle:
					driver.SetDoorOpenLamp(false)
					driver.SetMotorDirection(localElev.dirn)
				}
			}

			localStateCh <- localElev

			fmt.Println("\n New state: \n")
		}
	}
}
