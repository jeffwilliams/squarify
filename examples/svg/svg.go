package main

import (
	"fmt"
	"github.com/jeffwilliams/squarify"
	"io"
	"os"
)

func main() {

	// Options modifying Squarify behaviour.
	opt := squarify.Options{
		// Left, Right, and Bottom margin of 5, and a Top margin of 30.
		Margins: &squarify.Margins{5, 5, 30, 5},
		// Sort biggest to smallest
		Sort: true,
	}

	// Define the size of the output: 800x800
	canvas := squarify.Rect{W: 800, H: 800}

	// Run the Squarify function on our sample tree to get a slice of Blocks we can render,
	// and metadata for each block
	blocks, meta := squarify.Squarify(sampleTree(), canvas, opt)

	// Output the Blocks as an SVG image to stdout (redirect to a file to save)
	makeSvg(blocks, meta, canvas, os.Stdout)
}

// Our implementation of the TreeMap.
type TreeMap struct {
	children []TreeMap
	size     float64
	label    string
}

// Size implements a required method of the TreeSizer interface needed by Squarify.
func (t TreeMap) Size() float64 {
	return t.size
}

// NumChildren implements a required method of the TreeSizer interface needed by Squarify.
func (t TreeMap) NumChildren() int {
	return len(t.children)
}

// Child implements a required method of the TreeSizer interface needed by Squarify.
func (t TreeMap) Child(i int) squarify.TreeSizer {
	return squarify.TreeSizer(t.children[i])
}

// setSize sets the size of the TreeMap node to the sum of the sizes of it's children plus the passed size.
// This is a helper function for sampleTree below.
func (t *TreeMap) setSize(size float64) {
	t.size = size
	for _, c := range t.children {
		t.size += c.size
	}
}

// sampleTree builds the TreeMap we will output.
func sampleTree() TreeMap {
	a := TreeMap{label: "a (40)", size: 40}
	b := TreeMap{label: "b (30)", size: 30}
	c := TreeMap{label: "c (10)", children: []TreeMap{a, b}}
	c.setSize(10)

	d := TreeMap{label: "d (10)", size: 10}
	e := TreeMap{label: "e (20)", size: 20}
	f := TreeMap{label: "f (15)", size: 15}
	g := TreeMap{label: "g (15)", size: 15}
	h := TreeMap{label: "h (15)", size: 15}
	i := TreeMap{label: "i (30)", size: 30}
	j := TreeMap{label: "j (0)", children: []TreeMap{d, e, f, g, h, i}}
	j.setSize(0)

	k := TreeMap{label: "k (10)", size: 10}
	l := TreeMap{label: "l (20)", size: 20}
	m := TreeMap{label: "m (12)", size: 15}
	n := TreeMap{label: "n (16)", size: 15}
	o := TreeMap{label: "o (15)", size: 15}
	p := TreeMap{label: "p (15)", children: []TreeMap{k, l, m, n, o}}
	p.setSize(15)

	root := TreeMap{label: "root (0)", children: []TreeMap{c, j, p}}
	root.setSize(0)

	return root
}

// colors we will use to fill the Blocks at different depths.
var colors []string = []string{"588C7E", "F2E394", "F2AE72", "D96459", "8C4646"}

func color(depth int) string {
	return colors[(depth+2)%len(colors)]
}

// makeSvg outputs an SVG image from the passed blocks and meta. The size of the image is the size of canvas,
// and the SVG is output to the writer w.
func makeSvg(blocks []squarify.Block, meta []squarify.Meta, canvas squarify.Rect, w io.Writer) {
	fmt.Fprintf(w, "<svg xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\" width=\"%f\" height=\"%f\">\n",
		canvas.W, canvas.H)
	for i, b := range blocks {
		fmt.Fprintf(w, "  <rect x=\"%f\" y=\"%f\" width=\"%f\" height=\"%f\" style=\"fill: #%s;stroke-width: 1;stroke: #000000; font-family: verdana, bitstream vera sans, sans\"/>\n",
			b.X, b.Y, b.W, b.H, color(meta[i].Depth))

		if b.TreeSizer != nil {
			label := b.TreeSizer.(TreeMap).label
			fmt.Fprintf(w, "  <text x=\"%f\" y=\"%f\" style=\"font-size:20px\">%s</text>\n", b.X+5, b.Y+20, label)
		}
	}
	fmt.Fprintf(w, "</svg>\n")
}
