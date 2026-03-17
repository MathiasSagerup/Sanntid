package main

import (
	"flag"
	"fmt"
	"heis/communication"
	"heis/config"
	"heis/driver"
	//"heis/hallCallsAssigner"
	"heis/localElevator"
	"heis/network/localip"
	"heis/worldview"
	"os"
	"time"
)

func main() {

	//for å kjøre på fysik heis og sim
	// ./SimElevatorServer --port randum num (e.g 16042) --id elevator-num (e.g elevator-1) og go run . --server localhost:16042 --id elevator-num
	// i annen terminal elevatorserver
	// go run . --server 

	//for å kjøre på samme pc:
	// ./SimElevatorServer --port 15657 og go run . --server localhost:15657 --id elevator-1
	// ./SimElevatorServer --port 15658 og go run . --server localhost:15658 --id elevator-2

	//uses the local ip address + an id given on the command line to
	//create localID
	serverAddr := flag.String("server", "localhost:15657", "Elevator server address")
	idFlag := flag.String("id", "", "Elevator ID (optional)")
	flag.Parse()

	var id string
	if *idFlag != "" {
		id = *idFlag
	} else {
		localIP, err := localip.LocalIP()
		if err != nil {
			fmt.Println(err)
			localIP = "DISCONNECTED"
		}
		id = fmt.Sprintf("peer-%s-%d", localIP, os.Getpid())
	}

	fmt.Printf("Starting elevator %s connecting to %s\n", id, *serverAddr)

	driver.Init(*serverAddr, config.N_FLOORS)

	//Her mottas en recoverd state om ønsket fra communication module
	recoveredCabCallsCh := make(chan [config.N_FLOORS]bool, 1)

	//sensor channels
	floorSensorChan := make(chan int, 1)
	obstructionChan := make(chan bool, 1)
	stopBtnChan := make(chan bool, 1)
	buttonChan := make(chan driver.ButtonEvent, 1)
	worldviewButtonChan := make(chan driver.ButtonEvent, 1)

	//localElevator til Worldview
	elevStateToWorldview := make(chan localElevator.ElevState, 1)

	//HallCallAssigner til localELevator
	assignerToLocalElev := make(chan [config.N_FLOORS][2]bool, 1)

	//input kanal til hallCallsAssigner:
	worldviewToHallCallAssigner := make(chan worldview.HRAInput, 1)

	//worldview til communication:
	worldviewToCommuncation := make(chan worldview.PeerState, 1)

	//communication til worldview:
	peersConnectedCh := make(chan [config.N_ELEVATORS-1]bool, 1)

	peerStateChs := [config.N_ELEVATORS-1]chan worldview.PeerState{} //gis til communication
	peerStateChsReadOnly := [config.N_ELEVATORS-1]<-chan worldview.PeerState{} //gis til worldview 
	for i := range peerStateChs {
		peerStateChs[i] = make(chan worldview.PeerState, 1)
		peerStateChsReadOnly[i] = peerStateChs[i]
	}

	//Aktiver driver polling
	go driver.PollButtons(buttonChan)
	go driver.PollButtons(worldviewButtonChan)
	go driver.PollFloorSensor(floorSensorChan)
	go driver.PollObstructionSwitch(obstructionChan)
	go driver.PollStopButton(stopBtnChan)

	communication.NewCommunicationModule(
		id, 
		config.BroadcastPort, 
		worldviewToCommuncation, 
		peerStateChs, 
		recoveredCabCallsCh, 
		peersConnectedCh,
	)

	//Initialiser moduler

	initialCabCalls := checkForBackupState(recoveredCabCallsCh)

	worldview.NewWorldViewModule(
		peerStateChsReadOnly,
		worldviewToHallCallAssigner,
		worldviewButtonChan,
		worldviewToCommuncation,
		elevStateToWorldview,
		id,
		peersConnectedCh,
	)

	localElevator.NewLocalElev(floorSensorChan,
		obstructionChan,
		buttonChan,
		elevStateToWorldview,
		assignerToLocalElev,
		initialCabCalls,
	)

	//input: InputChan chan HRAInput, OutputChan chan [config.N_Floors][2]bool, ID string

	//hallCallsAssigner.NewHallCallAssigner(worldviewToHallCallAssigner, assignerToLocalElev, id)
	select {}
}

func checkForBackupState(recoverdCabCallsCh <-chan [config.N_FLOORS]bool) [config.N_FLOORS]bool {
	now := time.Now()
	recoverdCabCalls := [config.N_FLOORS]bool{}
	for {
		if time.Since(now) > config.IntialStateCheckTime*time.Millisecond {
			return recoverdCabCalls
		}
		select {
		case recoverdCabCalls = <-recoverdCabCallsCh:
			fmt.Println("[main] Backup state received, starting with recovered state")
		default:
		}
	}
}