package ui

// LayoutDimensions holds all calculated layout dimensions for the UI.
type LayoutDimensions struct {
	Width  int
	Height int

	TermWidth  int
	TermHeight int

	SidebarWidth int
	TWidth       int
	HWidth       int
	MWidth       int
	AWidth       int

	InnerListHeight int
}

const (
	marginWithSidebar    = 8
	marginWithoutSidebar = 6
	columnCount          = 4
	harnessWidthFactor   = 2
	minAgentWidth        = 10
	borderWidth          = 2
)

// Compute calculates all layout dimensions based on terminal size and sidebar visibility.
func Compute(termW, termH int, showSidebar bool) LayoutDimensions {
	h, v := docStyle.GetFrameSize()

	width := termW - h
	height := termH - v - verticalMargins - footerHeight

	if width < minWindowWidth {
		width = minWindowWidth
	}
	if height < minWindowHeight {
		height = minWindowHeight
	}

	var usableWidth int
	if showSidebar {
		usableWidth = width - marginWithSidebar
	} else {
		usableWidth = width - marginWithoutSidebar
	}

	baseX := usableWidth / columnCount

	var sidebarWidth, tWidth, hWidth, mWidth, aWidth int
	if showSidebar {
		sidebarWidth = baseX
		tWidth = baseX
		hWidth = baseX / harnessWidthFactor
		mWidth = baseX
		aWidth = usableWidth - (sidebarWidth + tWidth + hWidth + mWidth)
		if aWidth < minAgentWidth {
			aWidth = minAgentWidth
		}
	} else {
		sidebarWidth = 0
		tWidth = baseX
		hWidth = baseX
		mWidth = baseX
		aWidth = usableWidth - (tWidth + hWidth + mWidth)
	}

	listHeight := height - filterHeight
	innerListHeight := listHeight - borderWidth - 1
	if innerListHeight < 1 {
		innerListHeight = 1
	}

	return LayoutDimensions{
		Width:           width,
		Height:          height,
		TermWidth:       termW,
		TermHeight:      termH,
		SidebarWidth:    sidebarWidth,
		TWidth:          tWidth,
		HWidth:          hWidth,
		MWidth:          mWidth,
		AWidth:          aWidth,
		InnerListHeight: innerListHeight,
	}
}
