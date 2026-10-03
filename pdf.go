package pdf

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// PageFormat represents the page size
type PageFormat string

// PageOrientation represents the page orientation
type PageOrientation string

// Alignment represents text alignment
type Alignment string

// BreakMode represents text breaking behavior
type BreakMode int

const (
	A4     PageFormat = "A4"
	Letter PageFormat = "Letter"

	Portrait  PageOrientation = "Portrait"
	Landscape PageOrientation = "Landscape"

	Left    Alignment = "left"
	Center  Alignment = "center"
	Right   Alignment = "right"
	Justify Alignment = "justify" // justify text to fill width

	Wrap     BreakMode = iota // default: break at word boundaries
	NoWrap                    // keep on one line; overflow handled by caller
	Ellipsis                  // trim & add … if too long
	Clip                      // clip to maxWidth/maxHeight

	Automatic Unit = -1
)

// Spacing represents a box model with top, right, bottom, left values
type BoxSpacing struct {
	Top, Right, Bottom, Left Unit
}

func (s BoxSpacing) Width() Unit  { return s.Left + s.Right }
func (s BoxSpacing) Height() Unit { return s.Top + s.Bottom }

// NewBoxSpacing returns a BoxSpacing with the given top/right/bottom/left Units,
// interpreting 1, 2, or 4 args the same way CSS does.
//
// Example:
//
//	Spacing(10) // 10pt on all sides
//	Spacing(10, 20) // 10pt top/bottom, 20pt left/right
//	Spacing(10, 20, 30, 40) // 10pt top, 20pt right, 30pt bottom, 40pt left
func NewBoxSpacing(values ...Unit) BoxSpacing {
	switch len(values) {
	case 1:
		return BoxSpacing{values[0], values[0], values[0], values[0]}
	case 2:
		return BoxSpacing{values[0], values[1], values[0], values[1]}
	case 4:
		return BoxSpacing{values[0], values[1], values[2], values[3]}
	default:
		panic("Spacing expects 1, 2, or 4 arguments")
	}
}

var Spacing = NewBoxSpacing
var Margins = NewBoxSpacing
var Padding = NewBoxSpacing

// Utility functions
type Unit float64

const ptPerIn = 72.0
const ptPerMm = ptPerIn / 25.4

// NoLimit represents unlimited height/width for measure and draw operations
const NoLimit = Unit(math.MaxFloat64)

func Pt(x float64) Unit { return Unit(x) }
func Mm(x float64) Unit { return Unit(x) * ptPerMm }
func In(x float64) Unit { return Unit(x) * ptPerIn }
func Cm(x float64) Unit { return Unit(x) * ptPerMm * 10 }

func (u Unit) Pt() float64 { return float64(u) }
func (u Unit) Mm() float64 { return float64(u) / ptPerMm }

func (u Unit) Max(v Unit) Unit    { return Unit(math.Max(u.Pt(), v.Pt())) }
func (u Unit) Min(v Unit) Unit    { return Unit(math.Min(u.Pt(), v.Pt())) }
func (u Unit) Abs() Unit          { return Unit(math.Abs(u.Pt())) }
func (u Unit) Mul(x float64) Unit { return Unit(u.Pt() * x) }
func (u Unit) Div(x float64) Unit { return Unit(u.Pt() / x) }

