package ElevState

import (
	"fmt"
	"heis/Driver"
)

type ElevatorBehaviour int 

const (
	idle  = 0
	moving = 1
	doorOpen = 2 
	stopped = 3
)


type DirnBehaviourPair struct {
    dirn int;
    behaviour ElevatorBehaviour;
}

type Order int //Go har ikke Enums som en del av språket, men dette
//emulerer / har samme funksjon

const (
	
	No_Order = 0
	Unconfirmed_Order = 1
	Confirmed_Order = 2

)

//elevator struct, hvilken informasjon trenger ElevFSM?
type Elevator struct {

	floor int
	dirn Driver.MotorDirection 
	HallRequests [N_FLOORS][N_BUTTONS] Order//Oversikt over alle ordre
	// og om alle heiser vet om dem
	CabOrders [N_FLOORS] bool
	behaviour ElevatorBehaviour
	obstructed bool
	
}

func confirmedOrdersAtFloor(floor int, localElev Elevator) bool { //Sjekker om det er 
	// en bekreftet ordre på etasjen

	for b:=0 ; b <= N_BUTTONS; b++ {
		if localElev.HallRequests[floor][b] == Confirmed_Order {
			return true
		}
	} 

	return false

}

func clearConfirmedOrder(floor int, localElev Elevator) {

	for b:=0 ; b < N_BUTTONS; b++ {
		if localElev.HallRequests[floor][b] == Confirmed_Order {
			localElev.HallRequests[floor][b] = No_Order
		}
	}

	if localElev.CabOrders[floor] {
		localElev.CabOrders[floor] = false
	}

}

//kanaler opprettet fra før
//bestillinger lagt inn tidligere av HallCallsAssigner
func elevFSM() Elevator {

	var localElev Elevator
	Driver.Init(localElevAddr,N_FLOORS);

	//sjekk input på kanaler, hvis noe skjer, gjør dette...
	//Hva er det egentlig som vil komme via kanaler og hva kommer
	buttonCh := make(chan Driver.ButtonEvent)
	floorCh := make(chan int)
	stopCh := make(chan bool)
	obstructionCh := make(chan bool)


	for {

		//kommunikasjon om egen tilstand sendes før oppdatering av 
		//egen tilstand ikke sant? Implementere her eller annet sted?

		go Driver.PollButtons(buttonCh) 
		go Driver.PollFloorSensor(floorCh)
		go Driver.PollStobBUtton(stopCh)
		go Driver.POllObstructionSwitch(obstructionCh)
		
		select {

		//buttonPushed
		case btn:= <- buttonCh:
			//får ikke signal fra pollbutton med mindre knappen var av og knappen ikke var på før. 
			
			if  btn.Button != Driver.BT_Cab { //hvis det er en hall call 
			localElev.HallRequests[btn.floor][btn.Button] = Unconfirmed_Order

			} else {
				localElev.CabOrders[btn.floor] = true
			}
			


		//newFloorReached
		case floor:= <- floorCh:

			localElev.floor = floor
			Driver.SetFloorIndicator(floor)

			if confirmedOrdersAtFloor(floor, localElev) {
				Driver.SetMotorDirection(Driver.MD_Stop)
				Driver.SetDoorOpenLamp(true)
				clearConfirmedOrder(floor,localElev)
				//åpne dører, timer, fjern bestillinger
			}
		
		//stopBtn Pushed
		case stop:= <- stopCh:

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
		case obstr:= <- obstructionCh:
			
			if localElev.behaviour == doorOpen && obstr == true { //Hvis dør er åpen og obstruksjonsknapp. 
				localElev.obstructed = true

			} else if localElev.behaviour != doorOpen && obstr == true {
				localElev.obstructed = false //Hvis døren ikke er åpen fra før skal ikke obstruksjonsknapp gjøre noe

			} else {
				localElev.obstructed = false
				//obstruksjonsknapp av
			} 
			 
		}
	}
}