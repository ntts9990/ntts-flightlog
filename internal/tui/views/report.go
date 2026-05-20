package views

import "strings"

// RenderReport returns the report view content.
// Placeholder: B4 worker fills in actual metric report rendering.
// This view will display the 5-metric analytics summary once B3/B4 are complete.
func RenderReport(data *WorklogData) string {
	var sb strings.Builder

	sb.WriteString(Bold + ColorSection + "## 리포트" + Reset + "\n\n")
	sb.WriteString(Dim + "리포트 뷰는 Phase B4에서 구현됩니다." + Reset + "\n")
	sb.WriteString(Dim + "5개 메트릭 SQL 뷰가 완성되면 이 화면에 분석 결과가 표시됩니다." + Reset + "\n\n")

	if data != nil {
		sb.WriteString(Bold + "현재 데이터 요약:" + Reset + "\n")
		sb.WriteString(Dim + formatCount("세션", len(data.Sessions)) + Reset + "\n")
		sb.WriteString(Dim + formatCount("턴", len(data.Turns)) + Reset + "\n")
		sb.WriteString(Dim + formatCount("엔트리", len(data.Entries)) + Reset + "\n")

		// Count by kind.
		counts := make(map[string]int)
		for _, e := range data.Entries {
			counts[e.Kind]++
		}
		if n := counts["decision"]; n > 0 {
			sb.WriteString(Dim + formatCount("  결정", n) + Reset + "\n")
		}
		if n := counts["evidence"]; n > 0 {
			sb.WriteString(Dim + formatCount("  근거", n) + Reset + "\n")
		}
		if n := counts["blocker"]; n > 0 {
			sb.WriteString(Dim + formatCount("  블로커", n) + Reset + "\n")
		}
	}

	return sb.String()
}

func formatCount(label string, n int) string {
	return label + ": " + itoa(n) + "개"
}

// itoa converts an int to a decimal string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 10)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
