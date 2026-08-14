package parser_test

import (
	"strings"
	"testing"

	"github.com/marcelbrand/binderlm/internal/parser"
)

func TestShiftHeadingsASTSafety(t *testing.T) {
	input := `# Top Level Heading

Some introductory text.

## Second Level

Here is a Python code block with comments:
` + "```python\n# This is a Python comment, NOT a heading\ndef foo():\n    ## Another comment\n    return 42\n```" + `

Here is a shell block:
` + "```bash\n# Echo something\necho \"# Not a heading\"\n```" + `

### Third Level Heading

` + "    # Indented code block comment\n    ls -la" + `

#### Fourth Level
##### Fifth Level
###### Sixth Level
`

	shifter := parser.NewHeadingShifter(6)
	shifted, headings, err := shifter.ShiftHeadings([]byte(input), 2)
	if err != nil {
		t.Fatalf("unexpected error shifting headings: %v", err)
	}

	shiftedStr := string(shifted)

	// Check shifted levels:
	// Top Level was H1 -> should now be H3 (###)
	if !strings.Contains(shiftedStr, "### Top Level Heading") {
		t.Errorf("expected '### Top Level Heading', got:\n%s", shiftedStr)
	}
	// Second Level was H2 -> should now be H4 (####)
	if !strings.Contains(shiftedStr, "#### Second Level") {
		t.Errorf("expected '#### Second Level', got:\n%s", shiftedStr)
	}
	// Third Level was H3 -> should now be H5 (#####)
	if !strings.Contains(shiftedStr, "##### Third Level Heading") {
		t.Errorf("expected '##### Third Level Heading', got:\n%s", shiftedStr)
	}
	// Sixth Level was H6 -> clamped at H6 (######)
	if !strings.Contains(shiftedStr, "###### Sixth Level") {
		t.Errorf("expected '###### Sixth Level', got:\n%s", shiftedStr)
	}

	// Verify Python comments inside ```python are intact and UNTOUCHED
	if !strings.Contains(shiftedStr, "# This is a Python comment, NOT a heading") {
		t.Errorf("Python comment in code fence was corrupted! Result:\n%s", shiftedStr)
	}
	if !strings.Contains(shiftedStr, "## Another comment") {
		t.Errorf("Double-hash Python comment was corrupted! Result:\n%s", shiftedStr)
	}
	if !strings.Contains(shiftedStr, "# Echo something") {
		t.Errorf("Shell comment in code fence was corrupted! Result:\n%s", shiftedStr)
	}

	// Verify extracted headings count
	if len(headings) != 6 {
		t.Errorf("expected 6 headings, got %d", len(headings))
	}
}
