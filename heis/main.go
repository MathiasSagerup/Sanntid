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

	//Used from communication module at system initialization
	recoveredCabCallsCh := make(chan [config.N_FLOORS]bool, 1)

	//Channels from driver
	floorSensorCh := make(chan int, 1)
	obstructionCh := make(chan bool, 1)
	buttonEventCh := make(chan driver.ButtonEvent, 1)
	worldviewButtonEventCh := make(chan driver.ButtonEvent, 1)

	//localElevator to worldview
	localElevatorStateCh := make(chan localElevator.ElevState, 1)
	completedHallCallsCh := make(chan [config.N_FLOORS][2]bool, 16)

	//hallCallAssigner to localELevator
	assignedHallCallsCh := make(chan [config.N_FLOORS][2]bool, 1)

	//worldview to hallCallsAssigner:
	hallCallsAssignerInputCh := make(chan hallCallsAssigner.HRAInput, 1)

	//worldview to communication:
	localPeerStateCh := make(chan worldview.PeerState, 1)

	//communication to worldview:
	peersConnectedCh := make(chan [config.N_OTHER_ELEVATORS]bool, 1)
	otherPeersStateChs := [config.N_OTHER_ELEVATORS]chan worldview.PeerState{}

	otherPeersStateInputChs := [config.N_OTHER_ELEVATORS]<-chan worldview.PeerState{}
	for i := range otherPeersStateChs {
		otherPeersStateChs[i] = make(chan worldview.PeerState, 1)
		otherPeersStateInputChs[i] = otherPeersStateChs[i]
	}

	//Initialization of modules

	driver.Init(*serverAddr, config.N_FLOORS)
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

	initialCabCalls := setInitialCabCalls(recoveredCabCallsCh)

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

	hallCallsAssigner.NewHallCallAssigner(hallCallsAssignerInputCh, assignedHallCallsCh)
	select {}
}

func setInitialCabCalls(recoveredCabCallsCh <-chan [config.N_FLOORS]bool) [config.N_FLOORS]bool {
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