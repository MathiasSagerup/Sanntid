// Use `go run foo.go` to run your program

package main

import (
    "fmt"
	"runtime"
	"time"
)

func incrementing(inc chan<- struct{}, done_inc chan<- struct{}) {
    for j := 0; j<1000000; j++ {
		inc <- struct{}{}
	}
	done_inc <- struct{}{}
}

func decrementing(dec chan<- struct{}, done_dec chan<- struct{}) {
    for j := 0; j<1000000; j++ {
		dec <- struct{}{}
	}	
	done_dec <- struct{}{}
}

func server (inc <-chan struct{}, dec <-chan struct{}, send_result <-chan struct{}, result chan<- int){ //lytter på increase, decrease og done
	var i int = 0
	for {
		select {
		case <-inc:
			i--
		case <-dec:
			i++
		case <-send_result: //hvis done får en verdi, avslutt
			result <- i
		}
	}
}



func main() {
    // What does GOMAXPROCS do? What happens if you set it to 1?
    runtime.GOMAXPROCS(3)
	do_inc := make(chan struct{})
	do_dec := make(chan struct{})
	done_inc := make(chan struct{})
	done_dec := make(chan struct{})
	send_result := make(chan struct{})
	result := make(chan int)
	go server(do_inc, do_dec, send_result, result)

	go incrementing(do_inc, done_inc)
	go decrementing(do_dec, done_dec)

	<-done_inc
	<-done_dec

	send_result <- struct{}{}

	
    time.Sleep(500*time.Millisecond)
    fmt.Println("The magic number is:", <-result)
}
