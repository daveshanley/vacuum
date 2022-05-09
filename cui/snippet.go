package cui

import (
	"image"

	. "github.com/gizak/termui/v3"
)

type Snippet struct {
	Block
	Text      string
	TextStyle Style
	WrapText  bool
}

func NewSnippet() *Snippet {
	return &Snippet{
		Block:     *NewBlock(),
		TextStyle: Theme.Paragraph.Text,
		WrapText:  false,
	}
}

func (snip *Snippet) Draw(buf *Buffer) {
	snip.Block.Draw(buf)

	cells := ParseStyles(snip.Text, snip.TextStyle)
	if snip.WrapText {
		cells = WrapCells(cells, uint(snip.Inner.Dx()))
	}

	rows := SplitCells(cells, '\n')
	for y, row := range rows {
		if y+snip.Inner.Min.Y >= snip.Inner.Max.Y {
			break
		}
		row = TrimCells(row, snip.Inner.Dx())
		for _, cx := range BuildCellWithXArray(row) {
			x, cell := cx.X, cx.Cell
			buf.SetCell(cell, image.Pt(x, y).Add(snip.Inner.Min))
		}
	}
}
