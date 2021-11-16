package stubs

var Processsor = "GameofLifeOperations.Processor"

type Params struct {
	Turns       int
	Threads     int
	ImageWidth  int
	ImageHeight int
}
type Response struct {
	World [][]byte
	P     Params
	Turns int

	//gol.State
}

type Request struct {
	World  [][]byte
	P      Params
	Turns  int

	//gol.State
}

