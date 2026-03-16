package localElevator

import (
	//"fmt"
	"heis/config"
	"heis/driver"
	"time"
	"fmt"
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

type ElevState struct {
	Floor                 int
	Dirn                  driver.MotorDirection
	Behaviour             ElevatorBehaviour
	CabRequests           [N_FLOORS]bool
	Obstruction           bool
	AbleToServiceRequests bool
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
	assignedHallCalls     [N_FLOORS][2]bool

	//Internal request channels (one per public method)
	elevStateToWorldview chan ElevState

	//Input channel from hallcallassigner
	assignerToLocalElev chan [N_FLOORS][2]bool
}

// å kjøre denne vil få heisen til å kjøre ned til en etasje, da vil localElevFSM
// overta styring.

// -------------param--*package --- instans av package ---
func NewLocalElev(floorSensorChan chan int,
	obstructionChan chan bool,
	buttonChan chan driver.ButtonEvent,
	elevStateToWorldview chan ElevState,
	assignerToLocalElev chan [N_FLOORS][2]bool,
	initialCabCalls [N_FLOORS]bool,
	)*localElevator {

	l := &localElevator{
		doorTimeoutChan:      make(chan bool, 1),
		floorSensorChan:      floorSensorChan,
		obstructionChan:      obstructionChan,
		buttonChan:           buttonChan,
		elevStateToWorldview: elevStateToWorldview,
		assignerToLocalElev:  assignerToLocalElev,
	}

	//initialiser heis, kjør ned til nærmeste etasje
	if driver.GetFloor() == -1 {
		driver.SetMotorDirection(driver.MD_Down)
		l.dirn = driver.MD_Down
		l.behaviour = moving
		<-l.floorSensorChan
	}

	//sett til idle etter nådd nærmeste etasje og hent definert tilstand
	driver.SetMotorDirection(driver.MD_Stop)
	l.dirn = driver.MD_Stop
	l.behaviour = idle
	l.floor = driver.GetFloor()
	l.ableToServiceRequests = true

	//Aktiver initial cab calls
	l.applyInitialRequests(initialCabCalls)
	fmt.Println("initial requests:",l.requests)
	l.setAllLights()

	//Send initialtilstand til world view
	l.sendElevStateToWorldView()

	//opprett oppdateringsloop
	go l.run()

	return l
}

func (l *localElevator) run() {

	printTicker := time.NewTicker(time.Millisecond * 1000)
	defer printTicker.Stop()
	for {
		select {

		case newFloor := <-l.floorSensorChan:
			l.floor = newFloor
			l.fsmOnFloorArrival(l.floor)
			l.sendElevStateToWorldView()
			

		case newBtn := <-l.buttonChan:
			if newBtn.Button == driver.BT_Cab {
				l.requests[newBtn.Floor][newBtn.Button] = true
				l.fsmOnRequestButtonPress(newBtn.Floor, newBtn.Button)
				
			}
			l.sendElevStateToWorldView()

			//HallButton handled by worldview
		case newObstr := <-l.obstructionChan:
			l.obstruction = newObstr

			if l.obstruction == true {
				l.ableToServiceRequests = false
			} else {
				l.ableToServiceRequests = true
			}
			l.sendElevStateToWorldView()

		case newHallCalls := <-l.assignerToLocalElev:
			l.fsmOnReceivedHallCalls(newHallCalls)
			l.sendElevStateToWorldView()

		case <-l.doorTimeoutChan:
			if !l.obstruction {
				l.fsmOnDoorTimeout()

			} else {
				l.startDoorTimer()

			}
			l.sendElevStateToWorldView()
	
		}
	}
}

func (l *localElevator) getCabCalls() [N_FLOORS]bool {
	cabCalls := [N_FLOORS]bool{}

	for floor:= 0; floor < N_FLOORS; floor++ {
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

func (l *localElevator) applyInitialRequests(initialCabCalls [N_FLOORS]bool) {
	for floor := 0; floor < N_FLOORS; floor++ {
		l.requests[floor][driver.BT_Cab] = initialCabCalls[floor]
	}
}


func (l *localElevator) sendElevStateToWorldView() {
	select {
	case l.elevStateToWorldview <- l.getElevState():
	default:
		<-l.elevStateToWorldview
		l.elevStateToWorldview <- l.getElevState()
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

func (l *localElevator) fsmOnReceivedHallCalls(newHallCalls [config.N_FLOORS][2]bool) {
	l.combineHallCallsAndCabCalls(newHallCalls)

	switch l.behaviour {

	case doorOpen:
		if requestsShouldClearImmediately(*l, l.floor, driver.BT_HallUp) {
			l.startDoorTimer()
		} else if requestsShouldClearImmediately(*l, l.floor, driver.BT_HallDown) {
			l.startDoorTimer()
		} else {
			//do nothing
		}

	case moving:
		//do nothing, handled by fsmOnFloorArrival

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
			driver.SetMotorDirection(l.dirn)
		case idle:
			// nothing to do
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
