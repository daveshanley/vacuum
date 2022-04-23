package cui

import (
	"fmt"
	"github.com/daveshanley/vacuum/model"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
)

// StatsChart represents a bar chart showing statistics of the specification.
type StatsChart struct {
	bc *widgets.List
}

// NewStatsChart returns a new gauge widget that is ready to render
func NewStatsChart(index *model.SpecIndex, info *model.SpecInfo) StatsChart {
	bc := widgets.NewList()

	//paramCount := len(index.GetAllParameters()) index. + len(index.GetAllParametersFromOperations())
	opPCount := index.GetOperationsParameterCount()
	cPCount := index.GetComponentParameterCount()

	bc.Rows = []string{

		fmt.Sprintf("📏 Filesize: [%dkb](fg:green)", len(*info.SpecBytes)/1024),
		fmt.Sprintf("🔦 Spec Type: [%s/%s](fg:green)", info.SpecType, info.SpecFormat),
		fmt.Sprintf("#️⃣ Version: [%s](fg:green)", info.Version),
		fmt.Sprintf("🚗 References: [%d](fg:green)", len(index.GetMappedReferences())),
		fmt.Sprintf("📦 External Docs: [%d](fg:green)", len(index.GetAllExternalDocuments())),
		fmt.Sprintf("📜 Schemas: [%d](fg:green)", len(index.GetAllSchemas())),
		fmt.Sprintf("\U0001F9FE Parameters: [%d](fg:green)", opPCount+cPCount),
		fmt.Sprintf("🔗 Links: [%d](fg:green)", len(index.GetAllLinks())),
		fmt.Sprintf("➡️ Paths: [%d](fg:green)", index.GetPathCount()),
		fmt.Sprintf("⚡ Operations: [%d](fg:green)", index.GetOperationCount()),
		fmt.Sprintf("🏷️ Tags: [%d](fg:green)", index.GetTotalTagsCount()),
		fmt.Sprintf("🗺️ Examples: [%d](fg:green)", len(index.GetAllExamples())),
		fmt.Sprintf("✒️ Enums: [%d](fg:green)", len(index.GetAllEnums())),
		fmt.Sprintf("🔒 Security Schemes: [%d](fg:green)", len(index.GetAllSecuritySchemes())),
	}
	bc.Title = "Spec Statistics"
	//bc.Labels = []string{"R", "E", "S", "P", "L", "I", "O", "T"}
	bc.SelectedRowStyle = ui.NewStyle(ui.ColorGreen)
	bc.BorderBottom = false
	bc.BorderLeft = false
	bc.BorderRight = false
	bc.PaddingTop = 1
	return StatsChart{bc: bc}
}
