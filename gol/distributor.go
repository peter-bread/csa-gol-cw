package gol

import (
	"fmt"
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
	filename := fmt.Sprintf("%vx%v", p.ImageWidth, p.ImageHeight)
	c.ioCommand <- ioInput
	c.ioFilename <- filename

	// Create a 2D slice to store the world.
	world := make([][]byte, p.ImageHeight)
	for i := range world {
		world[i] = make([]byte, p.ImageWidth)
	}

	for i := 0; i < p.ImageHeight; i++ {
		for j := 0; j < p.ImageWidth; j++ {
			world[i][j] = <-c.ioInput
		}
	}

	turn := 0

	// initialise worker channels
	workers := make([]chan [][]byte, p.Threads)
	for i := 0; i < p.Threads; i++ {
		workers[i] = make(chan [][]byte)
	}

	// split heights as evenly as possible
	heights := calcHeights(p.ImageHeight, p.Threads)

	// create ticker that ticks every 2 seconds
	ticker := time.NewTicker(2 * time.Second)

	// start ticker and send AliveCellsCount events
	go func() {
		for {
			select {
			case <-ticker.C:
				c.events <- AliveCellsCount{
					CompletedTurns: turn,
					CellsCount:     len(calculateAliveCells(p, world)),
				}
			}
		}
	}()

	// Execute all turns of the Game of Life.
	for ; turn < p.Turns; turn++ {

		startY := 0

		// start workers
		for i := 0; i < p.Threads; i++ {
			go worker(startY, startY+heights[i], 0, p.ImageWidth, p.ImageHeight, p.ImageWidth, world, workers[i])
			startY += heights[i]
		}

		var newWorld [][]byte

		// reassemble world
		for i := 0; i < p.Threads; i++ {
			newWorld = append(newWorld, <-workers[i]...)
		}

		// replace world with new world
		copy(world, newWorld)

		// send TurnComplete event after each turn
		c.events <- TurnComplete{
			CompletedTurns: turn + 1,
		}

	}

	// stop ticker after all turns executed
	ticker.Stop()

	// Report the final state using FinalTurnCompleteEvent.
	alive := calculateAliveCells(p, world)
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          alive,
	}

	// after all turns send state of board to be outputted as a .pgm image
	filename = fmt.Sprintf("%vx%vx%v", p.ImageWidth, p.ImageHeight, p.Turns)
	c.ioCommand <- ioOutput
	c.ioFilename <- filename

	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	// send StateChange event to announce GoL is ended
	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calcHeights(imageHeight, threads int) []int {
	baseHeight := imageHeight / threads
	remainder := imageHeight % threads
	heights := make([]int, threads)

	for i := 0; i < threads; i++ {
		if remainder > 0 { // distribute the remainder as evenly as possible
			heights[i] = baseHeight + 1
			remainder--
		} else {
			heights[i] = baseHeight
		}
	}
	return heights
}

func worker(startY, endY, startX, endX, world_height, world_width int, world [][]byte, out chan<- [][]byte) {
	out <- calculateNextState(startY, endY, startX, endX, world_height, world_width, world)
}

func calculateNextState(startY, endY, startX, endX, world_height, world_width int, world [][]byte) [][]byte {
	//   world[ row ][ col ]
	//      up/down   left/right

	height := endY - startY
	width := endX - startX

	newWorld := make([][]byte, height)
	for i := range newWorld {
		newWorld[i] = make([]byte, width)
	}

	for rowI, row := range world[startY:endY] { // for each row of the grid
		for colI, cellVal := range row { // for each cell in the row
			aliveNeighbours := 0 // initially there are 0 living neighbours

			// iterate through neighbours
			for i := -1; i < 2; i++ {
				for j := -1; j < 2; j++ {

					// if cell is a neighbour (i.e. not the cell having its neighbours checked)
					if i != 0 || j != 0 {

						// Calculate neighbour coordinates with wrapping
						neighbourRow := (rowI + i + startY + world_height) % world_height
						neighbourCol := (colI + j + world_width) % world_width

						// Check if the wrapped neighbour is alive
						if world[neighbourRow][neighbourCol] == 255 {
							aliveNeighbours++
						}
					}
				}
			}

			// implement rules
			if cellVal == 255 && aliveNeighbours < 2 { // cell is lonely and dies
				newWorld[rowI][colI] = 0
			} else if cellVal == 255 && aliveNeighbours > 3 { // cell killed by overpopulation
				newWorld[rowI][colI] = 0
			} else if cellVal == 0 && aliveNeighbours == 3 { // new cell is born
				newWorld[rowI][colI] = 255
			} else { // cell remains as it is
				newWorld[rowI][colI] = world[rowI+startY][colI+startX]
			}
		}
	}
	return newWorld
}

func calculateAliveCells(p Params, world [][]byte) []util.Cell {
	aliveCells := make([]util.Cell, 0, p.ImageHeight*p.ImageWidth)
	for rowI, row := range world {
		for colI, cellVal := range row {
			if cellVal == 255 {
				aliveCells = append(aliveCells, util.Cell{X: colI, Y: rowI})
			}
		}
	}
	return aliveCells
}
