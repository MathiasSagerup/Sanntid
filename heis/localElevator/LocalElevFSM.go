package localElevator

import (
	"heis/driver"
	"time"
)

const N_BUTTONS = 3
const N_FLOORS = 4
const doorOpenDuration = 3 * time.Second

type ElevatorBehaviour int

const (
	idle     = 0
	moving   = 1
	doorOpen = 2
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

type ElevState struct {
	floor                 int
	dirn                  driver.MotorDirection
	cabRequests           [N_FLOORS]bool
	behaviour             ElevatorBehaviour
	obstruction           bool
	ableToServiceRequests bool
}

type HallCallRequest struct {
	receiveHallCalls chan<- [N_FLOORS][2]bool
}

type localElevator struct {

	//input channels from driver:
	floorSensorChan chan int
	obstructionChan chan bool
	buttonChan      chan driver.ButtonEvent
	doorTimeoutChan chan bool

	//internal states
	floor                 int
	dirn                  driver.MotorDirection
	requests              [N_FLOORS][N_BUTTONS]bool
	behaviour             ElevatorBehaviour
	obstruction           bool
	ableToServiceRequests bool
	dirnBehaviourPair     dirnBehaviourPair
	hallCalls             [N_FLOORS][2]bool
	cabRequests           [N_FLOORS]bool

	//Internal request channels (one per public method)
	elevStateToWorldview chan ElevState
}

// å kjøre denne vil få heisen til å kjøre ned til en etasje, da vil localElevFSM
// overta styring.

// -------------param--*package --- instans av package ---
func NewLocalElev(floorSensorChan chan int,
	obstructionChan chan bool,
	stopBtnChan chan bool,
	buttonChan chan driver.ButtonEvent,
	elevStateToWorldview chan ElevState,
	assignerToLocalElev chan [N_FLOORS][2]bool) *localElevator {

	l := &localElevator{
		doorTimeoutChan:      make(chan bool, 1),
		floorSensorChan:      floorSensorChan,
		obstructionChan:      obstructionChan,
		buttonChan:           buttonChan,
		elevStateToWorldview: elevStateToWorldview,
	}

	//initialiser heis, kjør ned til nærmeste etasje
	if driver.GetFloor() == -1 {
		driver.SetMotorDirection(driver.MD_Down)
		l.dirn = driver.MD_Down
		l.behaviour = moving
		<-l.floorSensorChan
	}

	//sett til idle etter nådd nærmeste etasje
	driver.SetMotorDirection(driver.MD_Stop)
	l.dirn = driver.MD_Stop
	l.behaviour = idle
	l.floor = driver.GetFloor()
	l.setAllLights()
	l.sendElevState()

	//opprett oppdateringsloop
	go l.run(floorSensorChan,
		obstructionChan,
		stopBtnChan,
		buttonChan,
		assignerToLocalElev, l.doorTimeoutChan)
	return l
}

func (l *localElevator) run(floorSensorChan chan int,
	obstructionChan chan bool,
	stopBtnChan chan bool,
	buttonChan chan driver.ButtonEvent,
	assignerToLocalElev chan [N_FLOORS][2]bool,
	doorTimeOut chan bool) {

	for {
		select {

		case newFloor := <-floorSensorChan:
			l.floor = newFloor
			l.fsmOnFloorArrival(l.floor)
			l.sendElevState()

		case newBtn := <-buttonChan:
			if newBtn.Button == driver.BT_Cab {
				l.requests[newBtn.Floor][newBtn.Button] = true
				l.fsmOnRequestButtonPress(newBtn.Floor, newBtn.Button)
				l.sendElevState()
			}

		case obstr := <-obstructionChan:
			l.obstruction = obstr

			if l.obstruction == true {
				l.ableToServiceRequests = false
			} else {
				l.ableToServiceRequests = true
			}

		case newHallCalls := <-assignerToLocalElev:
			l.combineHallCallsAndCabCalls(newHallCalls)
			//dersom heis er idle, og mottar newHallCalls, skjer ingenting per nå
			//TODO: Bruk logikk fra fsmOnRequestButtonPress der heis er idle

		case <-l.doorTimeoutChan:
			if !l.obstruction {
				l.fsmOnDoorTimeout()
			} else {
				l.startDoorTimer()
			}

		}
	}
}

func (l *localElevator) sendElevState() {
	state := ElevState{
		floor:                 l.floor,
		dirn:                  l.dirn,
		cabRequests:           l.cabRequests,
		behaviour:             l.behaviour,
		obstruction:           l.obstruction,
		ableToServiceRequests: l.ableToServiceRequests,
	}
	select {
	case l.elevStateToWorldview <- state:
		// Sent successfully
	default:
		// No receiver ready; skip to avoid blocking
	}
}

func (l *localElevator) setAllLights() {
	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			driver.SetButtonLamp(driver.ButtonType(btn), floor, l.requests[floor][btn])
		}
	}
}

func (l *localElevator) combineHallCallsAndCabCalls(newHallCalls [N_FLOORS][2]bool) {

	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS-1; btn++ {
			l.requests[floor][btn] = newHallCalls[floor][btn]
		}
	}
}

// logikk for tilstandsendring i etasje
func (l *localElevator) fsmOnFloorArrival(newFloor int) {
	l.floor = newFloor
	driver.SetFloorIndicator(l.floor)

	switch l.behaviour {
	case moving:
		if requestsShouldStop(*l) {
			driver.SetMotorDirection(driver.MD_Stop)
			driver.SetDoorOpenLamp(true)
			requestsClearAtCurrentFloor(l)
			l.startDoorTimer()
			l.setAllLights()
			l.behaviour = doorOpen
		}
	}
}

// logikk for tilstandsendring
func (l *localElevator) fsmOnRequestButtonPress(btnFloor int, btnType driver.ButtonType) {
	switch l.behaviour {
	case doorOpen:
		if requestsShouldClearImmediately(*l, btnFloor, btnType) {
			l.startDoorTimer()
		} else {
			l.requests[btnFloor][btnType] = true
		}

	case moving:
		l.requests[btnFloor][btnType] = true

	case idle:
		l.requests[btnFloor][btnType] = true
		pair := requestsChooseDirection(*l)
		l.dirn = pair.dirn
		l.behaviour = pair.behaviour

		switch pair.behaviour {
		case doorOpen:
			driver.SetDoorOpenLamp(true)
			l.startDoorTimer()
			requestsClearAtCurrentFloor(l)
		case moving:
			driver.SetMotorDirection(l.dirn)
		case idle:
			// nothing to do
		}
	}
	l.setAllLights()
}

func (l *localElevator) fsmOnDoorTimeout() {
	switch l.behaviour {
	case doorOpen:
		pair := requestsChooseDirection(*l)
		l.dirn = pair.dirn
		l.behaviour = pair.behaviour

		switch l.behaviour {
		case doorOpen:
			l.startDoorTimer()
			requestsClearAtCurrentFloor(l)
			l.setAllLights()
		case moving, idle:
			driver.SetDoorOpenLamp(false)
			driver.SetMotorDirection(l.dirn)
		}
	}
}

func (l *localElevator) startDoorTimer() {
	time.AfterFunc(doorOpenDuration, func() {
		l.doorTimeoutChan <- true
	})
}