// CalculateColumnWidths takes a slice of width specifications and total width,
// returning calculated widths as Unit values.
//
// Supported formats:
// - "15mm", "65pt", "2.5cm", "1in" - fixed units
// - "10%" - percentage of total width
// - "n", "2n", "3n" - flexible units (remaining space divided proportionally)
//
// Example: ["15mm", "65pt", "10%", "n", "2n"] with totalWidth=500pt
// - "15mm" -> ~42.5pt
// - "65pt" -> 65pt
// - "10%" -> 50pt
// - remaining space (500 - 42.5 - 65 - 50 = 342.5pt) divided 1:2
// - "n" -> ~114.2pt, "2n" -> ~228.3pt
func CalculateColumnWidths(specs []string, totalWidth Unit) ([]Unit, error) {
	if len(specs) == 0 {
		return nil, nil
	}

	widths := make([]Unit, len(specs))
	flexSpecs := make([]flexSpec, 0)
	usedWidth := Unit(0)

	// First pass: handle fixed units and percentages
	for i, spec := range specs {
		if spec == "" {
			continue
		}

		// Check for flexible units (n, 2n, 3n, etc.)
		if isFlexUnit(spec) {
			ratio, err := parseFlexRatio(spec)
			if err != nil {
				return nil, err
			}
			flexSpecs = append(flexSpecs, flexSpec{index: i, ratio: ratio})
			continue
		}

		// Check for percentage
		if strings.HasSuffix(spec, "%") {
			percent, err := parsePercentage(spec)
			if err != nil {
				return nil, err
			}
			width := totalWidth * Unit(percent/100)
			widths[i] = width
			usedWidth += width
			continue
		}

		// Handle fixed units
		width, err := parseFixedUnit(spec)
		if err != nil {
			return nil, err
		}
		widths[i] = width
		usedWidth += width
	}

	// Second pass: distribute remaining space among flexible units
	if len(flexSpecs) > 0 {
		remainingWidth := totalWidth - usedWidth
		if remainingWidth < 0 {
			remainingWidth = 0
		}

		// Calculate total ratio
		totalRatio := 0.0
		for _, flex := range flexSpecs {
			totalRatio += flex.ratio
		}

		// Distribute remaining space proportionally
		if totalRatio > 0 {
			for _, flex := range flexSpecs {
				flexWidth := remainingWidth * Unit(flex.ratio/totalRatio)
				widths[flex.index] = flexWidth
			}
		}
	}

	return widths, nil
}

type flexSpec struct {
	index int
	ratio float64
}

// isFlexUnit checks if a spec is a flexible unit (n, 2n, 3n, etc.)
func isFlexUnit(spec string) bool {
	return strings.HasSuffix(spec, "n") && (spec == "n" || isNumericPrefix(spec[:len(spec)-1]))
}

// parseFlexRatio extracts the ratio from a flex unit spec
func parseFlexRatio(spec string) (float64, error) {
	if spec == "n" {
		return 1.0, nil
	}

	ratioStr := spec[:len(spec)-1]
	ratio, err := strconv.ParseFloat(ratioStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid flex ratio: %s", spec)
	}

	if ratio <= 0 {
		return 0, fmt.Errorf("flex ratio must be positive: %s", spec)
	}

	return ratio, nil
}

// parsePercentage extracts percentage value from a percentage spec
func parsePercentage(spec string) (float64, error) {
	percentStr := spec[:len(spec)-1]
	percent, err := strconv.ParseFloat(percentStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid percentage: %s", spec)
	}

	if percent < 0 || percent > 100 {
		return 0, fmt.Errorf("percentage must be between 0 and 100: %s", spec)
	}

	return percent, nil
}

// parseFixedUnit parses fixed unit specifications like "15mm", "65pt", etc.
func parseFixedUnit(spec string) (Unit, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return 0, fmt.Errorf("empty unit specification")
	}

	// Try to find the unit suffix
	var valueStr string
	var unitStr string

	// Common unit suffixes to check
	units := []string{"mm", "pt", "cm", "in"}

	for _, unit := range units {
		if strings.HasSuffix(spec, unit) {
			valueStr = spec[:len(spec)-len(unit)]
			unitStr = unit
			break
		}
	}

	if unitStr == "" {
		return 0, fmt.Errorf("unknown unit in specification: %s", spec)
	}

	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric value: %s", spec)
	}

	if value < 0 {
		return 0, fmt.Errorf("negative values not allowed: %s", spec)
	}

	// Convert to points using existing conversion functions
	switch unitStr {
	case "pt":
		return Pt(value), nil
	case "mm":
		return Mm(value), nil
	case "cm":
		return Cm(value), nil
	case "in":
		return In(value), nil
	default:
		return 0, fmt.Errorf("unsupported unit: %s", unitStr)
	}
}

// isNumericPrefix checks if a string represents a valid numeric prefix
func isNumericPrefix(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
