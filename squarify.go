// Package squarify implements the Squarified Treemap algorithm of Bruls, Huizing, and Van Wijk:
//
//    http://www.win.tue.nl/~vanwijk/stm.pdf
//
// The basic idea is to generate a tiling of items of various sizes, each of which may have children which
// are tiled nested inside their parent.
//
// Tiling is performed by calling the Squarify() function.
package squarify

import (
	"sort"
)

// The TreeSizer interface must be implemented by any tree that will be Squarified.
type TreeSizer interface {
	// Size is the logical size of the item and will be used to compute it's area.
	Size() float64
	// NumChildren is the number of children that this tree node has.
	NumChildren() int
	// Child returns the child of this node at the specified index.
	Child(i int) TreeSizer
}

// Rect represents a two-dimensional rectangle having the upper-left corner at coordinate (X,Y) and having
// width W and height H.
type Rect struct {
	X, Y, W, H float64
}

// Block represents the output for a single TreeSizer node from Squarify(). It's the layed-out
// position and size of the TreeSizer node and the associated TreeSizer that it represents.
type Block struct {
	// Position and size of the layed-out TreeSizer
	Rect
	// The TreeSizer represented by this Block
	TreeSizer TreeSizer
}

// area is the product of an intermediate step that stores the computed area for a
// TreeSizer
type area struct {
	Area      float64
	TreeSizer TreeSizer
}

// direction represents the direction of layout for a row: Vertical or Horizontal.
type direction int

const (
	// Vertical lays out the row vertically
	Vertical direction = iota
	// Horizontal lays out the row horizontally
	Horizontal
)

const (
  // DoSort may be used in the Options struct to request sorting
	DoSort   = true
  // DontSort may be used in the Options struct to decline sorting
	DontSort = false
)

// Margins defines the empty space that should exist between a parent Block and it's internal
// children Blocks when layed out. Similar to how the margins of a page separate the content from
// the page edges. The left (L), right (R), top (T) and bottom (B) margins are specified separately.
type Margins struct {
	L, R, T, B float64
}

// Meta is metadata that is returned by Squarify() for each Block it lays out.
type Meta struct {
	// Depth is the depth of the Block in the TreeSizer tree, starting from 0.
	// This is useful when coloring the blocks, for example.
	Depth int
}

// Options controls how Squarify() behaves.
type Options struct {
	// Maximum depth in the tree to descend to. Blocks at depth <= MaxDepth are layed out,
	// and > MaxDepth are ignored. Depth is counted starting at the children of the root node being 1.
	// If MaxDepth is left as the zero value then a MaxDepth of 20 is used.
	MaxDepth int
	// Margins between a parent and it's children.
	Margins *Margins
	// Sort the blocks by size within their parent. This pushes larger blocks to the left/above
	// smaller blocks.
	Sort bool
	// MinW and MinH limit the smallness of Blocks that are output. Blocks who's width is < MinW
	// or who's height is < MinH are not output, nor are their children processed.
	MinW, MinH float64
}

// Squarify implements the Squarified Treemap algorithm. It lays out the children of root inside the area
// represented by rect with areas proportional to the Size() of the children. Squarify returns
// a slice of Blocks for rendering and a slice of Metas which contain metadata corresponding
// to the Blocks. Element block[i] has metadata in meta[i].
//
func Squarify(root TreeSizer, rect Rect, options Options) (blocks []Block, meta []Meta) {
	if options.MaxDepth <= 0 {
		options.MaxDepth = 20
	}

	return squarify(root, Block{Rect: rect}, options, 0)
}

// row is an internal structure used to represent the current row or column of blocks
// being layed out.
type row struct {
	areas    []*area
	X, Y     float64
	min, max float64 // Min and max areas in the row
	sum      float64 // Sum of areas
	Width    float64
	Dir      direction
}

// newRow returns a new row layed out in the specified direction.
func newRow(dir direction, width, x, y float64) *row {
	return &row{
		areas: make([]*area, 0),
		Width: width,
		X:     x,
		Y:     y,
		Dir:   dir,
	}
}

// push adds an area to the end of the row.
func (r *row) push(a *area) {
	if a.Area <= 0 {
		// We use 0 area as a sentinel in min and max.
		panic("Area must be >= 0")
	}

	r.areas = append(r.areas, a)
	r.updateCached(a)
}

// pop removes and returns the area at the end of the row.
func (r *row) pop() *area {
	r.min = 0
	r.max = 0
	r.sum = 0

	if len(r.areas) > 0 {
		last := len(r.areas) - 1
		a := r.areas[last]
		r.areas[last] = nil
		r.areas = r.areas[0:last]
		return a
	}

  return nil
}

// pushTemporarily pushes area `a` onto the Row, runs f(), then pops the Row.
// The Row is the same before and after this call.
func (r *row) pushTemporarily(a *area, f func()) {
	min := r.min
	max := r.max
	sum := r.sum
	r.push(a)
	f()
	r.pop()
	r.min = min
	r.max = max
	r.sum = sum
}

// calcCached calculates cached values for the row.
func (r *row) calcCached() {
	r.min = 0
	r.max = 0
	r.sum = 0
	for _, a := range r.areas {
		r.updateCached(a)
	}
}

// size returns the number of elements in the row
func (r row) size() int {
	return len(r.areas)
}

// updateCached updates the cached row values with `a` added to the row.
func (r *row) updateCached(a *area) {
	if r.min <= 0 || a.Area < r.min {
		r.min = a.Area
	}
	if r.max <= 0 || a.Area > r.max {
		r.max = a.Area
	}
	r.sum += a.Area
}

