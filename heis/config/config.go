package config

const (
	N_FLOORS    = 4
	N_ELEVATORS = 3
	N_BUTTONS   = 3 // HallUp, HallDown, Cab
	PEER_TIMEOUT_DURATION = 1000 // milliseconds
	DoorOpenDuration = 3.0 // seconds

	checkForInitialState = true
	IntialStateCheckTime = 1000 // milliseconds

	HallCallAssignerExec = "Project-resources/cost_fns/hall_request_assigner/hall_request_assigner"
	BroadcastPort        = 20007 //port used to broadcast worldviews. Should be 20000 + station number
	PeersPort            = 19475 //port used to discover heartbeats from other elevs
)
