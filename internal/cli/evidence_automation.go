package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var phaseEMetrics = []string{
	"turn_duration",
	"blocker_accumulation",
	"agent_completion",
	"agent_blocker_freq",
	"evidence_bound_decisions",
}

func newEvidenceCheckCmd() *cobra.Command {
	var strict bool
	var format string
	var root string
	cmd := &cobra.Command{
		Use:   "evidence-check",
		Short: "Phase E evidence readiness를 점검합니다",
		Long:  "Phase E/GA evidence 문서와 소스 artifact 상태를 read-only로 점검합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if format != "text" && format != "json" {
				return fmt.Errorf("--format must be text or json, got %q", format)
			}
			snap := checkPhaseEvidence(root, strict)
			if format == "json" {
				raw, err := json.MarshalIndent(snap, "", "  ")
				if err != nil {
					return fmt.Errorf("evidence-check json: %w", err)
				}
				cmd.Println(string(raw))
			} else {
				cmd.Print(renderEvidenceCheck(snap))
			}
			if strict && !snap.OK {
				return fmt.Errorf("evidence-check strict failed")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&strict, "strict", false, "GA-blocking strict mode")
	cmd.Flags().StringVar(&format, "format", "text", "출력 형식: text|json")
	cmd.Flags().StringVar(&root, "root", ".", "프로젝트 루트")
	return cmd
}

func newEvidenceReportCmd() *cobra.Command {
	var persona string
	var format string
	var root string
	cmd := &cobra.Command{
		Use:   "evidence-report",
		Short: "persona별 evidence report를 출력합니다",
		Long:  "self-retro, agent-operator, team-share persona의 Phase E evidence 상태와 다음 작업을 출력합니다.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if persona != "self-retro" && persona != "agent-operator" && persona != "team-share" {
				return fmt.Errorf("--persona must be self-retro, agent-operator, or team-share, got %q", persona)
			}
			if format != "text" && format != "json" {
				return fmt.Errorf("--format must be text or json, got %q", format)
			}
			report := buildEvidenceReport(root, persona)
			if format == "json" {
				raw, err := json.MarshalIndent(report, "", "  ")
				if err != nil {
					return fmt.Errorf("evidence-report json: %w", err)
				}
				cmd.Println(string(raw))
			} else {
				cmd.Print(renderEvidenceReport(report))
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&persona, "persona", "self-retro", "persona: self-retro|agent-operator|team-share")
	cmd.Flags().StringVar(&format, "format", "text", "출력 형식: text|json")
	cmd.Flags().StringVar(&root, "root", ".", "프로젝트 루트")
	return cmd
}

type evidenceCheckSnapshot struct {
	OK        bool                  `json:"ok"`
	Mode      string                `json:"mode"`
	Root      string                `json:"root"`
	Checks    []evidenceCheckResult `json:"checks"`
	Summary   evidenceCheckSummary  `json:"summary"`
	NextSteps []string              `json:"next_steps"`
}

type evidenceCheckResult struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type evidenceCheckSummary struct {
	Failures               int `json:"failures"`
	Warnings               int `json:"warnings"`
	PlaceholderCount       int `json:"placeholder_count"`
	AlphaDatedEntries      int `json:"alpha_dated_entries"`
	ChangedByMetricCount   int `json:"changed_by_metric_count"`
	PersonaSectionsPassing int `json:"persona_sections_passing"`
}

type evidenceReport struct {
	Persona       string               `json:"persona"`
	Source        string               `json:"source"`
	AcceptanceDoc string               `json:"acceptance_doc"`
	Metrics       []evidenceMetricLine `json:"metrics"`
	Placeholders  []string             `json:"placeholders"`
	NextAction    string               `json:"next_action"`
}

type evidenceMetricLine struct {
	Metric  string `json:"metric"`
	Present bool   `json:"present"`
	Line    string `json:"line,omitempty"`
}

