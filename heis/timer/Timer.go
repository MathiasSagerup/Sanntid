package timer

import (
	"time"
)

func WaitThreeSecondsAsync() <-chan bool {
	result := make(chan bool, 1)

	go func() {
		time.Sleep(3 * time.Second)
		result <- true
	}()

	return result
} //Asynkron, kan gjøre andre ting

func WaitThreeSeconds() bool {
	time.Sleep(3 * time.Second)
	return true
} //Sleep blokkerer, kan ikke gjøre andre ting samtidig.

func DoorTimer(timesUp chan<- struct{}, startTimer <-chan struct{}, done <-chan struct{}) {
	for {
		select {
		case _, ok := <-startTimer:
			if !ok {
				return
			}
			t := time.NewTimer(3 * time.Second)
			select {
			case <-t.C:
				timesUp <- struct{}{}
			case <-done:
				t.Stop()
				return
			}
		case <-done:
			return
		}
	}
}

func ResetDoorTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	t.Reset(3 * time.Second)
}
