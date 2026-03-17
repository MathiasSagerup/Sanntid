package config

const (
	N_FLOORS              = 4
	N_ELEVATORS           = 3
	N_BUTTONS             = 3    // HallUp, HallDown, Cab
	PEER_TIMEOUT_DURATION = 1000 //milliseconds
	DoorOpenDuration      = 3.0  // seconds
	IntialStateCheckTime  = 2000

	HallCallAssignerExec = "Project-resources/cost_fns/hall_request_assigner/hall_request_assigner"
	BroadcastPort        = 20009 //port used to broadcast worlview to other elevators.
	PeersPort            = 19846 //port used to discover heartbeats by other ports.
)