// Calculate the worst aspect ratio of all rectangles in the row
func (r *row) worst() float64 {
	if r.min == 0 {
		// We need to calculate min, max, and sum
		r.calcCached()
	}

	w2 := r.Width * r.Width
	sum2 := r.sum * r.sum
	worst1 := w2 * r.max / sum2
	worst2 := sum2 / (r.min * w2)

	if worst1 > worst2 {
		return worst1
	}

	return worst2
}

// makeBlocks creates the final slice of blocks for the row.
func (r *row) makeBlocks() (height float64, blocks []Block) {
	if r.min == 0 {
		// We need to calculate min, max, and sum
		r.calcCached()
	}

	blocks = make([]Block, 0)
	x := r.X
	y := r.Y

	for _, a := range r.areas {
		// Item width relative to the row
		relativeWidth := a.Area / r.sum
		itemWidth := relativeWidth * r.Width
		itemHeight := a.Area / itemWidth

		if height == 0 {
			height = itemHeight
		} else if itemHeight != height {
			itemHeight = height
		}

		if r.Dir == Vertical {
			// swap
			itemWidth, itemHeight = itemHeight, itemWidth
		}

		blocks = append(blocks, Block{Rect: Rect{X: x, Y: y, W: itemWidth, H: itemHeight}, TreeSizer: a.TreeSizer})

		if r.Dir == Vertical {
			y += itemHeight
		} else {
			x += itemWidth
		}
	}

	return
}

// Internal squarify function. Squarify is a frontend to this.
func squarify(root TreeSizer, block Block, options Options, depth int) (blocks []Block, meta []Meta) {
	blocks = make([]Block, 0)
	meta = make([]Meta, 0)

	if block.W <= options.MinW || block.H <= options.MinH || depth >= options.MaxDepth {
		return
	}

	output := func(newBlocks []Block) {
		for i := 0; i < len(newBlocks); i++ {
			// Filter out any blocks that are just placeholders for extra space
			if newBlocks[i].TreeSizer != nil {
				// Filter out any blocks that are too small
				if newBlocks[i].W > options.MinW || newBlocks[i].H > options.MinH {
					blocks = append(blocks, newBlocks[i])
					meta = append(meta, Meta{Depth: depth})
				}
			}
		}
	}

	areas := areas(root, block, options.Sort)

	rowX := block.X
	rowY := block.Y
	freeWidth := block.W
	freeHeight := block.H

	makeRow := func() (row *row) {
		if block.W > block.H {
			row = newRow(Vertical, freeHeight, rowX, rowY)
		} else {
			row = newRow(Horizontal, freeWidth, rowX, rowY)
		}
		return row
	}

	// Decide which direction to create the new row
	row := makeRow()

	for _, a := range areas {
		if row.size() > 0 {
			worstBefore := row.worst()
			worstAfter := float64(0)
			row.pushTemporarily(&a, func() {
				worstAfter = row.worst()
			})

			if worstBefore < worstAfter {
				// It's better to make a new row now.
				// Output the current blocks and make a new row
				offset, newBlocks := row.makeBlocks()
				output(newBlocks)

				if row.Dir == Vertical {
					rowX += offset
					freeWidth -= offset
				} else {
					rowY += offset
					freeHeight -= offset
				}

				row = makeRow()
			}
		}

		cp := &area{}
		*cp = a
		row.push(cp)
	}

	if row.size() > 0 {
		_, newBlocks := row.makeBlocks()
		output(newBlocks)
	}

	// Now, for each of the items we just processed, if they have children then
	// lay them out inside their parent box. The available area may be reduced by
	// certain size.
	for _, block := range blocks {
		if block.TreeSizer != nil {
			if options.Margins != nil {
				block.X += options.Margins.L
				block.Y += options.Margins.T
				block.W -= options.Margins.L + options.Margins.R
				block.H -= options.Margins.T + options.Margins.B
			}

			newBlocks, newMeta := squarify(block.TreeSizer, block, options, depth+1)
			blocks = append(blocks, newBlocks...)
			meta = append(meta, newMeta...)
		}
	}

	return
}

// Sort areas by area.
type byAreaAndPlaceholder []area

func (a byAreaAndPlaceholder) Len() int {
	return len(a)
}

func (a byAreaAndPlaceholder) Less(i, j int) bool {

	if a[i].TreeSizer != nil && a[j].TreeSizer != nil || a[i].TreeSizer == nil && a[j].TreeSizer == nil {
		return a[i].Area > a[j].Area
	}

	return a[i].TreeSizer != nil
}

func (a byAreaAndPlaceholder) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

// areas computes the areas for the children of `root` and sorts them if dosort is true.
// The area of a child is calculated as it's proportion of the parent's Block's size, where
// `root` is the parent, and `Block` is the dimensions of the parent.
func areas(root TreeSizer, block Block, dosort bool) (areas []area) {
	blockArea := block.W * block.H

	areas = make([]area, 0)
	itemsTotalSize := float64(0)

	for i := 0; i < root.NumChildren(); i++ {
		item := root.Child(i)

		// Ignore 0-size items
		if item.Size() <= 0 {
			continue
		}

		areas = append(areas, area{Area: item.Size() / root.Size() * blockArea, TreeSizer: item})
		itemsTotalSize += item.Size()
	}

	// Add a placeholder area for extra space
	if itemsTotalSize < root.Size() {
		a := (root.Size() - itemsTotalSize) / root.Size() * blockArea
		areas = append(areas, area{Area: a, TreeSizer: nil})
	}

	if dosort {
		sort.Sort(byAreaAndPlaceholder(areas))
	}

	return
}
