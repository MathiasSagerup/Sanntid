package model


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
	myID string
	Elevators map[string]ElevatorState
	Halls HallCalls
}

type Message struct {
	FromID string
	Elevators map[string]ElevatorState
	Halls HallCalls
}

