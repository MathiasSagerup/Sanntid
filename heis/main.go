package main

import (
	"fmt"
	"heis/communication"
	"heis/config"
	"heis/driver"
	"heis/hallCallsAssigner"
	"heis/localElevator"
	"heis/network/localip"
	"os"
)

func main() {

	//uses the local ip address + an id given on the command line to
	//create localID
	var id string
	if id == "" {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	serverAddr := "localhost:15657"

	fmt.Printf("Starting elevator %s connecting to %s\n", id, serverAddr)

	driver.Init(serverAddr, config.N_FLOORS)

	//sensor channels
	floorSensorChan := make(chan int)
	obstructionChan := make(chan bool)
	stopBtnChan := make(chan bool)
	buttonChan := make(chan driver.ButtonEvent)

	//localElevator til Worlview
	elevStateToWorldview := make(chan localElevator.ElevState)

	//HallCallAssigner til localELevator
	assignerToLocalElev := make(chan [config.N_FLOORS][2]bool)

	//input kanal til hallCallsAssigner:
	worldviewToHallCallAssigner := make(chan hallCallsAssigner.HRAInput)

	//localElevator til communication:
	localElevToCommuncation := make(chan<- localElevator.ElevState)

	//

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

	c := communication.NewCommunicationModule(id, config.BroadcastPort, localElevToCommuncation)

	//input: InputChan chan HRAInput, OutputChan chan [config.N_Floors][2]bool, ID string
	h := hallCallsAssigner.NewHallCallAssigner(worldviewToHallCallAssigner, assignerToLocalElev, id)

	select {}
}
