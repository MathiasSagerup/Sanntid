package localElevator

import (
	"fmt"
	"heis/config"
	"heis/driver"
	"time"
)

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

type ElevState struct {
	Floor                 int
	Dirn                  driver.MotorDirection
	Behaviour             ElevatorBehaviour
	CabRequests           [config.N_FLOORS]bool
	Obstruction           bool
	AbleToServiceRequests bool
}

type localElevator struct {

	//input channels from driver:
	floorSensorCh     	chan int
	obstructionCh     	chan bool
	buttonCh          	chan driver.ButtonEvent
	doorTimeoutCh     	chan bool
	motorLossTimeoutCh	chan bool

	//internal states
	floor                 int
	dirn                  driver.MotorDirection
	requests              [config.N_FLOORS][config.N_BUTTONS]bool
	behaviour             ElevatorBehaviour
	obstruction           bool
	ableToServiceRequests bool
	dirnBehaviourPair     dirnBehaviourPair
	assignedHallCalls     [config.N_FLOORS][config.N_TRAVEL_DIRN]bool
	completedHallCalls    [config.N_FLOORS][config.N_TRAVEL_DIRN]bool

	//Input/output channels
	localElevStateCh chan ElevState
	completedHallCallsCh chan [config.N_FLOORS][config.N_TRAVEL_DIRN]bool
	assignedHallCallsCh chan [config.N_FLOORS][config.N_TRAVEL_DIRN]bool
}

func NewLocalElev(
	floorSensorCh chan int,
	obstructionCh chan bool,
	buttonCh chan driver.ButtonEvent,
	localElevStateCh chan ElevState,
	assignedHallCallsCh chan [config.N_FLOORS][config.N_TRAVEL_DIRN]bool,
	completedHallCallsCh chan [config.N_FLOORS][config.N_TRAVEL_DIRN]bool,
	initialCabCalls [config.N_FLOORS]bool,
) *localElevator {

	l := &localElevator{
		doorTimeoutCh:     		make(chan bool, 1),
		motorLossTimeoutCh:	 	make(chan bool, 1),
		floorSensorCh:      	floorSensorCh,
		obstructionCh:      	obstructionCh,
		buttonCh:           	buttonCh,
		localElevStateCh: 		localElevStateCh,
		assignedHallCallsCh:  	assignedHallCallsCh,
		completedHallCallsCh: 	completedHallCallsCh,
	}

	//Initialize elevator to defined state
	if driver.GetFloor() == -1 { 	// -1 indicates between floors
		driver.SetMotorDirection(driver.MD_Down)
		l.dirn = driver.MD_Down
		l.behaviour = moving
		<-l.floorSensorCh
	}

	driver.SetMotorDirection(driver.MD_Stop)
	l.dirn = driver.MD_Stop
	l.behaviour = idle
	l.floor = driver.GetFloor()

	l.applyInitialRequests(initialCabCalls)
	l.setCabLights()
	driver.SetFloorIndicator(l.floor)

	l.ableToServiceRequests = true

	l.sendlocalElevStateCh()

	go l.run()

	return l
}

func (l *localElevator) run() {

	for {
		select {

		case newFloor := <-l.floorSensorCh:
			l.floor = newFloor
			l.fsmOnFloorArrival(l.floor)
			l.sendlocalElevStateCh()

		case newBtn := <-l.buttonCh:
			if newBtn.Button == driver.BT_Cab {
				l.requests[newBtn.Floor][newBtn.Button] = true
				l.fsmOnRequestButtonPress(newBtn.Floor, newBtn.Button)
			}
			l.sendlocalElevStateCh()

		case newObstr := <-l.obstructionCh:
			l.obstruction = newObstr

			if l.obstruction == true {
				l.ableToServiceRequests = false
			} else {
				l.ableToServiceRequests = true
			}
			l.sendlocalElevStateCh()

		case newHallCalls := <-l.assignedHallCallsCh:
			l.fsmOnReceivedHallCalls(newHallCalls)
			l.sendlocalElevStateCh()

		case <-l.doorTimeoutCh:
			if !l.obstruction {
				l.fsmOnDoorTimeout()

			} else {
				l.startDoorTimer()

			}
			l.sendlocalElevStateCh()

		case <-l.motorLossTimeoutCh:

			if l.behaviour == moving {
				l.ableToServiceRequests = false
				l.sendlocalElevStateCh()
			}
		}
	}
}

func (l *localElevator) getCabCalls() [config.N_FLOORS]bool {
	cabCalls := [config.N_FLOORS]bool{}

	for floor := 0; floor < config.N_FLOORS; floor++ {
		cabCalls[floor] = l.requests[floor][driver.BT_Cab]
	}
	return cabCalls
}

