package main

import (
	"localElevator"
)

func main() {

	d := driver.NewDriver()
	n := network.NewNetwork()
	c := communication.NewCommunication()
	h := hallcallassigner.NewHallCallAssigner(d, c)
	l := localElevator.NewLocalElev(d, c, h)

}
