package main

import (
	"heis/localElevator"
	"heis/driver"
	"heis/config"
	"fmt"
)


func main() {

	var id string
	serverAddr:="localhost:15657"

	fmt.Printf("Starting elevator %s connecting to %s\n", id, serverAddr)

	driver.Init(serverAddr, config.N_Floors)

	//sensor channels
	floorSensorChan:= make(chan int)
	obstructionChan:= make(chan bool)
	stopBtnChan:=make(chan bool)
	buttonChan:= make(chan driver.ButtonEvent)

	//localElevator kanaler
	elevStateToWorldview := make(chan localElevator.ElevState)
	assignerToLocalElev := make(chan localElevator.AssignedHallCalls)

	go driver.PollButtons(buttonChan)
	go driver.PollFloorSensor(floorSensorChan)
	go driver.PollObstructionSwitch(obstructionChan)
	go driver.PollStopButton(stopBtnChan)

	l:= localElevator.NewLocalElev(floorSensorChan,
		obstructionChan,
		stopBtnChan,
		buttonChan,
		elevStateToWorldview,
		assignerToLocalElev)

	l.Print()
	
	select{}
}
	