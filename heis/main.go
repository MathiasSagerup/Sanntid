package main

import (
	"fmt"
	"time"
	"localElevFSM"
)


func main() {

	buttonCh := make(chan Driver.ButtonEvent)
	floorCh := make(chan int)
	stopCh := make(chan bool)
	obstructionCh := make(chan bool)

	go Driver.PollButtons(buttonCh)
	go Driver.PollFloorSensor(floorCh)
	go Driver.PollStopButton(stopCh)
	go Driver.PollObstructionSwitch(obstructionCh)

	fmt.Println("Hello, world!")

	// EKSEMPEL KODE FRA STUD.ASS.:
	timer := time.NewTimer(100 * time.Millisecond)

	for {
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
