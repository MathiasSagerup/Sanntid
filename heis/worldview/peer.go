package worldview

import (
	"heis/config"
	"heis/localElevator"
)

type PeerState struct {
	LocalElevState localElevator.ElevState
	HallCalls [config.N_FLOORS][2]OrderState	
}

func (w *WorldViewDecider) getNumberOfConnectedPeers() int{
	connectedPeers := 0
	for elevID := 0; elevID < len(w.connectedElevators); elevID++{
		if w.connectedElevators[elevID] == true {
			connectedPeers++
		}
	}
	return connectedPeers
}

