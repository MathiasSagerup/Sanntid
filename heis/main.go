package main

import (
	"flag"
	"fmt"
	"heis/communication"
	"heis/config"
	"heis/driver"
	"heis/hallCallsAssigner"
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

	//sensor channel
	floorSensorCh := make(chan int, 1)
	obstructionCh := make(chan bool, 1)
	buttonEventCh := make(chan driver.ButtonEvent, 1)
	worldviewButtonEventCh := make(chan driver.ButtonEvent, 1)

	//localElevator til Worldview
	localElevatorStateCh := make(chan localElevator.ElevState, 1)
	completedHallCallsCh := make(chan [config.N_FLOORS][2]bool, 16)

	//HallCallAssigner til localELevator
	assignedHallCallsCh := make(chan [config.N_FLOORS][2]bool, 1)

	//input kanal til hallCallsAssigner:
	hallCallsAssignerInputCh := make(chan hallCallsAssigner.HRAInput, 1)

	//worldview til communication:
	localPeerStateCh := make(chan worldview.PeerState, 1)

	//communication til worldview:
	peersConnectedCh := make(chan [config.N_OTHER_ELEVATORS]bool, 1)

	//SE PÅ DENNE SENERE? 

	otherPeersStateChs := [config.N_OTHER_ELEVATORS]chan worldview.PeerState{}

	otherPeersStateInputChs := [config.N_OTHER_ELEVATORS]<-chan worldview.PeerState{}
	for i := range otherPeersStateChs {
		otherPeersStateChs[i] = make(chan worldview.PeerState, 1)
		otherPeersStateInputChs[i] = otherPeersStateChs[i]
	}

	//Aktiver driver polling
	go driver.PollButtons(buttonEventCh)
	go driver.PollButtons(worldviewButtonEventCh)
	go driver.PollFloorSensor(floorSensorCh)
	go driver.PollObstructionSwitch(obstructionCh)

	communication.NewCommunicationModule(
		id, 
		config.BroadcastPort, 
		localPeerStateCh, 
		otherPeersStateChs, 
		recoveredCabCallsCh, 
		peersConnectedCh,
	)

	//Initialiser moduler

	initialCabCalls := setRecoveredCabCalls(recoveredCabCallsCh)

	worldview.NewWorldViewModule(
		otherPeersStateInputChs,
		hallCallsAssignerInputCh,
		worldviewButtonEventCh,
		localPeerStateCh,
		localElevatorStateCh,
		id,
		peersConnectedCh,
		completedHallCallsCh,
	)

	localElevator.NewLocalElev(
		floorSensorCh,
		obstructionCh,
		buttonEventCh,
		localElevatorStateCh,
		assignedHallCallsCh,
		completedHallCallsCh,
		initialCabCalls,
	)

	//input: InputChan chan HRAInput, OutputChan chan [config.N_Floors][2]bool, ID string

	hallCallsAssigner.NewHallCallAssigner(hallCallsAssignerInputCh, assignedHallCallsCh)
	select {}
}

func setRecoveredCabCalls(recoveredCabCallsCh <-chan [config.N_FLOORS]bool) [config.N_FLOORS]bool {
	now := time.Now()
	recoveredCabCalls := [config.N_FLOORS]bool{}
	for {
		if time.Since(now) > config.IntialStateCheckTime*time.Millisecond {
			return recoveredCabCalls
		}
		select {
		case recoveredCabCalls = <-recoveredCabCallsCh:
			fmt.Println("[main] Backup state received, starting with recovered state")
		default:
		}
	}
}