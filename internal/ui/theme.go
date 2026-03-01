package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// DefaultFallbackColor is used when color parsing fails
const DefaultFallbackColor = "#808080" // Medium gray

// ThemePalette defines a complete color theme for the UI
type ThemePalette struct {
	Name           string
	Gradient       []string // 30-color gradient for pulsing effects
	FlashColor     lipgloss.Color
	GlowColor      lipgloss.Color // Background tint for glow effect
	TitleColor     lipgloss.Color // For focused column titles
	ReadyColor     lipgloss.Color // For "READY?" text
	LaunchBg       lipgloss.Color // For LAUNCH button background
	LaunchFg       lipgloss.Color // For LAUNCH button text
	ArcadeGold     lipgloss.Color // For "INSERT COIN" text
	FocusIndicator lipgloss.Color // For ▶ indicator
	AppBg          lipgloss.Color // Main app background
	AppFg          lipgloss.Color // Main app foreground
}

// Glow intensity constants - tune these to adjust glow effect
const (
	GlowOpacityNormal   = 0.15 // Base glow opacity (0.0-1.0)
	GlowOpacityEnhanced = 0.25 // Enhanced glow at pulse peak
)

// Color cycling constants
type ColorCycle struct {
	Base   string // Hex color
	Dark   string // Dark variant for pulse valley
	Bright string // Bright variant for pulse peak
}

// MatrixTheme: Current green-blue (the original aesthetic)
var MatrixTheme = ThemePalette{
	Name: "Matrix",
	Gradient: []string{
		"#90EE90", "#8BE88C", "#86E288", "#81DC88", "#7CD688",
		"#77D088", "#72CA88", "#6DC488", "#68BE88", "#63B888",
		"#5EB288", "#59AC88", "#54A688", "#4FA088", "#4A9A88",
		"#459488", "#408E88", "#3B8888", "#368288", "#317C88",
		"#2C7688", "#277088", "#226A88", "#1D6488", "#185E88",
		"#135888", "#0E5288", "#094C88", "#044688", "#004088",
	},
	FlashColor:     lipgloss.Color("51"),      // Bright cyan
	GlowColor:      lipgloss.Color("#0a2a2a"), // Dark teal for glow
	TitleColor:     lipgloss.Color("#90EE90"), // Light green
	ReadyColor:     lipgloss.Color("#00ff00"), // Matrix green
	LaunchBg:       lipgloss.Color("#00aa00"), // Matrix button
	LaunchFg:       lipgloss.Color("#ffffff"), // White text
	ArcadeGold:     lipgloss.Color("#ffcc00"), // Classic arcade gold
	FocusIndicator: lipgloss.Color("#ffff00"), // Yellow indicator
	AppBg:          lipgloss.Color("#051010"), // Very dark green/teal
	AppFg:          lipgloss.Color("#e0ffe0"), // Very light green fade
}

// MatrixThemeColorCycles defines accent colors for cycling within Matrix theme
var MatrixThemeColorCycles = []ColorCycle{
	{Base: "#90EE90", Dark: "#5EB288", Bright: "#c0ffc0"}, // Green
	{Base: "#00cccc", Dark: "#008888", Bright: "#66ffff"}, // Cyan
	{Base: "#66b2ff", Dark: "#0066cc", Bright: "#99ccff"}, // Blue
	{Base: "#99ff99", Dark: "#66cc66", Bright: "#ccffcc"}, // Light green
}

// CyberpunkTheme: Neon purple-pink-cyan (synthwave aesthetic)
var CyberpunkTheme = ThemePalette{
	Name: "Cyberpunk",
	Gradient: []string{
		"#ff00ff", "#f500f5", "#eb00eb", "#e100e1", "#d700d7",
		"#cd00cd", "#c300c3", "#b900b9", "#af00af", "#a500a5",
		"#9b009b", "#910091", "#870087", "#7d007d", "#730073",
		"#690069", "#5f005f", "#550055", "#4b004b", "#410041",
		"#370037", "#2d002d", "#230023", "#190019", "#0f000f",
		"#00cccc", "#00bbbb", "#00aaaa", "#009999", "#008888",
	},
	FlashColor:     lipgloss.Color("51"),      // Cyan (keeps contrast)
	GlowColor:      lipgloss.Color("#2a0a2a"), // Dark purple for glow
	TitleColor:     lipgloss.Color("#ff00ff"), // Magenta
	ReadyColor:     lipgloss.Color("#ff00ff"), // Neon magenta
	LaunchBg:       lipgloss.Color("#d700d7"), // Cyberpunk button
	LaunchFg:       lipgloss.Color("#ffffff"), // White text
	ArcadeGold:     lipgloss.Color("#ffcc00"), // Classic arcade gold
	FocusIndicator: lipgloss.Color("#00ffff"), // Cyan indicator
	AppBg:          lipgloss.Color("#10051a"), // Very dark purple
	AppFg:          lipgloss.Color("#ffe0ff"), // Very light pink
}

