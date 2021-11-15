package gol

import (
	"bufio"
	"flag"
	"fmt"
	"net/rpc"
	"os"
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
/*func distributor(p Params, c distributorChannels) {




	//string conversion for the filename

	height := strconv.Itoa(p.ImageHeight)
	width := strconv.Itoa(p.ImageWidth)

	FileName := width+"x"+height
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

	//lol := make (chan []util.Cell)

	// the following would iterate calculating next state till done with turns
	turn:=0

	for turn <p.Turns{
		world = calculateNextState(p, world)
		turn ++
	}
	alivers := calculateAliveCells(p, world)
	//lol <- alivers



	// report how many turns are over and how many alivers are remaining after each turn
	// and send that to a channel which goes into the FinalTurnComplete




	// TODO: Report the final state using FinalTurnCompleteEvent.

	final := FinalTurnComplete{CompletedTurns: p.Turns, Alive: alivers}
	c.events <- final



	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateNextState(p Params, world [][]byte) [][]byte {

	//making a separate world to check without disturbing the actual world
	testerworld := make([][]byte, len(world))
	for i := range world{
		testerworld[i] = make([]byte, len(world[i]))
		copy(testerworld[i], world[i])
	}

	for row := 0; row < p.ImageHeight; row++ {
		for col := 0; col < p.ImageWidth; col++ {

			alivemeter := 0
			for i := row - 1; i <= row+1; i++ {
				for j := col - 1; j <= col+1; j++{

					if (i==row && j==col) {
						continue
					}

					if world[((i+p.ImageWidth)%p.ImageWidth)] [(j+p.ImageHeight)%p.ImageHeight] == 255 {
						alivemeter++


					}
				}
			}

			// game of life conditions
			if (alivemeter<2 || alivemeter>3){
				testerworld[row][col] = 0
			}
			if alivemeter==3 {
				testerworld[row][col]= 255
			}
		}
	}

	return testerworld
}


//inner for loop calculates the state of the neighbours

//func CalcNeighbourState(p Params, world [][]byte)

func calculateAliveCells(p Params, world [][]byte) []util.Cell {

	var alivecells []util.Cell

	for row := 0; row < p.ImageWidth; row++ {
		for col := 0; col < p.ImageHeight; col++ {

			pair := util.Cell{}
			currentCell := world[row][col]

			if (currentCell == 255){
				pair.X = col
				pair.Y = row
				alivecells = append(alivecells, pair)

			}
		}
	}



	return alivecells
}*/

func makeCall(client rpc.Client, message string) {
	request := util.Request{Message: message}
	response := new(util.Response)
	client.Call(util.ReverseHandler, request, response)
	fmt.Println("Responded: " + response.Message)
}

func main() {
	server := flag.String("server", "54.226.128.78:8030", "IP:port string to connect to as server")
	flag.Parse()
	client, _ := rpc.Dial("tcp", *server)
	defer client.Close()

	file, _ := os.Open("wordlist")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		t := scanner.Text()
		fmt.Println("Called: " + t)
		makeCall(*client, t)
	}

}
