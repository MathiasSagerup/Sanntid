packages main 

import "fmt"
import "Network"
import "Driver"
import "heis/hallcallassigner"

main {
	CommunicationModule communiacetion := InitializeHallCallCommunication()
	HallCallAssigner hallcallassigner := InitializeHallCallAssigner(*communication)

	

	switch expression {
		case newhallcalls <- hallcallassigner.InterfaceChannels.AssignedHallcalls:
		
	}


}