// CyberpunkThemeColorCycles defines accent colors for cycling within Cyberpunk theme
var CyberpunkThemeColorCycles = []ColorCycle{
	{Base: "#ff00ff", Dark: "#990099", Bright: "#ff66ff"}, // Magenta
	{Base: "#ff007f", Dark: "#99004c", Bright: "#ff66b2"}, // Hot pink
	{Base: "#00d1ff", Dark: "#007a99", Bright: "#66e5ff"}, // Electric blue
	{Base: "#8a2be2", Dark: "#531a88", Bright: "#b380ee"}, // Blue-violet
}

// TokyoNightTheme: Deep rich blues and vibrant accents
var TokyoNightTheme = ThemePalette{
	Name: "TokyoNight",
	Gradient: []string{
		"#bb9af7", "#b597f3", "#af93f0", "#a990ec", "#a38ce9",
		"#9d89e5", "#9786e2", "#9182df", "#8b7fdc", "#857bd8",
		"#7f78d5", "#7975d2", "#7371ce", "#6d6ecb", "#676bc7",
		"#6168c4", "#5b64c1", "#5561bd", "#4f5dba", "#495ab7",
		"#4356b3", "#3d53b0", "#3750ad", "#314ca9", "#2b49a6",
		"#2546a3", "#1f42a0", "#193f9c", "#133c99", "#0d3996",
	},
	FlashColor:     lipgloss.Color("#ff9e64"), // Orange
	GlowColor:      lipgloss.Color("#24283b"), // Tokyo night storm bg
	TitleColor:     lipgloss.Color("#7aa2f7"), // Light Blue
	ReadyColor:     lipgloss.Color("#9ece6a"), // Green
	LaunchBg:       lipgloss.Color("#f7768e"), // Red
	LaunchFg:       lipgloss.Color("#15161e"), // Dark text
	ArcadeGold:     lipgloss.Color("#e0af68"), // Gold/Yellow
	FocusIndicator: lipgloss.Color("#7aa2f7"), // Light Blue
	AppBg:          lipgloss.Color("#1a1b26"), // Main theme bg
	AppFg:          lipgloss.Color("#a9b1d6"), // Main theme fg
}

// TokyoNightThemeColorCycles defines accent colors for cycling within TokyoNight theme
var TokyoNightThemeColorCycles = []ColorCycle{
	{Base: "#7aa2f7", Dark: "#3d59a1", Bright: "#89ddff"}, // Blue
	{Base: "#bb9af7", Dark: "#5a4a78", Bright: "#c0a0ff"}, // Purple
	{Base: "#9ece6a", Dark: "#4f6935", Bright: "#b4f9f8"}, // Green
	{Base: "#f7768e", Dark: "#7c3b47", Bright: "#ff9e64"}, // Red/Orange
}

// AvailableThemes is the list of all available themes
var AvailableThemes = []*ThemePalette{
	&MatrixTheme,
	&CyberpunkTheme,
	&TokyoNightTheme,
}

// parseHexColor parses a hex color string to RGB values
// Supports both "#RRGGBB" and "RRGGBB" formats
func parseHexColor(c string) (r, g, b uint8, err error) {
	// Remove # prefix if present
	c = strings.TrimPrefix(c, "#")

	// Parse hex values
	if len(c) != 6 {
		return 0, 0, 0, fmt.Errorf("invalid hex color format: %s", c)
	}

	// Parse each channel
	r64, err1 := strconv.ParseUint(c[0:2], 16, 8)
	g64, err2 := strconv.ParseUint(c[2:4], 16, 8)
	b64, err3 := strconv.ParseUint(c[4:6], 16, 8)

	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, fmt.Errorf("invalid hex color: %s", c)
	}

	return uint8(r64), uint8(g64), uint8(b64), nil
}

