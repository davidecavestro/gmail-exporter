package ui

import (
	"fmt"

	"github.com/vbauerster/mpb/v7"
	"github.com/vbauerster/mpb/v7/decor"
)

type ProgressUI struct {
	Hide         bool
	BarContainer *mpb.Progress

	bar         *mpb.Bar
	progressBar *mpb.Bar
}

func (pui *ProgressUI) Init(width int) {
	if pui.Hide {
		return
	}
	// initialize progress container, with custom width
	pui.BarContainer = mpb.New(mpb.WithWidth(width))
}

func (pui *ProgressUI) GmailPageTotal(pageTotal int64) {
	if pui.Hide {
		return
	}
	pui.bar.SetTotal(int64(pageTotal), false)
}

func (pui *ProgressUI) GmailIncrement() {
	if pui.Hide {
		return
	}
	pui.bar.Increment()
}

func (pui *ProgressUI) GmailNewPage(pageSize int64, pageNum int64) {
	if pui.Hide {
		return
	}
	taskName := fmt.Sprintf("Page %5d", pageNum+1)
	pui.bar = pui.BarContainer.New(pageSize,
		mpb.BarStyle(), /*.Lbound("╢").Filler("▌").Tip("▌").Padding("░").Rbound("╟")*/
		mpb.PrependDecorators(
			decor.Name(taskName, decor.WC{W: len(taskName), C: decor.DidentRight}),
			// decor.Name("acquiring ", decor.WCSyncSpaceR),
			// decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
		),
		mpb.AppendDecorators(decor.Percentage(decor.WC{W: 5})),
	)
}

func (pui *ProgressUI) SpreadsheetTotal(total int64) {
	if pui.Hide {
		return
	}
	taskName := "ETA"

	pui.progressBar = pui.BarContainer.New(total,
		// BarFillerBuilder with custom style
		mpb.BarStyle(),
		mpb.PrependDecorators(
			decor.Name(taskName, decor.WC{W: len(taskName) + 1, C: decor.DidentRight}),
			decor.OnComplete(
				decor.AverageETA(decor.ET_STYLE_GO, decor.WC{W: 6}), "  done",
			),
		),
		mpb.AppendDecorators(
			// decor.Percentage(decor.WC{W: 5})
			decor.CountersNoUnit("%d / %d", decor.WCSyncWidth),
		),
	)
}

func (pui *ProgressUI) SpreadsheetIncrement() {
	if pui.Hide {
		return
	}
	pui.progressBar.Increment()
}
