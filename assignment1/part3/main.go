// Use `go run foo.go` to run your program

package main

import (
    "fmt"
    "runtime"
    "time"
	"sync"
)

var i = 0

func incrementing(i*int) {
    for j := 0; j<1000000; j++ {
		*i++
		//
		*i++
	}
}

func decrementing(i *int) {
    for j := 0; j<1000000; j++ {
		*i--
	}	
}

func server (inc <- chan int, dex <- chan int, done chan <- int) { //lytter på increase, decrease og done
	var i int = 0
	for {
		select {
		case <-inc:
			increase(&i)
		case <-dec
			decrease(&i)
		case done: //hvis done får en verdi, avslutt
			return
		}



func main() {
    // What does GOMAXPROCS do? What happens if you set it to 1?
    runtime.GOMAXPROCS(2)
	
	

	var i int = 0
	go incrementing(&i)
	go decrementing(&i)
    time.Sleep(500*time.Millisecond)
    fmt.Println("The magic number is:", i)
}
