package gol

import (
	"fmt"
	"strconv"
	"time"
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

	// the worker pools to distribute the work to the different worker threads

	/*
		numthreads:=p.Threads

		jobs := make(chan int, numthreads)

		results := make(chan [][]byte, numthreads)

		for j := 1; j <= numthreads; j++{
			jobs <- j
		}
		close(jobs)

		for a := 1; a <= numthreads; a++{
			<-results
		}
	*/

	turn := 0

	var workerHeight int

	workerHeight = p.ImageHeight / p.Threads
	left := p.ImageHeight % p.Threads
	tk := time.NewTicker(2 * time.Second)

	out := make([]chan [][]byte, p.Threads)
	for k := range out {
		out[k] = make(chan [][]byte)
	}

	//workerS := make(chan int)
	//workerE := make(chan int)

	newWorldData := world

	for turn < p.Turns {

		if p.Threads == 1 {

			newWorldData = calculateNextState(p, 0, p.ImageHeight, newWorldData)
			turn++
		} else {

			for i := 0; i < p.Threads; i++ {
				start := i * workerHeight
				end := (i + 1) * workerHeight

				if left > 0 && i == p.Threads-1 {
					go worker(p, newWorldData, start, p.ImageHeight, out[i])

				} else {

					go worker(p, newWorldData, start, end, out[i])
				}

			}

			/*finalWorldData := make([][]byte, p.ImageHeight)
			for j := range finalWorldData {
				finalWorldData[j] = make([]byte, p.ImageWidth)
			}*/
			finalWorldData := make([][]byte, 0)
			for i := 0; i < p.Threads; i++ {
				part := <-out[i]
				finalWorldData = append(finalWorldData, part...)

			}

			newWorldData = finalWorldData

			turn++

		}

		fmt.Println(turn)

		select {
		case <-tk.C:
			c.events <- TurnComplete{CompletedTurns: turn}
			c.events <- AliveCellsCount{CompletedTurns: turn, CellsCount: len(calculateAliveCells(p, newWorldData))}

		}
	}
	tk.Stop()

	// TODO: Execute all turns of the Game of Life.

	// the following would iterate calculating next state till done with turns

	// send the next turn stuff thru to the response struct

	alivers := calculateAliveCells(p, newWorldData)

	final := FinalTurnComplete{CompletedTurns: turn, Alive: alivers}
	c.events <- final

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	c.ioCommand <- ioInput

	//c.ioFilename <- FileName

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

//func worldslice(p Params, world[][]byte) [][]byte {

/*var workerHeight int

workerHeight = p.ImageHeight / p.Threads

workerS := make(chan int)
workerE := make(chan int)

for i:=0;i<p.Threads; i++{
	start := i*workerHeight - 1
	end := (i+1)*workerH
eight + 1

	workerS <- start
	workerE <- end

	h := end - start

	worldslice1 := make([][]byte, h)
	for j := range worldslice1 {
		worldslice1[j] = make([]byte, p.ImageHeight)

	}*/

/*count := 0
	for i := start; i < end; i++ {
		for j := 0; j < p.ImageHeight; j++ {
			worldslice1[count][j] = world[i][j]
		}
		count++
	}

	return worldslice1
}*/

/*for j := 0; j <= height; j++ {
	for i:= 0; i <= p.ImageWidth; i++ {

		worldslice1 = world[:len(worldslice1)]

	}
}
*/

//}

func worker(p Params, world [][]byte, start, end int, out chan<- [][]byte) {

	workerWorldData := calculateNextState(p, start, end, world)

	out <- workerWorldData

}

// report how many turns are over and how many alivers are remaining after each turn
// and send that to a channel which goes into the FinalTurnComplete

// TODO: Report the final state using FinalTurnCompleteEvent.

func calculateNextState(p Params, startY, endY int, world [][]byte) [][]byte {

	//making a separate world to check without disturbing the actual world
	testworld := make([][]byte, endY-startY)
	for i := range testworld {
		testworld[i] = make([]byte, p.ImageWidth)
	}

	count := 0
	for x := startY; x < endY; x++ {
		for y := 0; y < p.ImageHeight; y++ {
			testworld[count][y] = world[x][y]
		}
		count++
	}

	plus := 0
	for row := startY; row < endY; row++ {
		for col := 0; col < p.ImageWidth; col++ {

			alivemeter := 0

			for i := row - 1; i <= row+1; i++ {
				for j := col - 1; j <= col+1; j++ {

					if i == row && j == col {
						continue
					}

					if world[((i + p.ImageHeight) % p.ImageHeight)][(j+p.ImageWidth)%p.ImageWidth] == 255 {
						alivemeter++

					}
				}
			}

			// game of life conditions
			if alivemeter < 2 || alivemeter > 3 {
				testworld[plus][col] = 0
			}
			if alivemeter == 3 {
				testworld[plus][col] = 255
			}

		}

		plus++
	}

	return testworld
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
