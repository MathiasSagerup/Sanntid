package main

import (
	"fmt"
	"time"
	"localElevator"
)


func main() {

	buttonCh := make(chan Driver.ButtonEvent)
	floorCh := make(chan int)
	stopCh := make(chan bool)
	obstructionCh := make(chan bool)
	assignedHallCallsCh := make(chan [N_FLOORS][N_BUTTONS]bool, 1)
	localStateCh        := make(chan Elevator, 1)


	driver.Init(localElevAddr, N_FLOORS)

	//følgende funksjoner må være egne goroutines, hvis ikke vil de blokkere programmet
	go Driver.PollButtons(buttonCh) 
	go Driver.PollFloorSensor(floorCh)
	go Driver.PollStopButton(stopCh)
	go Driver.PollObstructionSwitch(obstructionCh)

	go localElevatorFSM()
	go hallCallsAssigner(kane)
	go communication()

	fmt.Println("Hello, world!")

	// EKSEMPEL KODE FRA STUD.ASS.:
	timer := time.NewTimer(100 * time.Millisecond)

	for {

		//hallCallsAssigner
		//så localElevFSM

		select {
		case message <- rx_channel:
			// motta melding og oppdater worlview

		case elevator_event <- elevio_channel:
			// oppdater heis posisjon

		case <-timer.C:
			timer.Reset(100 * time.Millisecond)
			worldview -> tx_channel
		}
	}
}
