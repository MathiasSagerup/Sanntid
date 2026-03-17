package model

import (
	"heis/config"
	"heis/driver"
)

type ElevatorBehaviour int

const (
	Idle ElevatorBehaviour = iota
	Moving
	DoorOpen
)

type OrderState int

const (
	NoOrder OrderState = iota
	Unconfirmed
	Confirmed
	Completed
)

type HallButton int

const (
	HallUp HallButton = iota
	HallDown
)

type HallCallEvent struct {
	Floor  int
	Button HallButton
}

type ElevatorState struct {
	Floor                 int
	Dirn                  driver.MotorDirection
	Behaviour             ElevatorBehaviour
	CabRequests           [config.N_FLOORS]bool
	Obstruction           bool
	AbleToServiceRequests bool
}

type PeerState struct {
	LocalElevState ElevatorState
	HallCalls      [config.N_FLOORS][2]OrderState
}