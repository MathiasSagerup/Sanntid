package Timer

import (
	"time"
)

func waitThreeSecondsAsync() <-chan bool {
	result := make(chan bool, 1)

	go func() {
		time.Sleep(3 * time.Second)
		result <- true
	}()

	return result
} //Asynkron, kan gjøre andre ting

func waitThreeSeconds() bool {
	time.Sleep(3 * time.Second)
	return true
} //Sleep blokkerer, kan ikke gjøre andre ting samtidig. 
