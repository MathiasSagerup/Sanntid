package communication



type Communication struct {
	WorldViewACh chan WorldView
	WorldViewBCh chan WorldView
}

func InitializeCommunication() (*Communication){
	c := &Communication{
		worldViewACh: make(chan WorldView),
		worldViewBCh: make(chan WorldView),
	}

	return c
}