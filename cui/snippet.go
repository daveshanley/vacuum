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

func (self *Snippet) Draw(buf *Buffer) {
	self.Block.Draw(buf)

	cells := ParseStyles(self.Text, self.TextStyle)
	if self.WrapText {
		cells = WrapCells(cells, uint(self.Inner.Dx()))
	}

	rows := SplitCells(cells, '\n')

	for y, row := range rows {
		if y+self.Inner.Min.Y >= self.Inner.Max.Y {
			break
		}
		row = TrimCells(row, self.Inner.Dx())
		for _, cx := range BuildCellWithXArray(row) {
			x, cell := cx.X, cx.Cell
			buf.SetCell(cell, image.Pt(x, y).Add(self.Inner.Min))
		}
	}
}
