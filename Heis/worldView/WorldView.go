package worldview

const N_FLOORS = 4
const N_ELEVATORS = 3

type elevatorID int

type Direction int
const (
	DirStop Direction = iota
	DirUp
	DirDown
)

type Behaviour int
const (
	BehIdle Behaviour = iota
	BehMoving
	BehDoorOpen
)

type ElevState struct {
	Floor             	int
	Dir               	Direction
	Behaviour         	Behaviour
	CabRequests       	[N_FLOORS]bool
	AbleToServiceOrders bool
	HallCalls			HallCallStates
}

type HallCallActivation int
const (
	NoOrder HallCallActivation = iota
    Unconfirmed
    Confirmed
    Completed
)

type HallCallStates [N_FLOORS][2]HallCallActivation

// World View Decider module ------------------------------------------------------------------------------------------

type WorldViewDecider struct {
	thisElevState ElevState
	otherElevStates []ElevState //Index corresponds to ElevID

	messageFromLocalElevChannel <-chan ElevState 
	messageFromOtherElevChannels []<-chan ElevState //Index in array corresponds to ElevID

}


func InitializeWorldViewModule(
		messageFromOtherElevChannels []<-chan ElevState,
		initialLocalElevState ElevState,
		initialOtherElevStates []ElevState,
	) (*WorldViewDecider) {

	//Assert correct length of array to a contsistent amount of elevators

	w := &WorldViewDecider{
		messageFromOtherElevChannels: messageFromOtherElevChannels,
		thisElevState: initialLocalElevState,
		otherElevStates: initialOtherElevStates,
	}
	
	go w.loop()

	return w
}

func (w *WorldViewDecider) loop() {
	var hallCallsHasChanged bool
	var elevatorStateHasChanged bool
	for {
		hallCallsHasChanged = false
		elevatorStateHasChanged = false

		//Check for message from the local elevator
		select{
		case newElevState := <- w.messageFromLocalElevChannel:
			if (newElevState != w.thisElevState){
				w.thisElevState = newElevState
				elevatorStateHasChanged = true
			}
		default:
		}

		//Check for message from all other elevators
		for elevID := 0; elevID <= len(w.messageFromOtherElevChannels); elevID++ {
			select{
			case newElevState := <- w.messageFromOtherElevChannels[elevID]:
				if newElevState != w.otherElevStates[elevID]{
					w.otherElevStates[elevID] = newElevState
					elevatorStateHasChanged =  true
					w.compareWithAndUpdateLocalHallcalls(newElevState.HallCalls, elevatorID(elevID), hallCallsHasChanged)
				}
			default:
			}
		}

		if elevatorStateHasChanged || hallCallsHasChanged {
			w.sendUpdatedInformationToHallCallAssigner()
		}
	}
}

func (w *WorldViewDecider) recieveOtherElevMessage(incomingElevState ElevState, senderElevID elevatorID, hallCallsHasChanged bool){
	if incomingElevState.HallCalls != w.thisElevState.HallCalls{
		w.otherElevStates[senderElevID] = incomingElevState
		w.compareIncomingHallCalls(incomingElevState.HallCalls)
		hallCallsHasChanged = true
	}
}

func (w *WorldViewDecider) compareIncomingHallCalls(incomingHallCalls HallCallStates) {
	for floor:=0; floor<N_FLOORS; floor++{
		for direction:=0; direction<2; direction++{
			w.compareSingleIncomingHallCall(incomingHallCalls[floor][direction], floor, direction)
		}
	}
}

func (w *WorldViewDecider) compareSingleIncomingHallCall(incomingHallCallActivation HallCallActivation, floor int, direction int){
	switch w.thisElevState.HallCalls[floor][direction]{
	case NoOrder:
		switch incomingHallCallActivation{
		case Unconfirmed:
			w.thisElevState.HallCalls[floor][direction] = Unconfirmed
		case Confirmed:
			w.thisElevState.HallCalls[floor][direction] = Confirmed
		default:
		}
	
	case Unconfirmed:
		switch incomingHallCallActivation{
		case Unconfirmed:
			//If we recieve unconfirmed we must check if all elevators agree on unconfirmed and change to confirmed
			unconfirmedCounter := 0				
			for elevID := 0; elevID < len(w.otherElevStates); elevID++{
				if (w.otherElevStates[elevID].AbleToServiceOrders) && (w.otherElevStates[elevID].HallCalls[floor][direction] == Unconfirmed) {
					unconfirmedCounter++
				}
			}
			if unconfirmedCounter == w.getOtherElevsAliveCount(){
				w.thisElevState.HallCalls[floor][direction] = Confirmed
			}
		case Confirmed:
			w.thisElevState.HallCalls[floor][direction] = Confirmed
		default:
		}
	
	case Confirmed:
		switch incomingHallCallActivation{
		case Completed:
			w.thisElevState.HallCalls[floor][direction] = Completed
		default:
		}
	case Completed:
		switch incomingHallCallActivation{
		case Completed:
			//If we recieve completed we must check if all elevators agree on completed and if so, change to NoOrder
			completedCounter := 0
			for elevID := 0; elevID < len(w.otherElevStates); elevID++{
				if(w.otherElevStates[elevID].AbleToServiceOrders) && (w.otherElevStates[elevID].HallCalls[floor][direction] == Completed){
					completedCounter++
				}
			}
			if completedCounter == w.getOtherElevsAliveCount(){
				w.thisElevState.HallCalls[floor][direction] = NoOrder

			}
		case NoOrder:
			w.thisElevState.HallCalls[floor][direction] = NoOrder
		}
	}
}

func (w *WorldViewDecider) getOtherElevsAliveCount() int {
	counter := 0
	for elevID := 0; elevID < len(w.otherElevStates); elevID++ {
		if w.otherElevStates[elevID].AbleToServiceOrders{
			counter++
		} 
	}
	return counter
}

func (w *WorldViewDecider) sendUpdatedInformationToHallCallAssigner() {
	//Implement
}