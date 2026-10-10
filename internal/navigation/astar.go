package navigation

import (
	"container/heap"
	"errors"
	"fmt"

	"github.com/alirezarajaee/roveros/internal/world"
)

var (
	ErrNilGrid          = errors.New("grid is nil")
	ErrStartOutOfBounds = errors.New("start point is outside the grid")
	ErrGoalOutOfBounds  = errors.New("goal point is outside the grid")
	ErrStartBlocked     = errors.New("start point is blocked")
	ErrGoalBlocked      = errors.New("goal point is blocked")
	ErrNoPath           = errors.New("no path exists")
)

type pathNode struct {
	point  world.Point
	gScore int
	fScore int
}

type priorityQueue []*pathNode

func (pq priorityQueue) Len() int {
	return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
	if pq[i].fScore != pq[j].fScore {
		return pq[i].fScore < pq[j].fScore
	}

	if pq[i].point.Y != pq[j].point.Y {
		return pq[i].point.Y < pq[j].point.Y
	}

	return pq[i].point.X < pq[j].point.X
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *priorityQueue) Push(value any) {
	node := value.(*pathNode)
	*pq = append(*pq, node)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	last := len(old) - 1
	node := old[last]

	old[last] = nil
	*pq = old[:last]

	return node
}

// FindPath returns a shortest four-directional path from start to goal.
// The returned path includes both start and goal.
func FindPath(
	grid *world.Grid,
	start world.Point,
	goal world.Point,
) ([]world.Point, error) {
	if grid == nil {
		return nil, ErrNilGrid
	}

	if !grid.InBounds(start) {
		return nil, fmt.Errorf(
			"%w: (%d, %d)",
			ErrStartOutOfBounds,
			start.X,
			start.Y,
		)
	}

	if !grid.InBounds(goal) {
		return nil, fmt.Errorf(
			"%w: (%d, %d)",
			ErrGoalOutOfBounds,
			goal.X,
			goal.Y,
		)
	}

	if !grid.IsWalkable(start) {
		return nil, fmt.Errorf(
			"%w: (%d, %d)",
			ErrStartBlocked,
			start.X,
			start.Y,
		)
	}

	if !grid.IsWalkable(goal) {
		return nil, fmt.Errorf(
			"%w: (%d, %d)",
			ErrGoalBlocked,
			goal.X,
			goal.Y,
		)
	}

	if start == goal {
		return []world.Point{start}, nil
	}

	open := &priorityQueue{}
	heap.Init(open)

	gScore := map[world.Point]int{
		start: 0,
	}

	cameFrom := make(map[world.Point]world.Point)
	closed := make(map[world.Point]bool)

	heap.Push(open, &pathNode{
		point:  start,
		gScore: 0,
		fScore: heuristic(start, goal),
	})

	for open.Len() > 0 {
		current := heap.Pop(open).(*pathNode)

		bestScore, known := gScore[current.point]
		if !known || current.gScore != bestScore {
			continue
		}

		if closed[current.point] {
			continue
		}

		closed[current.point] = true

		if current.point == goal {
			return reconstructPath(cameFrom, goal), nil
		}

		for _, next := range adjacentPoints(current.point) {
			if !grid.IsWalkable(next) || closed[next] {
				continue
			}

			tentativeGScore := current.gScore + 1

			previousScore, exists := gScore[next]
			if exists && tentativeGScore >= previousScore {
				continue
			}

			cameFrom[next] = current.point
			gScore[next] = tentativeGScore

			heap.Push(open, &pathNode{
				point:  next,
				gScore: tentativeGScore,
				fScore: tentativeGScore + heuristic(next, goal),
			})
		}
	}

	return nil, ErrNoPath
}

func heuristic(a, b world.Point) int {
	dx := a.X - b.X
	if dx < 0 {
		dx = -dx
	}

	dy := a.Y - b.Y
	if dy < 0 {
		dy = -dy
	}

	return dx + dy
}

func adjacentPoints(point world.Point) [4]world.Point {
	return [4]world.Point{
		{X: point.X, Y: point.Y - 1},
		{X: point.X - 1, Y: point.Y},
		{X: point.X + 1, Y: point.Y},
		{X: point.X, Y: point.Y + 1},
	}
}

func reconstructPath(
	cameFrom map[world.Point]world.Point,
	current world.Point,
) []world.Point {
	path := []world.Point{current}

	for {
		previous, exists := cameFrom[current]
		if !exists {
			break
		}

		current = previous
		path = append(path, current)
	}

	for left, right := 0, len(path)-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}

	return path
}
