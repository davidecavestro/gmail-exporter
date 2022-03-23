package ui

import "github.com/vbauerster/mpb/v7"

type ProgressUI struct {
	Hide         bool
	BarContainer *mpb.Progress
}

func (pui *ProgressUI) Init(width int) {
	// initialize progress container, with custom width
	pui.BarContainer = mpb.New(mpb.WithWidth(width))
}