// interpolateColor blends between two hex colors using linear interpolation
// t should be between 0.0 and 1.0
func interpolateColor(c1, c2 string, t float64) string {
	if t <= 0 {
		return c1
	}
	if t >= 1 {
		return c2
	}

	r1, g1, b1, err1 := parseHexColor(c1)
	r2, g2, b2, err2 := parseHexColor(c2)

	if err1 != nil || err2 != nil {
		// Use default fallback color if parsing fails
		return DefaultFallbackColor
	}

	// Linear interpolation for each channel
	r := uint8(float64(r1) + (float64(r2)-float64(r1))*t)
	g := uint8(float64(g1) + (float64(g2)-float64(g1))*t)
	b := uint8(float64(b1) + (float64(b2)-float64(b1))*t)

	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}

// getPulsingColorWithTheme returns an interpolated color from the theme's gradient
// phase: 0.0 = darkest, 0.5 = base, 1.0 = brightest
func getPulsingColorWithTheme(phase float64, theme *ThemePalette) lipgloss.Color {
	if theme == nil || len(theme.Gradient) == 0 {
		return lipgloss.Color(DefaultFallbackColor)
	}

	// Get gradient boundaries
	gradientLen := len(theme.Gradient)
	darkestIdx := gradientLen - 3 // Near end of gradient
	brightestIdx := 3             // Near start of gradient

	if darkestIdx < 0 {
		darkestIdx = 0
	}
	if brightestIdx >= gradientLen {
		brightestIdx = gradientLen - 1
	}

	// Map phase 0-1 to gradient index range
	idxFloat := float64(darkestIdx) - phase*float64(darkestIdx-brightestIdx)
	idx := int(idxFloat)

	// Clamp to valid range
	if idx < brightestIdx {
		idx = brightestIdx
	}
	if idx > darkestIdx {
		idx = darkestIdx
	}

	// For smoother transitions, interpolate between adjacent gradient stops
	if idx < darkestIdx {
		// Calculate interpolation factor between this stop and the next
		nextIdx := idx + 1
		if nextIdx > darkestIdx {
			nextIdx = darkestIdx
		}

		// Get fractional part for interpolation
		frac := idxFloat - float64(idx)
		if frac > 0 {
			interpolated := interpolateColor(theme.Gradient[idx], theme.Gradient[nextIdx], frac)
			return lipgloss.Color(interpolated)
		}
	}

	return lipgloss.Color(theme.Gradient[idx])
}

// getCyclingColor returns a color from the theme's cycling palette
// Uses cycleIndex to select base color, phase for pulse within that color
func getCyclingColor(phase float64, cycleIndex int, theme *ThemePalette) lipgloss.Color {
	var cycles []ColorCycle
	if theme == &CyberpunkTheme {
		cycles = CyberpunkThemeColorCycles
	} else {
		cycles = MatrixThemeColorCycles
	}

	if len(cycles) == 0 {
		return getPulsingColorWithTheme(phase, theme)
	}

	// Get current cycle
	idx := cycleIndex % len(cycles)
	cycle := cycles[idx]

	// Interpolate between dark and bright based on phase
	var t float64
	if phase < 0.5 {
		// First half: dark -> base
		t = phase * 2
		return lipgloss.Color(interpolateColor(cycle.Dark, cycle.Base, t))
	}
	// Second half: base -> bright
	t = (phase - 0.5) * 2
	return lipgloss.Color(interpolateColor(cycle.Base, cycle.Bright, t))
}

// getGlowColor returns a background color for glow effect based on current pulse
func getGlowColor(phase float64, theme *ThemePalette) lipgloss.Color {
	if theme == nil {
		return lipgloss.Color(DefaultFallbackColor)
	}

	// Parse the theme's glow color
	r, g, b, err := parseHexColor(string(theme.GlowColor))
	if err != nil {
		return theme.GlowColor
	}

	// Calculate opacity based on phase (enhanced at peak)
	opacity := GlowOpacityNormal
	if phase > 0.5 {
		// Enhance glow during pulse peak
		peakFactor := (phase - 0.5) * 2 // 0 to 1 during second half
		opacity = GlowOpacityNormal + (GlowOpacityEnhanced-GlowOpacityNormal)*peakFactor
	}

	// Apply opacity to RGB (for background color, we just use the color as-is
	// but in a real alpha blend, we'd mix with terminal background)
	// Since lipgloss doesn't support alpha, we dim the color
	dimFactor := 0.3 + opacity*0.7 // At least 30% brightness
	r = uint8(float64(r) * dimFactor)
	g = uint8(float64(g) * dimFactor)
	b = uint8(float64(b) * dimFactor)

	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}