func checkPhaseEvidence(root string, strict bool) evidenceCheckSnapshot {
	root = cleanRoot(root)
	snap := evidenceCheckSnapshot{OK: true, Root: root}
	if strict {
		snap.Mode = "strict"
	} else {
		snap.Mode = "advisory"
	}
	required := []string{
		".omc/specs/alpha-dogfood-log.md",
		".omc/specs/v2-agent-operator-decisions.md",
		".omc/specs/v2-team-share-report.md",
		".omc/specs/v2-adversarial-review.md",
		"docs/v2-ga-acceptance-evidence.md",
		"docs/phase-e-persona-recruitment.md",
		"docs/adversarial-review-framework.md",
		"docs/e0-3-agent-tmux-sanity.md",
	}
	for _, path := range required {
		snap.addCheck("file:"+path, fileExists(root, path), path)
	}
	acceptance := readRootFile(root, "docs/v2-ga-acceptance-evidence.md")
	alpha := readRootFile(root, ".omc/specs/alpha-dogfood-log.md")
	snap.Summary.PlaceholderCount = countPattern(acceptance+"\n"+alpha, `(?i)TODO|_to be filled|placeholder`)
	snap.Summary.AlphaDatedEntries = countPattern(alpha, `(?m)^### [0-9]{4}-[0-9]{2}-[0-9]{2}`)
	snap.Summary.ChangedByMetricCount = countPattern(alpha, `\[CHANGED-BY-METRIC: [a-z_]+\]`)
	passSections := 0
	for _, persona := range []string{"Self-Retro", "Agent-Operator", "Team-Share"} {
		count := personaMetricCount(acceptance, persona)
		ok := count >= 4
		if ok {
			passSections++
		}
		snap.addCheck("persona:"+persona, ok, fmt.Sprintf("%d/5 metrics cited", count))
	}
	snap.Summary.PersonaSectionsPassing = passSections
	if snap.Summary.PlaceholderCount > 0 {
		if strict {
			snap.addCheck("placeholders", false, fmt.Sprintf("%d placeholders remain", snap.Summary.PlaceholderCount))
		} else {
			snap.Summary.Warnings++
			snap.NextSteps = append(snap.NextSteps, "Replace placeholder evidence in docs/v2-ga-acceptance-evidence.md before strict GA.")
		}
	}
	if strict {
		snap.addCheck("alpha_dated_entries", snap.Summary.AlphaDatedEntries >= 12, fmt.Sprintf("%d entries", snap.Summary.AlphaDatedEntries))
		snap.addCheck("changed_by_metric", snap.Summary.ChangedByMetricCount >= 1, fmt.Sprintf("%d entries", snap.Summary.ChangedByMetricCount))
		snap.addCheck("external_ack", regexp.MustCompile(`(?i)external .*ack|acknowledg`).MatchString(acceptance), "external acknowledgement reference")
		snap.addCheck("adversarial_review", regexp.MustCompile(`(?i)adversarial review`).MatchString(acceptance), "adversarial review reference")
	}
	if len(snap.NextSteps) == 0 {
		snap.NextSteps = append(snap.NextSteps, "Run ntts-flightlog evidence-report --persona team-share for the next concrete artifact gap.")
	}
	return snap
}

func (s *evidenceCheckSnapshot) addCheck(name string, ok bool, detail string) {
	s.Checks = append(s.Checks, evidenceCheckResult{Name: name, OK: ok, Detail: detail})
	if !ok {
		s.OK = false
		s.Summary.Failures++
	}
}

func buildEvidenceReport(root, persona string) evidenceReport {
	root = cleanRoot(root)
	source := map[string]string{
		"self-retro":     ".omc/specs/alpha-dogfood-log.md",
		"agent-operator": ".omc/specs/v2-agent-operator-decisions.md",
		"team-share":     ".omc/specs/v2-team-share-report.md",
	}[persona]
	section := map[string]string{
		"self-retro":     "Self-Retro",
		"agent-operator": "Agent-Operator",
		"team-share":     "Team-Share",
	}[persona]
	docPath := "docs/v2-ga-acceptance-evidence.md"
	acceptance := readRootFile(root, docPath)
	report := evidenceReport{Persona: persona, Source: source, AcceptanceDoc: docPath}
	text := extractMarkdownSection(acceptance, section)
	for _, metric := range phaseEMetrics {
		line := findMetricLine(text, metric)
		report.Metrics = append(report.Metrics, evidenceMetricLine{Metric: metric, Present: line != "", Line: line})
	}
	for _, line := range strings.Split(text, "\n") {
		if regexp.MustCompile(`(?i)TODO|placeholder|_to be filled`).MatchString(line) {
			report.Placeholders = append(report.Placeholders, strings.TrimSpace(line))
		}
	}
	report.NextAction = nextEvidenceAction(report)
	return report
}

