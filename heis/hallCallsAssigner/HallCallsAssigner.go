package hallCallsAssigner

func (h *HallCallAssigner) GetAssignedHallCalls(state ElevatorState) AssignedHallCalls {
	respChan := make(chan AssignedHallCalls)
	h.assignedHallCallRequestChan <- assignedHallCallRequest{
		state:        state,
		responseChan: respChan,
	}
	return <-respChan
}
