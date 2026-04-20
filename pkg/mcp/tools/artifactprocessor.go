package tools

import (
	"fmt"
	"strings"
)

const (
	artifactDefaultLines   = 100
	artifactMaxLines       = 200
	artifactGrepContext    = 3
	artifactGrepMaxMatches = 100
)

// ArtifactReadParams holds optional filtering parameters for artifact retrieval.
type ArtifactReadParams struct {
	StartLine int    // 1-based start line. 0 = from beginning.
	EndLine   int    // 1-based end line. 0 = default window of artifactDefaultLines (100) lines from startLine.
	Grep      string // Substring filter (case-insensitive). Includes context lines.
}

// ProcessArtifact filters artifact content and prepends a metadata header.
// For binary content, returns a summary message instead.
func ProcessArtifact(content []byte, filename string, params ArtifactReadParams) string {
	// Binary detection: check for null bytes in first 512 bytes
	checkLen := len(content)
	if checkLen > 512 {
		checkLen = 512
	}
	for i := 0; i < checkLen; i++ {
		if content[i] == 0 {
			return fmt.Sprintf("Binary artifact (%s, %s) -- line-based display not available. Use the artifact URL to view directly.",
				filename, formatSize(len(content)))
		}
	}

	text := string(content)
	lines := strings.Split(text, "\n")
	totalLines := len(lines)

	// Grep mode -- mutually exclusive with startLine/endLine (grep takes priority).
	if params.Grep != "" {
		return processGrep(lines, totalLines, filename, content, params.Grep)
	}

	// Range mode
	startLine := params.StartLine
	endLine := params.EndLine

	if startLine == 0 {
		startLine = 1
	}
	if endLine == 0 {
		endLine = startLine + artifactDefaultLines - 1
	}

	// Cap range at artifactMaxLines
	if endLine-startLine+1 > artifactMaxLines {
		endLine = startLine + artifactMaxLines - 1
	}

	// Clamp to actual content
	if startLine > totalLines {
		header := fmt.Sprintf("--- Artifact Metadata ---\nFile: %s\nTotal lines: %d\nSize: %s\nShowing: 0 lines (startLine %d exceeds total %d lines)\n---",
			filename, totalLines, formatSize(len(content)), startLine, totalLines)
		return header
	}
	if endLine > totalLines {
		endLine = totalLines
	}

	// Extract range (convert to 0-based)
	selected := lines[startLine-1 : endLine]

	header := fmt.Sprintf("--- Artifact Metadata ---\nFile: %s\nTotal lines: %d\nSize: %s\nShowing: lines %d-%d of %d\n---",
		filename, totalLines, formatSize(len(content)), startLine, endLine, totalLines)

	return header + "\n" + strings.Join(selected, "\n")
}

func processGrep(lines []string, totalLines int, filename string, content []byte, pattern string) string {
	lowerPattern := strings.ToLower(pattern)
	matchLineNums := make([]int, 0)
	matchSet := make(map[int]bool)

	for i, line := range lines {
		if strings.Contains(strings.ToLower(line), lowerPattern) {
			matchLineNums = append(matchLineNums, i)
			matchSet[i] = true
			if len(matchLineNums) >= artifactGrepMaxMatches {
				break
			}
		}
	}

	// Collect matching lines with context
	includeSet := make(map[int]bool)
	for _, lineNum := range matchLineNums {
		for c := lineNum - artifactGrepContext; c <= lineNum+artifactGrepContext; c++ {
			if c >= 0 && c < totalLines {
				includeSet[c] = true
			}
		}
	}

	var result []string
	lastIncluded := -2
	for i := 0; i < totalLines; i++ {
		if !includeSet[i] {
			continue
		}
		if lastIncluded >= 0 && i > lastIncluded+1 {
			result = append(result, "...")
		}
		prefix := "  "
		if matchSet[i] {
			prefix = "> "
		}
		result = append(result, fmt.Sprintf("%s%4d: %s", prefix, i+1, lines[i]))
		lastIncluded = i
	}

	// Cap output (only count content lines, not separators)
	contentCount := 0
	capIndex := len(result)
	for i, line := range result {
		if line != "..." {
			contentCount++
		}
		if contentCount > artifactMaxLines {
			capIndex = i
			break
		}
	}
	result = result[:capIndex]

	// Count visible matches after capping
	visibleMatches := 0
	for _, line := range result {
		if len(line) > 2 && line[0] == '>' && line[1] == ' ' {
			visibleMatches++
		}
	}

	truncated := ""
	if visibleMatches < len(matchLineNums) {
		truncated = " (truncated)"
	}

	header := fmt.Sprintf("--- Artifact Metadata ---\nFile: %s\nTotal lines: %d\nSize: %s\nShowing: %d grep matches for %q (with %d lines context)%s\n---",
		filename, totalLines, formatSize(len(content)), visibleMatches, pattern, artifactGrepContext, truncated)

	return header + "\n" + strings.Join(result, "\n")
}

func formatSize(bytes int) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
