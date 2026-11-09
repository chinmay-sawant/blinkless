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
	return OperationOwnersFromParentsBuf(res, parents, nil)
}

// OperationOwnersFromParentsBuf calculates operation owners using element parent
// indices into the provided destination buffer, allocating or growing only if
// cap(buf) < len(res.Ops).
func OperationOwnersFromParentsBuf(res *Result, parents []int, buf []int) []int {
	if res == nil {
		return nil
	}

	n := len(res.Ops)
	if cap(buf) < n {
		buf = make([]int, n)
	} else {
		buf = buf[:n]
	}

	for index := range buf {
		buf[index] = -1
	}

	placedIndex := 0
	walkOperationOwnersParents(res.root, -1, &placedIndex, buf, parents)

	return buf
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

// spliceOwnerWindow splices the owners slice over one committed op window:
// the head stays, the window is recomputed from the new box subtree, and
// the tail shifts into place with owner values after the old window moved by
// the element delta. The window recompute seeds the deepest visible element
// ancestor and overwrites in document order, so the deepest covering element
// wins, the rule the full walk applies: overlapping ranges nest, and a later
// writer in document order is always a descendant-or-self of the earlier
// ones covering the same operation.
func spliceOwnerWindow(
	owners []int, start, oldCount, newCount, seed, elemBase, oldElemCount int, newBox *box,
) []int {
	oldLen := len(owners)
	newLen := oldLen + newCount - oldCount

	if newCount > oldCount {
		owners = append(owners, make([]int, newCount-oldCount)...)
	}

	copy(owners[start+newCount:newLen], owners[start+oldCount:oldLen])
	owners = owners[:newLen]

	newElemCount := writeOwnerWindow(owners, start, newCount, seed, elemBase, newBox)
	shift := newElemCount - oldElemCount
	threshold := elemBase + oldElemCount

	for index := start + newCount; index < newLen; index++ {
		if owners[index] >= threshold {
			owners[index] += shift
		}
	}

	return owners
}

// writeOwnerWindow recomputes the owners of the spliced window from the new
// box subtree and returns the element count it holds. The seed covers gaps
// no subtree box paints; every element box in document order overwrites its
// range intersected with the window. It mirrors walkOperationOwnersParents
// over children only: table cells never own operations.
func writeOwnerWindow(owners []int, start, newCount, seed, elemBase int, newBox *box) int {
	end := start + newCount
	if end > len(owners) {
		end = len(owners)
	}

	if start < 0 {
		start = 0
	}

	for index := start; index < end; index++ {
		owners[index] = seed
	}

	local := 0

	var walk func(node *box)

	walk = func(node *box) {
		if node == nil {
			return
		}

		if node.node != nil && node.node.Type == html.ElementNode {
			writeOwnerRange(owners, node.opStart, node.opEnd, start, end, elemBase+local)

			local++
		}

		for _, child := range node.children {
			walk(child)
		}
	}

	walk(newBox)

	return local
}

// writeOwnerRange overwrites owners with owner over the intersection of
// [opStart, opEnd] and the spliced window [start, end). It mirrors the
// walkOperationOwnersParents rule that table cells never own operations.
func writeOwnerRange(owners []int, opStart, opEnd, start, end, owner int) {
	if opStart > opEnd {
		return
	}

	from := opStart
	if from < start {
		from = start
	}

	to := opEnd
	if to >= end {
		to = end - 1
	}

	for index := from; index <= to; index++ {
		owners[index] = owner
	}
}
