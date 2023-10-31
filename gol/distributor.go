package gol

import (
	"fmt"

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

	// TODO: Create a 2D slice to store the world.
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

	// TODO: Execute all turns of the Game of Life.
	for ; turn < p.Turns; turn++ {
		// start worker goroutines here
		world = calculateNextState(p, world)
	}

	// TODO: Report the final state using FinalTurnCompleteEvent.
	alive := calculateAliveCells(p, world)
	c.events <- FinalTurnComplete{
		CompletedTurns: turn,
		Alive:          alive,
	}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{turn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

func calculateNextState(p Params, world [][]byte) [][]byte {
	//   world[ row ][ col ]
	//      up/down    left/right

	newWorld := make([][]byte, p.ImageHeight)
	for i := range newWorld {
		newWorld[i] = make([]byte, p.ImageWidth)
	}

	for rowI, row := range world { // for each row of the grid
		for colI, cellVal := range row { // for each cell in the row

			aliveNeighbours := 0 // initially there are 0 living neighbours

			// iterate through neighbours
			for i := -1; i < 2; i++ {
				for j := -1; j < 2; j++ {

					// if cell is a neighbour (i.e. not the cell having its neighbours checked)
					if i != 0 || j != 0 {

						// Calculate neighbour coordinates with wrapping
						neighbourRow := (rowI + i + p.ImageHeight) % p.ImageHeight
						neighbourCol := (colI + j + p.ImageWidth) % p.ImageWidth

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
				newWorld[rowI][colI] = world[rowI][colI]
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
