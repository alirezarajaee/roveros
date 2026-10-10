package main

import (
	"fmt"
	"os"

	"github.com/alirezarajaee/roveros/internal/rover"
	"github.com/alirezarajaee/roveros/internal/simulation"
	"github.com/alirezarajaee/roveros/internal/world"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "ROVEROS error:", err)
		os.Exit(1)
	}
}

func run() error {
	grid, err := world.NewGrid(20, 10)
	if err != nil {
		return err
	}

	// Create a wall with an opening at the bottom.
	for y := 0; y < 8; y++ {
		if err := grid.SetBlocked(world.Point{X: 8, Y: y}, true); err != nil {
			return err
		}
	}

	// Add a few additional rocks to the terrain.
	obstacles := []world.Point{
		{X: 4, Y: 2},
		{X: 4, Y: 3},
		{X: 4, Y: 4},
		{X: 12, Y: 4},
		{X: 12, Y: 5},
	}

	for _, obstacle := range obstacles {
		if err := grid.SetBlocked(obstacle, true); err != nil {
			return err
		}
	}

	start := world.Point{X: 1, Y: 1}
	goal := world.Point{X: 15, Y: 7}

	initialRover, err := rover.New(start, 100)
	if err != nil {
		return err
	}

	engine, err := simulation.New(grid, initialRover)
	if err != nil {
		return err
	}

	if err := engine.Plan(goal); err != nil {
		return fmt.Errorf("plan mission: %w", err)
	}

	initialState := engine.Snapshot()

	fmt.Println("====================================")
	fmt.Println("  ROVEROS | MISSION CONTROL")
	fmt.Println("  Autonomous Rover Simulation")
	fmt.Println("====================================")
	fmt.Printf("Start: (%d, %d)\n", start.X, start.Y)
	fmt.Printf("Goal:  (%d, %d)\n", goal.X, goal.Y)
	fmt.Printf("Planned moves: %d\n\n", len(initialState.Path)-1)

	fmt.Println("Initial terrain and planned route:")
	printMap(grid, initialState)

	fmt.Println("\nExecuting mission...")

	for {
		current := engine.Snapshot()
		if current.Status == simulation.StatusCompleted {
			break
		}

		state, err := engine.Step()
		if err != nil {
			return fmt.Errorf("mission failed: %w", err)
		}

		fmt.Printf(
			"Step %02d | Position: (%d,%d) | Battery: %.1f%% | Status: %s\n",
			state.Steps,
			state.Rover.Position.X,
			state.Rover.Position.Y,
			state.Rover.BatteryPercent,
			state.Status,
		)
	}

	finalState := engine.Snapshot()

	fmt.Println("\nMission completed successfully.")
	fmt.Printf("Final position: (%d, %d)\n",
		finalState.Rover.Position.X,
		finalState.Rover.Position.Y,
	)
	fmt.Printf("Remaining battery: %.1f%%\n", finalState.Rover.BatteryPercent)
	fmt.Printf("Total moves: %d\n", finalState.Steps)

	fmt.Println("\nFinal terrain:")
	printMap(grid, finalState)

	return nil
}

func printMap(grid *world.Grid, state simulation.Snapshot) {
	pathCells := make(map[world.Point]bool, len(state.Path))
	for _, point := range state.Path {
		pathCells[point] = true
	}

	for y := 0; y < grid.Height(); y++ {
		row := make([]rune, grid.Width())

		for x := 0; x < grid.Width(); x++ {
			point := world.Point{X: x, Y: y}

			switch {
			case point == state.Rover.Position &&
				state.HasTarget &&
				point == state.Target:
				row[x] = 'X'
			case point == state.Rover.Position:
				row[x] = 'R'
			case state.HasTarget && point == state.Target:
				row[x] = 'G'
			case !grid.IsWalkable(point):
				row[x] = '#'
			case pathCells[point]:
				row[x] = '*'
			default:
				row[x] = '.'
			}
		}

		fmt.Println(string(row))
	}

	fmt.Println("R = Rover | G = Goal | X = Rover at goal")
	fmt.Println("# = Obstacle | * = Planned route | . = Free terrain")
}
