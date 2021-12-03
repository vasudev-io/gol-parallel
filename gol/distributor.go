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
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {

	//string conversion for the filename
	height := strconv.Itoa(p.ImageHeight)
	width := strconv.Itoa(p.ImageWidth)

	FileName := width + "x" + height
	c.ioCommand <- ioInput

	c.ioFilename <- FileName

	//2D slice to store the world
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	//updating it with the bytes sent from io.go
	for row := 0; row < p.ImageHeight; row++ {
		for col := 0; col < p.ImageWidth; col++ {
			world[row][col] = <-c.ioInput //not sure if it works
			if world[row][col] == 255 {
				cell := util.Cell{X: row, Y: col}
				c.events <- CellFlipped{CompletedTurns: 0, Cell: cell}
			}

		}
	}

	//the necessary elements to evolve the Game of Life are initialized
	turn := 0
	workerHeight := p.ImageHeight / p.Threads
	left := p.ImageHeight % p.Threads
	tk := time.NewTicker(2 * time.Second)

	out := make([]chan [][]byte, p.Threads)
	for k := range out {
		out[k] = make(chan [][]byte)
	}
	newWorldData := world

	// Execute all turns of the Game of Life.
	for turn < p.Turns {

		// Serial Implementation
		if p.Threads == 1 {

			newWorldData = calculateNextState(p, 0, p.ImageHeight, newWorldData)
			for row := 0; row < p.ImageHeight; row++ {
				for col := 0; col < p.ImageWidth; col++ {

					if newWorldData[row][col] != world[row][col] {
						cell := util.Cell{X: row, Y: col}
						c.events <- CellFlipped{CompletedTurns: turn, Cell: cell}
					}
				}
			}
			c.events <- TurnComplete{CompletedTurns: turn}
			turn++

		} else {

			//Parallel Implementation
			for i := 0; i < p.Threads; i++ {
				start := i * workerHeight
				end := (i + 1) * workerHeight

				// Used when there is an odd number of threads and there is a remainder on the modulus func
				if left > 0 && i == p.Threads-1 {
					go worker(p, newWorldData, start, p.ImageHeight, out[i])

				} else {

					// Works for all other cases, which means for all even cases
					go worker(p, newWorldData, start, end, out[i])
				}

			}

			//Appending of the slices of the world sent from each worker thread
			finalWorldData := make([][]byte, 0)
			for i := 0; i < p.Threads; i++ {
				part := <-out[i]
				finalWorldData = append(finalWorldData, part...)

			}

			//Updating the CellFlipped event so that the sdl window can be updated
			for row := 0; row < p.ImageHeight; row++ {
				for col := 0; col < p.ImageWidth; col++ {

					if newWorldData[row][col] != finalWorldData[row][col] {
						cell := util.Cell{X: row, Y: col}
						c.events <- CellFlipped{CompletedTurns: turn, Cell: cell}
					}
				}
			}
			c.events <- TurnComplete{CompletedTurns: turn}

			newWorldData = finalWorldData

			turn++

		}

		select {

		//the ticker which sends the AliveCellsCount to the events channel after every two seconds
		case <-tk.C:
			c.events <- AliveCellsCount{CompletedTurns: turn, CellsCount: len(calculateAliveCells(p, newWorldData))}

		//The keypresses which carry out specific tasks
		case x := <-keyPresses:

			// saving of the current state of the state of the board
			if x == 's' {
				c.ioCommand <- ioOutput

				fturn := strconv.Itoa(turn)
				FileName = width + "x" + height + "x" + fturn
				c.ioFilename <- FileName

				for row := 0; row < p.ImageHeight; row++ {
					for col := 0; col < p.ImageWidth; col++ {
						c.ioOutput <- newWorldData[row][col]
					}
				}

				c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: FileName}

				//saving the current state of the board and terminating the process
			} else if x == 'q' {
				c.ioCommand <- ioOutput

				fturn := strconv.Itoa(turn)
				FileName = width + "x" + height + "x" + fturn
				c.ioFilename <- FileName

				for row := 0; row < p.ImageHeight; row++ {
					for col := 0; col < p.ImageWidth; col++ {
						c.ioOutput <- newWorldData[row][col]
					}
				}

				c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: FileName}

				c.events <- FinalTurnComplete{CompletedTurns: turn, Alive: calculateAliveCells(p, newWorldData)}

				fmt.Println("Terminated.")

				//pausing the processing of the image , continuing on repeated click
			} else if x == 'p' {
				fmt.Println(turn)
				fmt.Println("Paused!")

				for {
					tempKey := <-keyPresses
					if tempKey == 'p' {
						fmt.Println("Continuing...")
						break
					}
				}
			}
		default:
		}
	}
	tk.Stop()

	// calculating the number of alivecells at the end of the loop
	alivers := calculateAliveCells(p, newWorldData)

	// and send that to a channel which goes into the FinalTurnComplete
	c.events <- FinalTurnComplete{CompletedTurns: turn, Alive: alivers}

	c.ioCommand <- ioOutput

	//Sending new filename to the channel
	fturn := strconv.Itoa(turn)
	FileName = width + "x" + height + "x" + fturn
	c.ioFilename <- FileName

	// sending the array of bytes from the world to the ioOutput channel
	for row := 0; row < p.ImageHeight; row++ {
		for col := 0; col < p.ImageWidth; col++ {
			c.ioOutput <- newWorldData[row][col]
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.events <- ImageOutputComplete{CompletedTurns: turn, Filename: FileName}

	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)

}

//worker function which takes the world works on a particular slice as directed in the distributor
func worker(p Params, world [][]byte, start, end int, out chan<- [][]byte) {

	workerWorldData := calculateNextState(p, start, end, world)

	out <- workerWorldData

}

// Calculates the next evolution of the board
func calculateNextState(p Params, startY, endY int, world [][]byte) [][]byte {

	//making a separate world to check without disturbing the actual world
	testworld := make([][]byte, endY-startY)
	for i := range testworld {
		testworld[i] = make([]byte, p.ImageWidth)
	}

	//copying the parts of the world from the original world
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

			//AliveNeighbourChecker For Loop - checks all 8 neighbours of the current cells to check if they have the pixel value of 255 (alive)
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

//Calculates whether a cell is alive and then updates this to the alive cells variable
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
