// Use `go run foo.go` to run your program

package main

import (
    "fmt"
    "runtime"
    "time"
)

var i = 0

func incrementing(i*int) {
    for j := 0; j<1000000; j++ {
		*i++
	}
}

func decrementing(i *int) {
    for j := 0; j<1000000; j++ {
		*i--
	}	
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
