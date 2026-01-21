package conflict

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"

	"github.com/matsen/bipartite/internal/reference"
)

// Parser state machine states
type parserState int

const (
	stateNormal parserState = iota
	stateInOurs
	stateInTheirs
)

// Conflict marker prefixes
const (
	oursMarker      = "<<<<<<<"
	separatorMarker = "======="
	theirsMarker    = ">>>>>>>"
)

// Parse reads a conflicted file and returns the parse result.
// It identifies clean lines and conflict regions with their content.
func Parse(r io.Reader) (*ParseResult, error) {
	scanner := bufio.NewScanner(r)
	result := &ParseResult{}

	state := stateNormal
	lineNum := 0
	var currentConflict *ConflictRegion
	var oursLines []string
	var theirsLines []string

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		switch state {
		case stateNormal:
			if strings.HasPrefix(line, oursMarker) {
				// Start of conflict region
				currentConflict = &ConflictRegion{
					StartLine: lineNum,
				}
				oursLines = nil
				theirsLines = nil
				state = stateInOurs
			} else if strings.HasPrefix(line, separatorMarker) {
				return nil, ParseError{
					Line:    lineNum,
					Message: "unexpected separator marker outside conflict region",
					Context: line,
				}
			} else if strings.HasPrefix(line, theirsMarker) {
				return nil, ParseError{
					Line:    lineNum,
					Message: "unexpected end marker outside conflict region",
					Context: line,
				}
			} else {
				// Clean line
				result.CleanLines = append(result.CleanLines, CleanLine{
					LineNum: lineNum,
					Content: line,
				})
			}

		case stateInOurs:
			if strings.HasPrefix(line, oursMarker) {
				return nil, ParseError{
					Line:    lineNum,
					Message: "nested conflict markers not allowed",
					Context: line,
				}
			} else if strings.HasPrefix(line, separatorMarker) {
				// Transition to theirs
				state = stateInTheirs
			} else if strings.HasPrefix(line, theirsMarker) {
				return nil, ParseError{
					Line:    lineNum,
					Message: "unexpected end marker before separator",
					Context: line,
				}
			} else {
				oursLines = append(oursLines, line)
			}

		case stateInTheirs:
			if strings.HasPrefix(line, oursMarker) {
				return nil, ParseError{
					Line:    lineNum,
					Message: "nested conflict markers not allowed",
					Context: line,
				}
			} else if strings.HasPrefix(line, separatorMarker) {
				return nil, ParseError{
					Line:    lineNum,
					Message: "duplicate separator marker in conflict region",
					Context: line,
				}
			} else if strings.HasPrefix(line, theirsMarker) {
				// End of conflict region
				currentConflict.EndLine = lineNum
				currentConflict.OursRaw = strings.Join(oursLines, "\n")
				currentConflict.TheirsRaw = strings.Join(theirsLines, "\n")

				// Parse JSONL content from both sides
				var err error
				currentConflict.OursRefs, err = parseJSONLContent(oursLines, currentConflict.StartLine+1)
				if err != nil {
					return nil, err
				}
				currentConflict.TheirsRefs, err = parseJSONLContent(theirsLines, currentConflict.StartLine+1+len(oursLines)+1)
				if err != nil {
					return nil, err
				}

				result.Conflicts = append(result.Conflicts, *currentConflict)
				currentConflict = nil
				state = stateNormal
			} else {
				theirsLines = append(theirsLines, line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Check for unterminated conflict
	if state != stateNormal {
		return nil, ParseError{
			Line:    lineNum,
			Message: "unterminated conflict region at end of file",
			Context: "",
		}
	}

	return result, nil
}

// parseJSONLContent parses JSONL lines into references.
func parseJSONLContent(lines []string, startLine int) ([]reference.Reference, error) {
	var refs []reference.Reference

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var ref reference.Reference
		if err := json.Unmarshal([]byte(line), &ref); err != nil {
			return nil, ParseError{
				Line:    startLine + i,
				Message: "invalid JSON: " + err.Error(),
				Context: truncate(line, 50),
			}
		}
		refs = append(refs, ref)
	}

	return refs, nil
}

// truncate truncates a string to maxLen characters, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// ParseString is a convenience function that parses from a string.
func ParseString(content string) (*ParseResult, error) {
	return Parse(strings.NewReader(content))
}

// HasConflicts returns true if the parse result contains any conflict regions.
func (r *ParseResult) HasConflicts() bool {
	return len(r.Conflicts) > 0
}
