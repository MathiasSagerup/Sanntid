package communication

import "time"
import "heis/model"


type ElevatorState struct {
	ID string
	Floor int
	Dir int
	Behavior int
}

type HallCalls struct {
	Up map[int] bool
	Down map[int] bool
}

type Worldview struct {
	MyID string
	Elevators map[string]ElevatorState
	Halls HallCalls
}

type Message struct {
	FromID string
	Elevators map[string]ElevatorState
	Halls HallCalls
}


func NewWorldView(myID string) *Worldview {
	//return updated wordl wiev
}

func Encode(data []bytes)(Messager, error){ //tar inn rå data og gir ut type Message og en error 
	//return message 
}

func OrHalls(ver1 HallCalls, ver2 HallCalls) HallCalls {
	//merge hall calls  ved ulike versjoner med OR
}

func UpdateFromPeer(msg Message) pointerToWorld{
	//opdater worldView fra mer infor
}

func BuildOutgoingMessafe(informationnbeeded) Message{
	//decode til network
}











task CommunicationTask(
	myID, 
	inNet: chan bytes //fra nettet 
	outNet: chan bytes //til nett for broadcast
	inlocal: chan LocalState //ny lokal state fra localElevator
	outWold: chan WorldSNap //snapcshot av world 
	period: Duration 
)

func CommunicationTask() {

	var worldview Worldview //variabel for worldview settes som struktur 

	wordview := NewWorldView() //lager tomt objekt av type worldView
	world.elevators[myID] := DefaultLocalState() //start tilstand 
	ticker := NewTicker(Period); // trigger broadcast

	for {
		select {
		case msg <- inNet:
			
			msg, ok := Decode(data) //dekoder data fra channel og sjekkr gyldighet 
			of not ok: 
				//ignorere 

			if ok: 

				//merge state
				peerID := msg.FromID//check peer id from message
				if peerID is not my ID 
					peerState := msg.elevators[peerID] //overskriv siste state fra peerID
					world.updatePeer(msg) //oppdater  med peer info

				//merge hall calls
				world.halls = OrHallCalls(world.halls, msg.Halls)

		case msg <- inLocal:
				localState = msg
				peerState := msg.elevators[myID] //overskriv siste state fra peerID

		case ev <- inHallEvent:
				world.halls = ApplyHallCallEvent(world.halls, ev) //legg til knappetrykk 

		case <- tickerBroadcast
				
				//construct message
				outMsg := Message{
					FromID: myID, 
					Elevators: CopyElevatorMap(world.elevators), //kopi av data for å hindre global data 
					Halls: CopyHalls(world.halls), //kopi 
				}
				//encode and send 
				bytes, ok := Encode(outMsg)

				if ok: 
					outNet <- bytes //send hvis ok 

		}
	}
}


func NewWorlView(myId) (model.)

















///---------------------------------------------------------------

// broadcasts worldview, ElevID indicates which elevator sent info
// seqNum indicates how many messges elev has sent
// receiving Elevators can use this to compare if their data is old
func broadcastWorldview(ElevID ID, worldview Worldview, SendSeqNum int) {

}

func updateWorldview(msg message) ([]bytes, error) {

	var msg message

}

func isBroadcastMsgOld() {

}

// func

// go func communicationFSM () {

// 	var WorldviewElev1 Elevator
// 	WorldViewElev1Ch:= make(chan Elevator)
// 	Elev1AbleToTakeOrders:= make(chan Elevator)

// 	var WorldviewElev2 Elevator
// 	WorldviewElev2:= make(chan Elevator)
// 	Elev2AbleToTakeOrders:= make(chan Elevator)

// 	localElevWorldview:= make(chan Elevator)

// 	for {

// 		select{

// 		case AvailablePeers:= <- peerUpdateCh: //Endring i antall noder i nettverket
// 			if AvailablePeers.New != "" {
// 				//new node joined network
// 				//which elevator was it? Set as alive,
// 				//possibly as readyToTakeOrder
// 			} else if len(AvailablePeers.Lost > 1) {
// 				//flere noder har dødd, det betyr at det er den lokale
// 				//noden som har mistet nettverkstilkobling
// 				localElevWorldview.AbleToServiceHallOrders = false

// 			} else if len(AvailablePeers.Lost == 1){
// 				//which node?

// 			} else {
// 				//happens on the second update after a node joins
// 				//or leaves network, everything normal
// 			}

// 		case <- time.After(interval) //15*millisekund så broadcaste state
// 			broadcastWorldview()

// 		case WorldviewElev1 <-

// 		case WorldviewElev2 <-

// 		case

// 		case:

// 		}
// 	}

// }
