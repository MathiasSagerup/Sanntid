package main

import (
	"fmt"
	"time"
)

func main() {
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
