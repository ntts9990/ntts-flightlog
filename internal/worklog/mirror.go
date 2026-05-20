package worklog

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// Timestamp returns the current UTC time in ISO 8601 format (v1 compatible).
func Timestamp() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05Z")
}

// EpochSeconds returns current Unix epoch as a string.
func EpochSeconds() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}

// FormatDuration formats seconds as "Xh Ym Zs" / "Ym Zs" / "Zs" (v1 format).
func FormatDuration(totalSeconds int64) string {
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm %02ds", m, s)
	default:
		return fmt.Sprintf("%ds", s)
	}
}

// EnsureMainMd creates the worklog directory and main.md if they don't exist.
// Mirrors v1 ensure_worklog behavior exactly (Korean headings, same structure).
func EnsureMainMd(c *Config, title string) error {
	if err := c.EnsureDir(); err != nil {
		return err
	}
	if _, err := os.Stat(c.MainMd); os.IsNotExist(err) {
		ts := Timestamp()
		if title == "" {
			title = "작업 기록"
		}
		content := fmt.Sprintf(`# %s

업데이트: %s

## 현재 상태

- 상태: 초기화됨
- 모드: 미지정
- 초점: 작업 기록 pane이 준비되었습니다.
- 다음: 다음 작업 턴을 시작하세요.
- 시작: %s

## 작업 기록

`, title, ts, ts)
		if err := os.WriteFile(c.MainMd, []byte(content), 0o644); err != nil {
			return fmt.Errorf("mirror: create main.md: %w", err)
		}
	}
	// Ensure session-start-epoch exists.
	if _, err := os.Stat(c.SessionStart); os.IsNotExist(err) {
		if werr := WriteFile(c.SessionStart, EpochSeconds()); werr != nil {
			return werr
		}
	}
	return nil
}

// AppendEntry appends a v1-format entry line to main.md and the current turn file.
// Format: `\n### TS [kind] title\ndetail`
func AppendEntry(c *Config, kind, title, detail string) error {
	if err := EnsureMainMd(c, ""); err != nil {
		return err
	}
	block := formatEntryBlock(kind, title, detail)
	if err := appendToFile(c.MainMd, block); err != nil {
		return fmt.Errorf("mirror: append to main.md: %w", err)
	}
	return appendToCurrentTurn(c, kind, title, detail)
}

// formatEntryBlock returns the v1-format `### TS [kind] title\ndetail` block.
func formatEntryBlock(kind, title, detail string) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("\n### %s [%s] %s\n", Timestamp(), kind, title))
	if detail != "" {
		sb.WriteString(detail)
		sb.WriteString("\n")
	}
	return sb.String()
}

// appendToCurrentTurn appends an entry to the current turn file if one is active.
func appendToCurrentTurn(c *Config, kind, title, detail string) error {
	if _, err := os.Stat(c.TurnStart); os.IsNotExist(err) {
		return nil // no active turn
	}
	n := c.CurrentTurnNumber()
	if n == 0 {
		return nil
	}
	turnFile := c.TurnFilePath(n)
	block := formatEntryBlock(kind, title, detail)
	return appendToFile(turnFile, block)
}

// appendToFile appends content to a file, creating it if needed.
func appendToFile(path, content string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, werr := f.WriteString(content)
	cerr := f.Close()
	if werr != nil {
		return werr
	}
	return cerr
}

// ReplaceStatus replaces the `## 현재 상태` section in main.md.
// Mirrors v1 replace_status: updates the 업데이트: timestamp and status block.
func ReplaceStatus(c *Config, label, focus, nextStep string) error {
	if err := EnsureMainMd(c, ""); err != nil {
		return err
	}

	ts := Timestamp()
	mode := c.CurrentMode()

	var elapsed string
	if raw := ReadFile(c.SessionStart); raw != "" {
		var startEpoch int64
		fmt.Sscanf(raw, "%d", &startEpoch)
		elapsed = FormatDuration(time.Now().Unix() - startEpoch)
	} else {
		elapsed = "unknown"
	}

	data, err := os.ReadFile(c.MainMd)
	if err != nil {
		return fmt.Errorf("mirror: read main.md: %w", err)
	}

	out := rewriteStatus(string(data), ts, label, mode, focus, nextStep, elapsed)

	tmp := c.MainMd + ".tmp"
	if err := os.WriteFile(tmp, []byte(out), 0o644); err != nil {
		return fmt.Errorf("mirror: write main.md.tmp: %w", err)
	}
	return os.Rename(tmp, c.MainMd)
}

// rewriteStatus applies the same logic as v1's awk replace_status.
func rewriteStatus(src, ts, label, mode, focus, nextStep, elapsed string) string {
	lines := strings.Split(src, "\n")
	var out []string
	inStatus := false
	replaced := false

	for _, line := range lines {
		// Update the timestamp line.
		if strings.HasPrefix(line, "업데이트:") || strings.HasPrefix(line, "Updated:") {
			out = append(out, "업데이트: "+ts)
			continue
		}
		// Detect start of current-status section.
		if line == "## 현재 상태" || line == "## Current Status" {
			out = append(out, "## 현재 상태")
			out = append(out, "")
			out = append(out, "- 상태: "+label)
			out = append(out, "- 모드: "+mode)
			if focus != "" {
				out = append(out, "- 초점: "+focus)
			}
			if nextStep != "" {
				out = append(out, "- 다음: "+nextStep)
			}
			out = append(out, "- 경과: "+elapsed)
			out = append(out, "")
			inStatus = true
			replaced = true
			continue
		}
		// Skip lines inside the old status section until the next ## heading.
		if inStatus {
			if strings.HasPrefix(line, "## ") {
				inStatus = false
				out = append(out, line)
			}
			// skip old status lines
			continue
		}
		out = append(out, line)
	}

	if !replaced {
		out = append(out, "")
		out = append(out, "## 현재 상태")
		out = append(out, "")
		out = append(out, "- 상태: "+label)
		out = append(out, "- 모드: "+mode)
		if focus != "" {
			out = append(out, "- 초점: "+focus)
		}
		if nextStep != "" {
			out = append(out, "- 다음: "+nextStep)
		}
		out = append(out, "- 경과: "+elapsed)
	}

	return strings.Join(out, "\n")
}

// CreateTurnFile creates the per-turn markdown file (turns/turn-N.md).
func CreateTurnFile(c *Config, n int, title string) error {
	if err := c.EnsureDir(); err != nil {
		return err
	}
	ts := Timestamp()
	content := fmt.Sprintf("# Turn %d: %s\n\n시작: %s\n\n", n, title, ts)
	return os.WriteFile(c.TurnFilePath(n), []byte(content), 0o644)
}

// AppendToTurnFile appends content directly to a specific turn file.
func AppendToTurnFile(c *Config, n int, content string) error {
	return appendToFile(c.TurnFilePath(n), content)
}

// ReadMainMdLines reads main.md into lines for processing.
func ReadMainMdLines(c *Config) ([]string, error) {
	data, err := os.ReadFile(c.MainMd)
	if err != nil {
		return nil, err
	}
	return splitLines(data), nil
}

// splitLines splits bytes into lines, preserving empty lines.
func splitLines(data []byte) []string {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines
}
