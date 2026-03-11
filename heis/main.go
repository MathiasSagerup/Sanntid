package main

import (
	"fmt"
	"heis/config"
	"heis/driver"
	"heis/hallCallsAssigner"
	"heis/localElevator"
	"heis/network/localip"
)

func main() {

	LocalID, err := localip.LocalIP()

	if err != nil {
		fmt.Println("Error when initalizing localID")
	}

	serverAddr := "localhost:15657"

	fmt.Printf("Starting elevator %s connecting to %s\n", LocalID, serverAddr)

	driver.Init(serverAddr, config.N_FLOORS)

	//sensor channels
	floorSensorChan := make(chan int)
	obstructionChan := make(chan bool)
	stopBtnChan := make(chan bool)
	buttonChan := make(chan driver.ButtonEvent)

	//localElevator kanaler
	elevStateToWorldview := make(chan localElevator.ElevState)
	assignerToLocalElev := make(chan [config.N_FLOORS][2]bool)

	//input kanal til hallCallsAssigner:
	worldviewToHallCallAssigner := make(chan hallCallsAssigner.HRAInput)

	go driver.PollButtons(buttonChan)
	go driver.PollFloorSensor(floorSensorChan)
	go driver.PollObstructionSwitch(obstructionChan)
	go driver.PollStopButton(stopBtnChan)

	l := localElevator.NewLocalElev(floorSensorChan,
		obstructionChan,
		stopBtnChan,
		buttonChan,
		elevStateToWorldview,
		assignerToLocalElev)

	l.Print()

	//input: InputChan chan HRAInput, OutputChan chan [config.N_Floors][2]bool, ID string
	h := hallCallsAssigner.NewHallCallAssigner(worldviewToHallCallAssigner, assignerToLocalElev, LocalID)

	select {}
}