func renderEvidenceCheck(snap evidenceCheckSnapshot) string {
	var b strings.Builder
	fmt.Fprintf(&b, "NTTS Flightlog evidence-check\nmode: %s\nroot: %s\n\n", snap.Mode, snap.Root)
	for _, check := range snap.Checks {
		status := "PASS"
		if !check.OK {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "- %s %s: %s\n", status, check.Name, check.Detail)
	}
	fmt.Fprintf(&b, "\nsummary: failures %d, warnings %d, placeholders %d\n", snap.Summary.Failures, snap.Summary.Warnings, snap.Summary.PlaceholderCount)
	b.WriteString("next:\n")
	for _, step := range snap.NextSteps {
		fmt.Fprintf(&b, "- %s\n", step)
	}
	return b.String()
}

func renderEvidenceReport(report evidenceReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "NTTS Flightlog evidence-report\npersona: %s\nsource: %s\n\n", report.Persona, report.Source)
	for _, metric := range report.Metrics {
		status := "missing"
		if metric.Present {
			status = "present"
		}
		fmt.Fprintf(&b, "- %s: %s", metric.Metric, status)
		if metric.Line != "" {
			fmt.Fprintf(&b, " — %s", metric.Line)
		}
		b.WriteByte('\n')
	}
	if len(report.Placeholders) > 0 {
		b.WriteString("\nplaceholders:\n")
		for _, item := range report.Placeholders {
			fmt.Fprintf(&b, "- %s\n", item)
		}
	}
	fmt.Fprintf(&b, "\nnext: %s\n", report.NextAction)
	return b.String()
}

func cleanRoot(root string) string {
	if root == "" {
		root = "."
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return root
	}
	return abs
}

func fileExists(root, path string) bool {
	_, err := os.Stat(filepath.Join(root, path))
	return err == nil
}

func readRootFile(root, path string) string {
	data, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		return ""
	}
	return string(data)
}

func countPattern(text, pattern string) int {
	return len(regexp.MustCompile(pattern).FindAllString(text, -1))
}

func personaMetricCount(doc, section string) int {
	text := extractMarkdownSection(doc, section)
	count := 0
	for _, metric := range phaseEMetrics {
		if findMetricLine(text, metric) != "" {
			count++
		}
	}
	return count
}

func extractMarkdownSection(doc, section string) string {
	inSection := false
	var lines []string
	for _, line := range strings.Split(doc, "\n") {
		if strings.HasPrefix(line, "## ") {
			if inSection {
				break
			}
			inSection = strings.Contains(line, section)
			continue
		}
		if inSection {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

func findMetricLine(text, metric string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.Contains(line, metric) || metricAliasRegexp(metric).MatchString(line) {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func metricAliasRegexp(metric string) *regexp.Regexp {
	switch metric {
	case "turn_duration":
		return regexp.MustCompile(`(?i)turn 소요시간|turn duration|turn elapsed`)
	case "blocker_accumulation":
		return regexp.MustCompile(`(?i)blocker 누적|blocker accumulation|차단 시간`)
	case "agent_completion":
		return regexp.MustCompile(`(?i)agent 완료율|agent completion|완료율`)
	case "agent_blocker_freq":
		return regexp.MustCompile(`(?i)agent blocker 빈도|blocker 빈도`)
	default:
		return regexp.MustCompile(`(?i)evidence-bound|evidence가 붙은|evidence bound`)
	}
}

func nextEvidenceAction(report evidenceReport) string {
	for _, item := range report.Metrics {
		if !item.Present {
			return fmt.Sprintf("Add concrete %s evidence to %s and cite the source artifact.", item.Metric, report.AcceptanceDoc)
		}
	}
	if len(report.Placeholders) > 0 {
		return "Replace placeholder lines with dated concrete evidence and rerun evidence-check --strict."
	}
	if report.Persona == "team-share" {
		return "Add dated external acknowledgement after sharing ntts-flightlog share --window week --format md."
	}
	return "Run evidence-check --strict and address any remaining GA gate failures."
}
