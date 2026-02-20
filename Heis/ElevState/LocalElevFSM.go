package ElevState

import (
	"Driver"
	"fmt"
	"heis/Driver"
)

//elevator struct, hvilken informasjon trenger ElevFSM?

type DirnBehaviourPair struct {
    dirn MotorDirection;
    behaviour ElevatorBehaviour;
}

type Order int //Go har ikke Enums som en del av språket, men dette
//emulerer / har samme funksjon

const (
	
	No_Order = 0
	Unconfirmed_Order = 1
	Confirmed_Order = 2
	Cab_Call = 3

)

type Elevator struct {

	floor int;
	dirn MotorDirection;
	requests [N_FLOORS][N_BUTTONS] Order//Oversikt over alle ordre
	// og om alle heiser vet om dem
	behaviour ElevatorBehaviour
	
}

func confirmedOrdersAtFloor(floor, localElev) bool { //Sjekker om det er 
	// en bekreftet ordre på etasjen

	for b:=0 ; b <= N_BUTTONS; b++ {
		if localElev.requests[floor][b] == Confirmed_Order {
			return true
		}
	} 

	return false

}



//kanaler opprettet fra før
//bestillinger lagt inn tidligere av HallCallsAssigner
func elevFSM() Elevator {

	var localElev Elevator
	Driver.Init(localElevAddr,numFloors);

	//sjekk input på kanaler, hvis noe skjer, gjør dette...
	//Hva er det egentlig som vil komme via kanaler og hva kommer
	buttonCh := make(chan Driver.ButtonEvent)
	floorCh := make(chan int)
	stopCh := make(chan bool)
	obstructionCh := make(chan bool)

	for {

		//kommunikasjon om egen tilstand sendes før oppdatering av 
		//egen tilstand ikke sant? Implementere her eller annet sted?

		Driver.PollButtons(buttonCh) //potensielt ha som goroutines?
		Driver.PollFloorSensor(floorCh)
		Driver.PollStobBUtton(stopCh)
		Driver.POllObstructionSwitch(obstructionCh)
		
		select {

		//buttonPushed
		case btn:= <- buttonCh:
			//finn ut hvilken knapp. trykket før? gjør ingenting. 
			//ikke trykket, kommuniser med resten av systemet 
			
			

		//newFloorReached
		case floor:= <- floorCh:
			localElev.floor = floor //uneødvendig?
			
			//ny etasje, hva må gjøres?
			if confirmedOrdersAtFloor(floor, localElev) {
				//åpne dører, timer, fjern bestillinger, gi beskjed 
				//til andre heiser. 
			}

		
		//stopBtn Pushed
		case stop:= <- stopCh:
			//stopKnapp, trykket hva nå?

			

		//obstruction button (husker ikke om denne er en del av kravene)
		case obstr:= <- obstructionCh:



		}
	}
}