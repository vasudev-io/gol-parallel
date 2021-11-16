package util

import "uk.ac.bris.cs/gameoflife/gol"

var Processsor = "GameofLifeOperations.Processor"


type Response struct {
	World [][]byte
	gol.Params
	//gol.State
}

type Request struct {
	World [][]byte
	gol.Params
	//gol.State
}

