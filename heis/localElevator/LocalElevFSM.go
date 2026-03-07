package localElevator

import (
	"heis/communication"
	"heis/driver"
	"time"
)

const N_BUTTONS = 3
const N_FLOORS = 4

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

type AssignedHallCalls struct {
	//not including cab calls
	hallCalls [N_FLOORS][N_BUTTONS - 1]bool
}

type localElevator struct {

	//dependencies
	driver        *driver                            // pointer to a Driver struct instance
	communication *communication                     //tilsvarende
	assigner      *hallcallassigner.HallCallAssigner //tilsvarende. Hvilken lages først?? Hierarki problem

	//input channels from driver:
	floorSensorChan chan int
	obstructionChan chan bool
	buttonChan      chan driver.ButtonEvent

	//internal states
	floor                 int
	dirn                  driver.MotorDirection
	requests              [N_FLOORS][N_BUTTONS]bool
	behaviour             ElevatorBehaviour
	obstruction           bool
	ableToServiceRequests bool
	dirnBehaviourPair     dirnBehaviourPair
	hallCalls             AssignedHallCalls

	//Internal request channels (one per public method)
	elevatorStateRequestChan chan elevatorStateRequest
	doorTimeoutChan          chan bool
}

type elevatorStateRequest struct {
	responseChan chan localElevator
}

// å kjøre denne vil få heisen til å kjøre ned til en etasje, da vil localElevFSM
// overta styring.

// -------------param--*package --- instans av package ---
func NewLocalElev(d *driver.Driver, c *communication.Communication, h *hallcallassigner.HallCallAssigner) *localElevator {

	l := &localElevator{
		driver:                   d,
		communication:            c,
		assigner:                 h,
		floorSensorChan:          make(chan int),
		obstructionChan:          make(chan bool),
		buttonChan:               make(chan driver.ButtonEvent),
		elevatorStateRequestChan: make(chan elevatorStateRequest),
		doorTimeoutChan:          make(chan bool, 1),
	}

	go d.PollFloorSensor(l.floorSensorChan)
	go d.PollObstructionSwitch(l.obstructionChan)
	go d.PollButtons(l.buttonChan)

	if d.GetFloor() == -1 {
		d.SetMotorDirection(driver.MD_Down)
		l.dirn = driver.MD_Down
		l.behaviour = moving
		<-l.floorSensorChan
	}

	d.SetMotorDirection(driver.MD_Stop)
	l.dirn = driver.MD_Stop
	l.behaviour = idle
	l.floor = d.GetFloor()
	l.setAllLights()

	go l.run()
	return l
}

func (l *localElevator) run() {
	for {
		select {
		case req := <-l.elevatorStateRequestChan:
			req.responseChan <- *l //derefererer så vi sender kopi

		case newFloor := <-l.floorSensorChan:
			l.floor = newFloor

		case newBtn := <-l.buttonChan:
			if newBtn.Button == driver.BT_Cab {
				l.requests[newBtn.Floor][newBtn.Button] = true
			}

		case obstr := <-l.obstructionChan:
			l.obstruction = obstr

		case newHallCalls := <-l.assigner.AssignedHallCallsChan:
			l.hallCalls = newHallCalls
			l.combineHallCallsAndCabCalls(newHallCalls)
		}
	}
}

// Lag en get funksjon, returner elevator struct minus obstruction og hallCalls
// TODO: ikke send over obstruction og hallCalls
func (l *localElevator) GetLocalElevator() localElevator {
	respChan := make(chan localElevator)
	l.elevatorStateRequestChan <- elevatorStateRequest{
		responseChan: respChan,
	}
	return <-respChan
}

func (l *localElevator) setAllLights() {
	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS; btn++ {
			l.driver.SetButtonLamp(driver.ButtonType(btn), floor, l.requests[floor][btn])
		}
	}
}

func (l *localElevator) combineHallCallsAndCabCalls(newHallCalls hallCalls) {

	for floor := 0; floor < N_FLOORS; floor++ {
		for btn := 0; btn < N_BUTTONS-1; btn++ {
			l.requests[floor][btn] = newHallCalls[floor][btn]
		}
	}
}

// Husk å tømme kanaler før de oppdateres.

//TODO: Read from driver, oppdatere tilstander, så funksjonskall til hallCallsAssigner
// så switch case logikk på tilstandsendringer heisoppførsel)

//TODO Endre logikken slik at det ikke er event basert.
//

// logikk for tilstandsendring i etasje
func (l *localElevator) fsmOnFloorArrival(newFloor int) {
	l.floor = newFloor
	l.driver.SetFloorIndicator(l.floor)

	switch l.behaviour {
	case moving:
		if requestsShouldStop(*l) {
			l.driver.SetMotorDirection(driver.MD_Stop)
			l.driver.SetDoorOpenLamp(true)
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
			l.driver.SetDoorOpenLamp(true)
			l.startDoorTimer()
			requestsClearAtCurrentFloor(l)
		case moving:
			l.driver.SetMotorDirection(l.dirn)
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
			l.driver.SetDoorOpenLamp(false)
			l.driver.SetMotorDirection(l.dirn)
		}
	}
}

func (l *localElevator) startDoorTimer() {
	time.AfterFunc(doorOpenDuration, func() {
		l.doorTimeoutChan <- true
	})
}
