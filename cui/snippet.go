package cui

import (
	"image"

	"github.com/gizak/termui/v3"
)

type Snippet struct {
	termui.Block
	Text      string
	TextStyle termui.Style
	WrapText  bool
}

func NewSnippet() *Snippet {
	return &Snippet{
		Block:     *termui.NewBlock(),
		TextStyle: termui.Theme.Paragraph.Text,
		WrapText:  false,
	}
}

func (snip *Snippet) Draw(buf *termui.Buffer) {
	snip.Block.Draw(buf)

	cells := termui.ParseStyles(snip.Text, snip.TextStyle)
	if snip.WrapText {
		cells = termui.WrapCells(cells, uint(snip.Inner.Dx()))
	}

	rows := termui.SplitCells(cells, '\n')
	for y, row := range rows {
		if y+snip.Inner.Min.Y >= snip.Inner.Max.Y {
			break
		}
		row = termui.TrimCells(row, snip.Inner.Dx())
		for _, cx := range termui.BuildCellWithXArray(row) {
			x, cell := cx.X, cx.Cell
			buf.SetCell(cell, image.Pt(x, y).Add(snip.Inner.Min))
		}
	}
}
