package main

import (
	"flag"
	"fmt"
	"heis/communication"
	"heis/config"
	"heis/driver"
	"heis/model"
	//"heis/hallCallsAssigner"
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

	//KODEKVALITET: navngivning
	////InitialCabCalls - kanal skal ha flyt av initial cabCalls 
	recoveredCabCallsCh := make(chan [config.N_FLOORS]bool, 1) //kanal for reco

	floorSensorChan := make(chan int, 1)
	obstructionChan := make(chan bool, 1)
	stopBtnChan := make(chan bool, 1)


	//KODEKVALITET ENDRINGER


	//buttonChan := make(chan driver.ButtonEvent, 1)
	//worldviewButtonChan := make(chan driver.ButtonEvent, 1)

	//---------------


	//KODEKVALITET: navngignine. Kanalnavn er tunge 
	//navn skal beskrive hva man får ved å bruke den: assignerToLocalElev til assignedHallCalls
	//unngå retning som To. Det viktige er betydning: worldviewToHallCallAssigner til .. og worldviewToCommuncation til peerStateUpdates
	//Channels kan navgis som strømmer: assignedHallCalls, buttonEvents osv. Ikke bruk chs for det er stygg forkortelse 
	//vær konsekvent på bruk: f.eks events 


	elevStateToWorldview := make(chan localElevator.ElevState, 1) //localElevatorState
	assignerToLocalElev := make(chan [config.N_FLOORS][2]bool, 1)//assignedHallCalls
	worldviewToHallCallAssigner := make(chan worldview.HRAInput, 1) //hallCallAssignerInput
	worldviewToCommuncation := make(chan worldview.PeerState, 1) //peerStateUpdates
	peersConnectedCh := make(chan [config.N_ELEVATORS-1]bool, 1) //connectedPeers
	peerStateChs := [config.N_ELEVATORS-1]chan worldview.PeerState{} //PeerStatesChannels / peerStateOutputs
	peerStateChsReadOnly := [config.N_ELEVATORS-1]<-chan worldview.PeerState{} //com-skriverettighet på peerStateOutputs, ww - leserettighet på peerStateOutputs
	//men fokusere på implementasjon og er lang. peerStateInputs
	for i := range peerStateChs {
		peerStateChs[i] = make(chan worldview.PeerState, 1)
		peerStateChsReadOnly[i] = peerStateChs[i]
	}



	//KODEKVALITET ENDRING: navnendring + legge til kanaler etter oversettelse i main . ufullført


	//go driver.PollButtons(buttonChan)
	//go driver.PollButtons(worldviewButtonChan)

	rawButtonEvents := make(chan driver.ButtonEvent, 1)
	localButtonEvents := make(chan driver.ButtonEvent, 1)
	hallCallEvents := make(chan model.HallCallEvent, 1)


	rawButtonEvents := make(chan driver.ButtonEvent, 1)
	go driver.PollButtons(driverButtonChan) //Vi henter ut herfra 
	localButtonEvents := make(chan driver.ButtonEvent, 1) //localButtonEvents er bedr navn 
	hallCallEvents := make(chan model.HallCallEvent, 1)//det wordwiew mottar
	go fanOutButtons(driverButtonChan, buttonChan, hallCallEventChan) //tar inn driverButtonChan og formatere utverdier på buttonChan  og hallCallEventChan
	
	//-----------------------------
	go driver.PollFloorSensor(floorSensorChan)
	go driver.PollObstructionSwitch(obstructionChan)
	go driver.PollStopButton(stopBtnChan) //DEN KAN FJERNES? 



	communication.NewCommunicationModule( //endre navn her som sant over
		id, 
		config.BroadcastPort, 
		worldviewToCommuncation, 
		peerStateChs, 
		recoveredCabCallsCh, 
		peersConnectedCh,
	)

	//SetInitialCabCalls bør være funksjonavn- den skal retunrer initial cabCakks
	//Men hvor settes denne til null standard? bør ikke det v'e her?
	initialCabCalls := checkForBackupState(recoveredCabCallsCh) //checkForBackupState - den gjør mer enn å sjekke, den returner osv. getRecoveredCabCalls s

	//KODEKVALITET:endre navn som nevnt tildiger 
	worldview.NewWorldViewModule(
		peerStateChsReadOnly,
		worldviewToHallCallAssigner,
		worldviewButtonChan,
		worldviewToCommuncation,
		elevStateToWorldview,
		id,
		peersConnectedCh,
	)

	//kalle denne NewLocalElevModule?
	localElevator.NewLocalElev(
		floorSensorChan, //byttes til floorUpdates
		obstructionChan, //peerStateInputs
		buttonChan, //	localButtonEvents
		elevStateToWorldview, //localElevatorState
		assignerToLocalElev, //assignedHallcalls
		initialCabCalls,
	)

	
	select {} //hva er denne`?`
}


//obs over. Denne handler om init så bra her - samle i GetInitCabCalls? 
func checkForBackupState(recoverdCabCallsCh <-chan [config.N_FLOORS]bool) [config.N_FLOORS]bool { //recoverdCabCalls, 
	now := time.Now()
	recoverdCabCalls := [config.N_FLOORS]bool{} //cabCalls
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


///KODEKVALITET ENDRING: Dette gjør at worldview og local elevator slipper å forstå driver logikk (pakke ut tastetrykk)

func fanOutButtons(
    in <-chan driver.ButtonEvent,
    toLocalElev chan<- driver.ButtonEvent,
    toWorldview chan<- model.HallCallEvent,
) {
    for btn := range in {

        // 1. send ALT til localElevator
        toLocalElev <- btn

        // 2. send KUN hall calls til worldview
        switch btn.Button {
        case driver.BT_HallUp:
            toWorldview <- model.HallCallEvent{
                Floor:  btn.Floor,
                Button: model.HallUp,
            }
        case driver.BT_HallDown:
            toWorldview <- model.HallCallEvent{
                Floor:  btn.Floor,
                Button: model.HallDown,
            }
        }
    }
}


//------------------