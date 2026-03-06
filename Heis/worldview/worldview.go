/*
This module is made to keep hold of our current view of the overall system state. This includes the most recent
state update from each elevator and hall calls.

It takes new information updates from the communication module and elvator module to determine the current elevator states

It takes new infromation updates from the driver and communication module to determine what the hall call states are.

Since this module contains the overall view of what hall calls are active it also sets hall call lights by accessing the driver module
*/

package worldview

import "heis/ElevState"

type ElevatorStates struct {
	This ElevState
	ExternalA ElevState
	ExternalB ElevState
}

type WorldView struct{



}

func InitializeWorldView() *WorldView{
	w := &WorldView{}
}