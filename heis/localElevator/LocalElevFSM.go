package localElevator

import (
	Driver "heis/driver"
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

type DirnBehaviourPair struct {
	dirn      int
	behaviour ElevatorBehaviour
}

type Order int

const (
	No_Order          = 0
	Unconfirmed_Order = 1
	Confirmed_Order   = 2
)

// elevator struct, hvilken informasjon trenger ElevFSM?
type Elevator struct {
	floor                   int
	dirn                    Driver.MotorDirection
	requests                [N_FLOORS][N_BUTTONS]bool
	behaviour               ElevatorBehaviour
	obstruction             bool
	ableToServiceHallOrders bool
	dirnBehaviourPair       DirnBehaviourPair
}

func chooseDirection() Driver.MotorDirection {

}

func shouldStop() bool {

}

func clearRequestsAtFloor() {

}

// bestillinger lagt inn tidligere av HallCallsAssigner
// hvor skal fsm_onInitBetweenFloors?
func elevFSM(localElevAddr string, N_FLOORS int) Elevator {

	var localElev Elevator
	localElevPtr := &Elevator{}
	Driver.Init(localElevAddr, N_FLOORS)

	buttonCh := make(chan Driver.ButtonEvent)
	floorCh := make(chan int)
	stopCh := make(chan bool)
	obstructionCh := make(chan bool)

	go Driver.PollButtons(buttonCh)
	go Driver.PollFloorSensor(floorCh)
	go Driver.PollStopButton(stopCh)
	go Driver.PollObstructionSwitch(obstructionCh)

	doorTimer := time.NewTimer(0)
	<-doorTimer.C //drain initial tick

	for {
		select {

		case btn := <-buttonCh:

			if btn.Button == Driver.BT_Cab {
				localElev.requests[btn.Floor][btn.Button] = true
			} //implementer fsm_onRequestButtonPress

		//newFloorReached
		case floor := <-floorCh: //implementert fsm_onFloorArrival, all good

			localElev.floor = floor
			Driver.SetFloorIndicator(floor)

			switch localElev.behaviour {

			case moving:
				if requests_shouldStop(localElev) {

					Driver.SetMotorDirection(Driver.MD_Stop)

					//door behaviour
					Driver.SetDoorOpenLamp(true)
					localElev.behaviour = doorOpen
					localElev = requests_clearAtCurrentFloor(localElev)
					timer.resetDoorTimer(doorTimer)
				}
			}

		//stopBtn Pushed
		case stop := <-stopCh:

			if stop { //stoppknapp aktiv
				Driver.SetStopLamp(stop)
				Driver.SetMotorDirection(Driver.MD_Stop)
				localElev.behaviour = stopped
				localElev.dirn = Driver.MD_Stop
			}

			if !stop && localElev.behaviour == stopped {
				Driver.SetStopLamp(stop)
				localElev.behaviour = idle //klar til å ta imot ordre igjen
			}

		//obstruction button
		case obstr := <-obstructionCh: //sender når det er endring i obstr knapp

			localElev.obstruction = obstr

			if localElev.behaviour == doorOpen {
				timer.resetDoorTimer(doorTimer)
			}

		case <-doorTimer.C: //implementer fsm_onDoorTimeout

			if localElev.behaviour == doorOpen && !localElev.obstruction {
				Driver.SetDoorOpenLamp(false)
				localElev.behaviour = idle
			}
		}
	}
}
