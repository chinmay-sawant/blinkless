package layout

import "github.com/chinmay-sawant/blinkless/internal/html"

// OperationOwners returns the nearest common element owner of each operation.
// Shared operations whose box ranges cross sibling elements belong to their
// nearest common element ancestor.
func OperationOwners(res *Result) []int {
	if res == nil {
		return nil
	}

	return OperationOwnersPlaced(res, PlacedElements(res))
}

// OperationOwnersPlaced calculates operation owners using an already computed
// slice of PlacedElements, avoiding a redundant tree walk.
func OperationOwnersPlaced(res *Result, placed []PlacedElement) []int {
	if res == nil {
		return nil
	}

	parents := make([]int, len(placed))
	for i := range placed {
		parents[i] = placed[i].ParentIndex
	}

	return OperationOwnersFromParents(res, parents)
}

// OperationOwnersFromParents calculates operation owners using element parent indices.
func OperationOwnersFromParents(res *Result, parents []int) []int {
	if res == nil {
		return nil
	}

	owners := make([]int, len(res.Ops))
	for index := range owners {
		owners[index] = -1
	}

	placedIndex := 0
	walkOperationOwnersParents(res.root, -1, &placedIndex, owners, parents)

	return owners
}

func walkOperationOwnersParents(boxNode *box, parent int, placedIndex *int, owners []int, parents []int) {
	if boxNode == nil {
		return
	}

	owner := parent
	if boxNode.node != nil && boxNode.node.Type == html.ElementNode {
		owner = *placedIndex
		(*placedIndex)++

		if owner >= 0 && boxNode.opStart >= 0 && boxNode.opEnd >= boxNode.opStart {
			for index := boxNode.opStart; index <= boxNode.opEnd && index < len(owners); index++ {
				owners[index] = closestOwnerParents(owners[index], owner, parents)
			}
		}
	}

	for _, child := range boxNode.children {
		walkOperationOwnersParents(child, owner, placedIndex, owners, parents)
	}
}

func closestOwnerParents(current, candidate int, parents []int) int {
	if current < 0 {
		return candidate
	}

	if elementAncestorParents(current, candidate, parents) {
		return candidate
	}

	if elementAncestorParents(candidate, current, parents) {
		return current
	}

	for ancestor := parents[current]; ancestor >= 0 && ancestor < len(parents); ancestor = parents[ancestor] {
		if elementAncestorParents(ancestor, candidate, parents) {
			return ancestor
		}
	}

	return -1
}

func elementAncestorParents(ancestor, child int, parents []int) bool {
	for child >= 0 && child < len(parents) {
		if child == ancestor {
			return true
		}

		child = parents[child]
	}

	return false
}
