package config

import "time"

const (
	N_FLOORS				= 4
	N_ELEVATORS				= 3
	N_BUTTONS             	= 3      // HallUp, HallDown, Cab
	PEER_TIMEOUT_DURATION_MS 	= 1000 
	N_OTHER_ELEVATORS		= N_ELEVATORS - 1 
	N_TRAVEL_DIRN			= 2
	DoorOpenDuration      	= 3 * time.Second
	MotorLossDuration     	= 10 * time.Second

	checkForInitialState 	= true
	IntialStateCheckTime 	= 1000 // milliseconds

	HallCallAssignerExec 	= "Project-resources/cost_fns/hall_request_assigner/hall_request_assigner"
	BroadcastPort        	= 20007 //port used to broadcast worldviews. Should be 20000 + station number
	PeersPort            	= 19475 //port used to discover heartbeats from other elevs
)
