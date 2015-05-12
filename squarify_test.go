package squarify

import (
	"testing"
)

func TestRowPushTemporarily(t *testing.T) {
	r := newRow(Vertical, 20, 30, 30)

	r.push(&area{40, nil})

	if len(r.areas) != 1 {
		t.Fatal("Push failed")
	}

	r.pushTemporarily(&area{50, nil}, func() {
		if len(r.areas) != 2 {
			t.Fatal("Temporary Push failed")
		}

		if r.areas[0].Area != 40 {
			t.Fatal("Temporary Push was wrong")
		}

		if r.areas[1].Area != 50 {
			t.Fatal("Temporary Push was wrong")
		}
	})

	if len(r.areas) != 1 {
		t.Fatal("Temporary Push didn't clean up")
	}

	if r.areas[0].Area != 40 {
		t.Fatal("Temporary Push was wrong")
	}

	if r.min != 40 {
		t.Fatal("Temporary Push was wrong")
	}

	if r.max != 40 {
		t.Fatal("Temporary Push was wrong")
	}

	if r.sum != 40 {
		t.Fatal("Temporary Push was wrong")
	}
}

type TestNode struct {
	name     string
	children []*TestNode
	size     float64
}

func (t TestNode) Size() float64 {
	return t.size
}

func (t TestNode) NumChildren() int {
	return len(t.children)
}

func (t TestNode) Child(i int) TreeSizer {
	return t.children[i]
}

func TestSquarifyAreas(t *testing.T) {
	// func squarify(root TreeSizer, block Block, maxDepth int, margins *Margins, sort bool, depth int) (blocks []Block, meta []Meta) {

	// root -> size 30 + 20 local files
	//   b  -> size 10
	//   c  -> size 20

	nodes := map[string]*TestNode{}
	addToMap := func(t *TestNode) *TestNode {
		nodes[t.name] = t
		return t
	}

	b := addToMap(&TestNode{name: "b", size: 10})
	c := addToMap(&TestNode{name: "c", size: 20})
	root := TestNode{name: "root", children: []*TestNode{b, c}, size: 80}

	canvas := Rect{X: 0, Y: 0, W: 100, H: 100}

	validateSize := func(tn TestNode, b Block) {
		expectedArea := tn.size / root.size
		actualArea := (b.W * b.H) / (canvas.W * canvas.H)

		if expectedArea != actualArea {
			t.Fatal("Bad area for", tn.name, ": expected", expectedArea, "got", actualArea)
		}
	}

	options := Options{Sort: DoSort}
	blocks, _ := Squarify(root, canvas, options)

	// Ensure the computed areas are correct
	for _, blk := range blocks {
		if blk.TreeSizer != nil {
			n, ok := nodes[blk.TreeSizer.(*TestNode).name]
			if !ok {
				t.Fatal("Squarify produced a block with no matching TestNode")
			}

			validateSize(*n, blk)
		}
	}

	// Ensure that the blocks are sorted largest to smallest
	lastArea := float64(-1)
	for _, blk := range blocks {
		area := blk.W * blk.H
		if lastArea >= 0 {
			if area > lastArea {
				t.Fatal("Areas are not sorted descending")
			}
		}
		lastArea = area
	}
}

func TestMaxDepth(t *testing.T) {
	// func squarify(root TreeSizer, block Block, maxDepth int, margins *Margins, sort bool, depth int) (blocks []Block, meta []Meta) {

	// root   -> size 50 + 20 local files
	//   b    -> size 10
	//    b.1 -> size 10
	//   c    -> size 40
	//    c.1 -> size 20
	//    c.2 -> size 10

	nodes := map[string]*TestNode{}
	addToMap := func(t *TestNode) *TestNode {
		nodes[t.name] = t
		return t
	}

	b1 := addToMap(&TestNode{name: "b1", size: 10})

	b := addToMap(&TestNode{name: "b", children: []*TestNode{b1}, size: 10})

	c1 := addToMap(&TestNode{name: "c1", size: 20})
	c2 := addToMap(&TestNode{name: "c2", size: 10})

	c := addToMap(&TestNode{name: "c", children: []*TestNode{c1, c2}, size: 40})
	root := TestNode{name: "root", children: []*TestNode{b, c}, size: 70}

	canvas := Rect{X: 0, Y: 0, W: 100, H: 100}

	options := Options{MaxDepth: 1, Sort: DoSort}
	blocks, _ := Squarify(root, canvas, options)

	if len(blocks) != 2 {
		t.Fatal("Squarify produced", len(blocks), "blocks when 2 were expected")
	}

}
