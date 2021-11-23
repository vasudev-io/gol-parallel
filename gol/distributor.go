package gol

import (
	"strconv"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {

	//string conversion for the filename

	height := strconv.Itoa(p.ImageHeight)
	width := strconv.Itoa(p.ImageWidth)

	FileName := width + "x" + height
	c.ioCommand <- ioInput

	c.ioFilename <- FileName

	// TODO: Create a 2D slice to store the world.

	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	//updating it with the bytes sent from io.go
	for row := 0; row < p.ImageHeight; row++ {
		for col := 0; col < p.ImageWidth; col++ {
			world[row][col] = <-c.ioInput //not sure if it works
		}
	}

	// TODO: Execute all turns of the Game of Life.

	// the following would iterate calculating next state till done with turns
	turn := 0

	for turn < p.Turns {

		world = appender(world, 0, p.ImageHeight, p)
		//print_world(req.World, req.P.ImageHeight, req.P.ImageWidth)
		//fmt.Println()
		turn++
	}

	// send the next turn stuff thru to the response struct
	p.Turns = turn

	alivers := calculateAliveCells(p, world)

	final := FinalTurnComplete{CompletedTurns: p.Turns, Alive: alivers}
	c.events <- final

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

// report how many turns are over and how many alivers are remaining after each turn
// and send that to a channel which goes into the FinalTurnComplete

// TODO: Report the final state using FinalTurnCompleteEvent.

func calculateNextState(p Params, starty, endy int, world [][]byte) [][]byte {

	//making a separate world to check without disturbing the actual world
	testerworld := make([][]byte, len(world))
	for i := range world {
		testerworld[i] = make([]byte, len(world[i]))
		copy(testerworld[i], world[i])
	}

	for row := starty; row < endy; row++ {
		for col := 0; col < p.ImageWidth; col++ {

			alivemeter := 0
			for i := row - 1; i <= row+1; i++ {
				for j := col - 1; j <= col+1; j++ {

					if i == row && j == col {
						continue
					}

					if world[((i + p.ImageWidth) % p.ImageWidth)][(j+p.ImageHeight)%p.ImageHeight] == 255 {
						alivemeter++

					}
				}
			}

			// game of life conditions
			if alivemeter < 2 || alivemeter > 3 {
				testerworld[row][col] = 0
			}
			if alivemeter == 3 {
				testerworld[row][col] = 255
			}
		}
	}

	return testerworld
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {

	var alivecells []util.Cell

	for row := 0; row < p.ImageWidth; row++ {
		for col := 0; col < p.ImageHeight; col++ {

			pair := util.Cell{}
			currentCell := world[row][col]

			if currentCell == 255 {
				pair.X = col
				pair.Y = row
				alivecells = append(alivecells, pair)

			}
		}
	}

	return alivecells
}

func worker(world [][]byte, startY, endY, startX, endX int, p Params, out chan<- [][]byte) {
	//imagePart := worldslice(startY, endY, startX, endX, p)

	imagePart := world

	out <- imagePart
}

/*func worldslice(world [][] byte, startY, endY, startX, endX int, p Params) [][]byte {
	height := endY - startY
	width := endX - startX

	neWorldSlice:= world


	return
}
*/

func appender(world [][]byte, startY, endY int, p Params) [][]byte {

	var newWorldData [][]byte

	if p.Threads == 1 {
		newWorldData = calculateNextState(p, startY, endY, world)
	} else {
		workerHeight := p.ImageHeight / p.Threads
		out := make([]chan [][]byte, p.Threads)
		for i := range out {
			out[i] = make(chan [][]byte)
		}

		for i := 0; i < p.Threads; i++ {
			go worker(world, i*workerHeight, (i+1)*workerHeight, 0, p.ImageWidth, p, out[i])
		}

		newWorldData = world

		for i := 0; i < p.Threads; i++ {
			part := <-out[i]
			newWorldData = append(newWorldData, part...)
		}
	}

	return world

}
