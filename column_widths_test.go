package pdf

import (
	"fmt"
	"math"
	"testing"
)

func TestCalculateColumnWidths(t *testing.T) {
	tests := []struct {
		name       string
		specs      []string
		totalWidth Unit
		expected   []Unit
		shouldErr  bool
	}{
		{
			name:       "User's example",
			specs:      []string{"15mm", "65pt", "10%", "n", "2n"},
			totalWidth: Pt(500),
			// Expected: 15mm≈42.52pt, 65pt, 10%=50pt, remaining 342.48pt split 1:2 (≈114.16pt, ≈228.32pt)
			expected: []Unit{Mm(15), Pt(65), Pt(50), Pt(114.16), Pt(228.32)},
		},
		{
			name:       "Mixed units with percentages",
			specs:      []string{"2cm", "25%", "1in", "n"},
			totalWidth: Pt(400),
			// Expected: 2cm≈56.69pt, 25%=100pt, 1in=72pt, remaining 171.31pt for n
			expected: []Unit{Cm(2), Pt(100), In(1), Pt(171.31)},
		},
		{
			name:       "Only flex units",
			specs:      []string{"n", "2n", "3n"},
			totalWidth: Pt(600),
			// Expected: 600pt split 1:2:3 = 100pt, 200pt, 300pt
			expected: []Unit{Pt(100), Pt(200), Pt(300)},
		},
		{
			name:       "Only fixed units",
			specs:      []string{"50pt", "30mm", "1in"},
			totalWidth: Pt(500),
			// Expected: 50pt, 30mm≈85.04pt, 1in=72pt
			expected: []Unit{Pt(50), Mm(30), In(1)},
		},
		{
			name:       "Decimal flex ratios",
			specs:      []string{"100pt", "1.5n", "0.5n"},
			totalWidth: Pt(500),
			// Expected: 100pt, remaining 400pt split 1.5:0.5 = 300pt, 100pt
			expected: []Unit{Pt(100), Pt(300), Pt(100)},
		},
		{
			name:      "Invalid percentage",
			specs:     []string{"150%"},
			shouldErr: true,
		},
		{
			name:      "Invalid unit",
			specs:     []string{"15px"},
			shouldErr: true,
		},
		{
			name:      "Invalid flex ratio",
			specs:     []string{"-2n"},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalculateColumnWidths(tt.specs, tt.totalWidth)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d widths, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				actual := result[i]
				// Allow for small floating point differences
				if math.Abs(float64(actual-expected)) > 0.1 {
					t.Errorf("Width %d: expected %.2f pt, got %.2f pt", i, expected.Pt(), actual.Pt())
				}
			}
		})
	}
}

func TestCalculateColumnWidthsDemo(t *testing.T) {
	// Demonstrate the user's example with detailed output
	specs := []string{"15mm", "65pt", "10%", "n", "2n"}
	totalWidth := Pt(500)

	fmt.Printf("\n=== Column Width Calculation Demo ===\n")
	fmt.Printf("Input specs: %v\n", specs)
	fmt.Printf("Total width: %.2f pt\n", totalWidth.Pt())

	widths, err := CalculateColumnWidths(specs, totalWidth)
	if err != nil {
		t.Fatalf("Error calculating widths: %v", err)
	}

	fmt.Printf("\nCalculated widths:\n")
	for i, width := range widths {
		fmt.Printf("  %s -> %.2f pt (%.2f mm)\n", specs[i], width.Pt(), width.Mm())
	}

	// Verify total
	total := Unit(0)
	for _, width := range widths {
		total += width
	}
	fmt.Printf("\nTotal calculated width: %.2f pt\n", total.Pt())
	fmt.Printf("Difference from target: %.2f pt\n", math.Abs(total.Pt()-totalWidth.Pt()))
}

func ExampleCalculateColumnWidths() {
	// Example usage for table column widths
	specs := []string{"15mm", "65pt", "10%", "n", "2n"}
	totalWidth := Pt(500) // e.g., canvas.Width()

	widths, err := CalculateColumnWidths(specs, totalWidth)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Column widths: ")
	for i, width := range widths {
		if i > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%.1f pt", width.Pt())
	}
	// Output: Column widths: 42.5 pt, 65.0 pt, 50.0 pt, 114.2 pt, 228.3 pt
}
