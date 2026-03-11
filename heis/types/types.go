package types

import "time"

// ---- Constants ----

const (
	N_FLOORS         = 4
	N_BUTTONS        = 3
	DoorOpenDuration = 3 * time.Second
)

// ---- Motor Direction ----

type MotorDirection int

const (
	MD_Up   MotorDirection = 1
	MD_Down MotorDirection = -1
	MD_Stop MotorDirection = 0
)

// ---- Button Types ----

type ButtonType int

const (
	BT_HallUp   ButtonType = 0
	BT_HallDown ButtonType = 1
	BT_Cab      ButtonType = 2
)

// ---- Elevator Behaviour ----

type ElevatorBehaviour int

const (
	Idle ElevatorBehaviour = iota
	Moving
	DoorOpen
)

// ---- Internal Types ----

type ButtonEvent struct {
	Floor  int
	Button ButtonType
}

type DirnBehaviourPair struct {
	Dirn      MotorDirection
	Behaviour ElevatorBehaviour
}

type AssignedHallCalls [][2]bool

// ---- Internal Elevator State ----
// Used between modules inside one elevator

type ElevatorState struct {
	Floor                 int
	Dirn                  MotorDirection
	Behaviour             ElevatorBehaviour
	Requests              [N_FLOORS][N_BUTTONS]bool
	Obstruction           bool
	AbleToServiceRequests bool
}

// ---- Network Message Types ----
// These are sent over UDP via bcast - must be JSON serializable
// Each must be a unique type since bcast uses type name as tag

// Sent by each elevator to broadcast its current state
type ElevatorStateMsg struct {
	ID          string `json:"id"`
	Behaviour   string `json:"behaviour"` // "idle" | "moving" | "doorOpen"
	Floor       int    `json:"floor"`
	Direction   string `json:"direction"` // "up" | "down" | "stop"
	CabRequests []bool `json:"cabRequests"`
}

// Sent to broadcast the current known hall requests
type HallRequestMsg struct {
	HallRequests [][2]bool `json:"hallRequests"`
}

// ---- Hall Call Assigner JSON Types ----
// Used when calling the external hall_request_assigner binary

type AssignerInput struct {
	HallRequests [][2]bool                   `json:"hallRequests"`
	States       map[string]ElevatorStateMsg `json:"states"`
}

type AssignerOutput map[string][][2]bool

// ---- Worldview ----
// The known state of all elevators on the network

type Worldview struct {
	ElevatorStates map[string]ElevatorStateMsg
	HallRequests   [][2]bool
	Peers          []string
}

// ---- Peer Update ----
// Mirrors peers.PeerUpdate but kept in types to avoid
// importing the network package from non-network modules

type PeerUpdate struct {
	Peers []string
	New   string
	Lost  []string
}
