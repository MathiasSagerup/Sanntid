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

	driver.Init(*serverAddr, config.N_FLOORS)

	//Her mottas en recoverd state om ønsket fra communication module
	//ENDRET NAVN og flyttet recoveredCabCalls := make(chan [config.N_FLOORS]bool, 1) til nærmere der den brukes 

	//sensor channels
	//ENDRET NAVN 
	floorEvents := make(chan int, 1) //
	obstructionEvents := make(chan bool, 1)//
	stopButtonEvents := make(chan bool, 1)//
	//ENDRE DETTE? 
	buttonEvents := make(chan driver.ButtonEvent, 1) //ENDRET NAVN
	worldviewbuttonEvents := make(chan driver.ButtonEvent, 1) //HVA R DETTE? ENDRET NAVN 


	//localElevator til Worldview
	localElevatorState := make(chan localElevator.ElevState, 1)//
	completedHallCalls := make(chan [config.N_FLOORS][2]bool, 16)//
	assignedHallcalls := make(chan [config.N_FLOORS][2]bool, 1)//ENDRET NAVN
	hallcallassignerInput := make(chan hallCallsAssigner.HRAInput, 1)//ENDRET NAVN
	peerStateUpdates := make(chan worldview.PeerState, 1)//ENDRET NAVN

	//communication til worldview:
	//HVA TENKER VI HER PÅ DENNE DELEN? 
	connectedPeers := make(chan [config.N_ELEVATORS-1]bool, 1)//endret navn 
	peersStateOutputs:= [config.N_ELEVATORS-1]chan worldview.PeerState{} //endret navn 
	peerStateInputs := [config.N_ELEVATORS-1]<-chan worldview.PeerState{} //endret navn

	for i := range peersStateOutputs {
		peersStateOutputs[i] = make(chan worldview.PeerState, 1)
		peerStateInputs[i] = peersStateOutputs[i]
	}

	//Aktiver driver polling
	go driver.PollButtons(buttonEvents)
	go driver.PollButtons(worldviewbuttonEvents)
	go driver.PollFloorSensor(floorEvents)
	go driver.PollObstructionSwitch(obstructionEvents)
	go driver.PollStopButton(stopButtonEvents)

	communication.NewCommunicationModule(
		id, 
		config.BroadcastPort, 
		peerStateUpdates, 
		peersStateChannels, 
		recoveredCabCalls, 
		connectedPeers,
	)

	//Initialiser moduler

	recoveredCabCalls := make(chan [config.N_FLOORS]bool, 1)//config ved oppstart. Flyttet denne ned hit. 
	initialCabCalls := loadRecoveredCabCalls(recoveredCabCalls) //endret navn 

	worldview.NewWorldViewModule(
		otherPeersStateReadOnlyChs,
		hallcallassignerInput,
		worldviewbuttonEvents,
		peerStateUpdates,
		localElevatorState,
		id,
		connectedPeers,
		completedHallCalls,
	)

	localElevator.NewLocalElev(
		floorEvent,
		obstructionEvent,
		buttonEvents,
		localElevatorState,
		assignedHallcalls,
		completedHallCalls,
		initialCabCalls,
	)

	//input: InputChan chan HRAInput, OutputChan chan [config.N_Floors][2]bool, ID string

	hallCallsAssigner.NewHallCallAssigner(hallcallassignerInput, assignedHallcalls)
	select {}//HVA GJØR DENNE 
}

func loadRecoveredCabCalls(recoverdCabCalls <-chan [config.N_FLOORS]bool) [config.N_FLOORS]bool { //navnendring
	start := time.Now()//navnendring
	cabCalls := [config.N_FLOORS]bool{} //
	for {
		if time.Since(start) > config.IntialStateCheckTime*time.Millisecond {
			return cabCalls
		}
		select {
		case cabCalls = <-recoverdCabCalls:
			fmt.Println("[main] Backup state received, starting with recovered state")
		default:
		}
	}
}