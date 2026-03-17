package config

const (
	N_FLOORS              = 4
	N_ELEVATORS           = 3
	N_BUTTONS             = 3    // HallUp, HallDown, Cab
	PEER_TIMEOUT_DURATION = 1000 //milliseconds
	DoorOpenDuration      = 3.0  // seconds
	IntialStateCheckTime  = 500

	HallCallAssignerExec = "Project-resources/cost_fns/hall_request_assigner/hall_request_assigner"
	BroadcastPort        = 20002 //port used to broadcast worlview to other elevators.
	PeersPort            = 20001 //port used to discover heartbeats by other ports.
)
