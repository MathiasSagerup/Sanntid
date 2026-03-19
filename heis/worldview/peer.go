package worldview

import (
	"heis/config"
	"heis/localElevator"
)

type PeerState struct {
	LocalElevState localElevator.ElevState
	HallCalls [config.N_FLOORS][config.N_TRAVEL_DIRN]OrderState	
}

func (w *WorldView) getNumberOfConnectedPeers() int{
	connectedPeers := 0
	for elevID := 0; elevID < len(w.connectedElevators); elevID++{
		if w.connectedElevators[elevID]{
			connectedPeers++
		}
	}
	return connectedPeers
}