func (l *localElevator) getElevState() ElevState {
	state := ElevState{
		Floor:                 l.floor,
		Dirn:                  l.dirn,
		CabRequests:           l.getCabCalls(),
		Behaviour:             l.behaviour,
		Obstruction:           l.obstruction,
		AbleToServiceRequests: l.ableToServiceRequests,
	}
	return state
}

func (l *localElevator) applyInitialRequests(initialCabCalls [config.N_FLOORS]bool) {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		l.requests[floor][driver.BT_Cab] = initialCabCalls[floor]
	}
}

func (l *localElevator) sendlocalElevStateCh() {
	select {
	case l.localElevStateCh <- l.getElevState():
	default:
		<-l.localElevStateCh
		l.localElevStateCh <- l.getElevState()
	}
}

func (l *localElevator) setCabLights() {
	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS; btn++ {
			if btn == driver.BT_Cab {
				driver.SetButtonLamp(driver.BT_Cab, floor, l.requests[floor][btn])
			}
		}
	}
}

func (l *localElevator) combineHallCallsAndCabCalls(newHallCalls [config.N_FLOORS][config.N_TRAVEL_DIRN]bool) {

	for floor := 0; floor < config.N_FLOORS; floor++ {
		for btn := 0; btn < config.N_BUTTONS-1; btn++ {
			l.requests[floor][btn] = newHallCalls[floor][btn]
		}
	}
}

func (l *localElevator) fsmOnReceivedHallCalls(newHallCalls [config.N_FLOORS][config.N_TRAVEL_DIRN]bool) {
	l.combineHallCallsAndCabCalls(newHallCalls)

	switch l.behaviour {

	case doorOpen:
		if requestsShouldClearImmediately(*l, l.floor, driver.BT_HallUp) {
			l.startDoorTimer()
			l.completedHallCalls[l.floor][driver.BT_HallUp] = true
			l.sendCompletedHallCallsToWorldView()

		} else if requestsShouldClearImmediately(*l, l.floor, driver.BT_HallDown) {
			l.completedHallCalls[l.floor][driver.BT_HallDown] = true
			l.sendCompletedHallCallsToWorldView()
			l.startDoorTimer()
		} 

	case moving:
		//do nothing, fsmOnFloorArrival handles state transition

	case idle:
		pair := requestsChooseDirection(*l)
		l.dirn = pair.dirn
		l.behaviour = pair.behaviour

		switch pair.behaviour {
		case doorOpen:
			driver.SetDoorOpenLamp(true)
			l.startDoorTimer()
			requestsClearAtCurrentFloor(l)
		case moving:
			l.startMotorLossTimer()
			driver.SetMotorDirection(l.dirn)
		case idle:
			// nothing to do
		}

	}
}

func (l *localElevator) sendCompletedHallCallsToWorldView() {
	select {
	case l.completedHallCallsCh <- l.completedHallCalls:
	default:
		fmt.Println("[localElevator] Warning: completedHallCalls channel is full, skipping update")
	}
	l.completedHallCalls = [config.N_FLOORS][config.N_TRAVEL_DIRN]bool{}
}

func (l *localElevator) fsmOnFloorArrival(newFloor int) {
	l.floor = newFloor
	driver.SetFloorIndicator(l.floor)
	l.startMotorLossTimer()
	l.ableToServiceRequests = true

	switch l.behaviour {
	case moving:
		if requestsShouldStop(*l) {
			driver.SetMotorDirection(driver.MD_Stop)
			driver.SetDoorOpenLamp(true)
			requestsClearAtCurrentFloor(l)
			l.startDoorTimer()
			l.setCabLights()
			l.behaviour = doorOpen
		}
	}
}

func (l *localElevator) fsmOnRequestButtonPress(btnFloor int, btnType driver.ButtonType) {

	switch l.behaviour {
	case doorOpen:
		if requestsShouldClearImmediately(*l, btnFloor, btnType) {
			l.completedHallCalls[btnFloor][btnType] = true
			l.sendCompletedHallCallsToWorldView()
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
			l.startMotorLossTimer()
			driver.SetMotorDirection(l.dirn)
		case idle:
			// nothing to do
		}
	}
	l.setCabLights()
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
			l.setCabLights()
		case moving, idle:
			driver.SetDoorOpenLamp(false)
			driver.SetMotorDirection(l.dirn)
			l.startMotorLossTimer()

		}
	}
}

func (l *localElevator) startDoorTimer() {
	time.AfterFunc(config.DoorOpenDuration,func() {
		l.doorTimeoutCh <- true 
	})
}

func (l *localElevator) startMotorLossTimer() {
	time.AfterFunc(config.MotorLossDuration,func() {
		l.motorLossTimeoutCh	<- true
	})
}

func(eb ElevatorBehaviour) String() string {
	switch eb {
	case moving:
		return "moving"
	case idle:
		return "idle"
	case doorOpen:
		return "doorOpen"
	default:
		return "undefined"
	}
}