package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

const comparisonReportFile = "性能对比表格.md"

type ComparisonSection struct {
	Title    string
	Subtitle string
	Results  []Result
}

func writeComparisonMarkdown(path string, sections []ComparisonSection) error {
	return os.WriteFile(path, []byte(renderComparisonMarkdown(sections)), 0644)
}

func renderComparisonMarkdown(sections []ComparisonSection) string {
	var b strings.Builder
	b.WriteString("# 性能对比表格\n\n")
	b.WriteString("对比对象: `QCP / TCP / KCP`\n\n")
	b.WriteString("> UDP 不纳入对比，因为它非可靠，不适用于游戏业务。\n\n")

	if len(sections) == 0 {
		b.WriteString("_暂无可用数据。_\n")
		return b.String()
	}

	for i, section := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		title := section.Title
		if title == "" {
			title = fmt.Sprintf("Section %d", i+1)
		}
		b.WriteString("## ")
		b.WriteString(title)
		b.WriteString("\n\n")
		if section.Subtitle != "" {
			b.WriteString(section.Subtitle)
			b.WriteString("\n\n")
		}
		b.WriteString("| 协议 | P50 | P95 | P99 | 吞吐量 | 带宽 | 连接数 |")
		b.WriteString("\n|---|---:|---:|---:|---:|---:|---:|")
		for _, r := range orderedComparisonResults(section.Results) {
			b.WriteString("\n| ")
			b.WriteString(strings.ToUpper(r.Protocol))
			b.WriteString(" | ")
			b.WriteString(formatDurationCell(r.Latency.P50))
			b.WriteString(" | ")
			b.WriteString(formatDurationCell(r.Latency.P95))
			b.WriteString(" | ")
			b.WriteString(formatDurationCell(r.Latency.P99))
			b.WriteString(" | ")
			b.WriteString(fmt.Sprintf("%.1f MB/s", r.Throughput))
			b.WriteString(" | ")
			b.WriteString(fmt.Sprintf("%.0f%%", r.Bandwidth))
			b.WriteString(" | ")
			b.WriteString(fmt.Sprintf("%d", r.Connections))
			b.WriteString(" |")
		}
		b.WriteString("\n")
	}

	return b.String()
}

func orderedComparisonResults(results []Result) []Result {
	priority := map[string]int{
		"qcp": 0,
		"tcp": 1,
		"kcp": 2,
	}
	filtered := make([]Result, 0, len(results))
	for _, r := range results {
		if _, ok := priority[strings.ToLower(r.Protocol)]; ok {
			filtered = append(filtered, r)
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return priority[strings.ToLower(filtered[i].Protocol)] < priority[strings.ToLower(filtered[j].Protocol)]
	})
	return filtered
}

func formatDurationCell(d time.Duration) string {
	return d.Round(time.Microsecond).String()
}
