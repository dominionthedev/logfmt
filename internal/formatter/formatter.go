package formatter

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	styleTime = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	styleLevelDebug = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Bold(true)
	styleLevelInfo  = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	styleLevelWarn  = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	styleLevelError = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	styleLevelFatal = lipgloss.NewStyle().Foreground(lipgloss.Color("197")).Bold(true).Underline(true)

	styleKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("111"))
	styleVal     = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	styleMsg     = lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Bold(true)
	styleSep     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	styleUnknown = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true)
)

// ── Level ─────────────────────────────────────────────────────────────────────

func formatLevel(level string) string {
	upper := strings.ToUpper(strings.TrimSpace(level))
	switch upper {
	case "DEBUG", "DBG", "TRACE":
		return styleLevelDebug.Render(fmt.Sprintf("%-5s", upper))
	case "INFO", "INF":
		return styleLevelInfo.Render(fmt.Sprintf("%-5s", upper))
	case "WARN", "WRN", "WARNING":
		return styleLevelWarn.Render(fmt.Sprintf("%-5s", "WARN"))
	case "ERROR", "ERR":
		return styleLevelError.Render(fmt.Sprintf("%-5s", "ERROR"))
	case "FATAL", "CRIT", "CRITICAL":
		return styleLevelFatal.Render(fmt.Sprintf("%-5s", "FATAL"))
	default:
		return styleLevelInfo.Render(fmt.Sprintf("%-5s", upper))
	}
}

// ── Time ──────────────────────────────────────────────────────────────────────

var timeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006/01/02 15:04:05",
	time.UnixDate,
}

func formatTime(ts string) string {
	for _, layout := range timeFormats {
		if t, err := time.Parse(layout, ts); err == nil {
			return styleTime.Render(t.Format("15:04:05.000"))
		}
	}
	return styleTime.Render(ts)
}

// ── KV pairs ─────────────────────────────────────────────────────────────────

func formatKV(key, val string) string {
	return styleKey.Render(key) +
		styleSep.Render("=") +
		styleVal.Render(val)
}

// ── Format dispatch ───────────────────────────────────────────────────────────

// Options holds runtime flags passed from cobra.
type Options struct {
	Filter    string // only show lines containing this string
	LevelMin  string // minimum level to show (debug|info|warn|error)
	NoColor   bool
	TimeOnly  bool // suppress KV pairs, show time+level+msg only
}

func levelRank(l string) int {
	switch strings.ToUpper(strings.TrimSpace(l)) {
	case "DEBUG", "DBG", "TRACE":
		return 0
	case "INFO", "INF":
		return 1
	case "WARN", "WRN", "WARNING":
		return 2
	case "ERROR", "ERR":
		return 3
	case "FATAL", "CRIT", "CRITICAL":
		return 4
	}
	return 1
}

// FormatLine takes a raw log line and returns a formatted string (or "" to skip).
func FormatLine(raw string, opts Options) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Filter
	if opts.Filter != "" && !strings.Contains(strings.ToLower(raw), strings.ToLower(opts.Filter)) {
		return ""
	}

	// Try JSON first, then logfmt, prefix (LEVEL: msg), then plain
	if strings.HasPrefix(raw, "{") {
		return formatJSON(raw, opts)
	}
	if strings.Contains(raw, "=") {
		return formatLogfmt(raw, opts)
	}
	if level, msg, ok := tryPrefixFormat(raw); ok {
		if opts.LevelMin != "" && levelRank(level) < levelRank(opts.LevelMin) {
			return ""
		}
		parts := []string{formatLevel(level)}
		if msg != "" {
			parts = append(parts, styleMsg.Render(msg))
		}
		return strings.Join(parts, " ")
	}
	return formatPlain(raw, opts)
}

// ── JSON ──────────────────────────────────────────────────────────────────────

var jsonLevelKeys = []string{"level", "lvl", "severity", "loglevel"}
var jsonMsgKeys = []string{"msg", "message", "text", "body"}
var jsonTimeKeys = []string{"time", "ts", "timestamp", "t", "@timestamp"}

func jsonFind(m map[string]any, keys []string) (string, string) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return k, fmt.Sprintf("%v", v)
		}
	}
	return "", ""
}

func formatJSON(raw string, opts Options) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return formatPlain(raw, opts)
	}

	levelKey, level := jsonFind(m, jsonLevelKeys)
	msgKey, msg := jsonFind(m, jsonMsgKeys)
	timeKey, ts := jsonFind(m, jsonTimeKeys)

	// Level filter
	if opts.LevelMin != "" && levelRank(level) < levelRank(opts.LevelMin) {
		return ""
	}

	var parts []string

	if ts != "" {
		parts = append(parts, formatTime(ts))
	}
	if level != "" {
		parts = append(parts, formatLevel(level))
	}
	if msg != "" {
		parts = append(parts, styleMsg.Render(msg))
	}

	if !opts.TimeOnly {
		skip := map[string]bool{levelKey: true, msgKey: true, timeKey: true}
		for k, v := range m {
			if skip[k] {
				continue
			}
			parts = append(parts, formatKV(k, fmt.Sprintf("%v", v)))
		}
	}

	return strings.Join(parts, " ")
}

// ── logfmt ────────────────────────────────────────────────────────────────────

// Simple logfmt parser (key=value, key="quoted value")
func parseLogfmt(s string) []struct{ k, v string } {
	var pairs []struct{ k, v string }
	s = strings.TrimSpace(s)
	for s != "" {
		// find key
		eqIdx := strings.IndexByte(s, '=')
		if eqIdx < 0 {
			// bare word
			pairs = append(pairs, struct{ k, v string }{s, ""})
			break
		}
		key := strings.TrimSpace(s[:eqIdx])
		s = s[eqIdx+1:]

		var val string
		if strings.HasPrefix(s, `"`) {
			// quoted
			end := strings.Index(s[1:], `"`)
			if end < 0 {
				val = s[1:]
				s = ""
			} else {
				val = s[1 : end+1]
				s = strings.TrimSpace(s[end+2:])
			}
		} else {
			spIdx := strings.IndexByte(s, ' ')
			if spIdx < 0 {
				val = s
				s = ""
			} else {
				val = s[:spIdx]
				s = strings.TrimSpace(s[spIdx+1:])
			}
		}
		pairs = append(pairs, struct{ k, v string }{key, val})
	}
	return pairs
}

func formatLogfmt(raw string, opts Options) string {
	pairs := parseLogfmt(raw)

	var level, msg, ts string
	kvMap := map[string]string{}

	for _, p := range pairs {
		switch strings.ToLower(p.k) {
		case "level", "lvl", "severity":
			level = p.v
		case "msg", "message":
			msg = p.v
		case "time", "ts", "timestamp", "t":
			ts = p.v
		default:
			kvMap[p.k] = p.v
		}
	}

	if opts.LevelMin != "" && levelRank(level) < levelRank(opts.LevelMin) {
		return ""
	}

	var parts []string
	if ts != "" {
		parts = append(parts, formatTime(ts))
	}
	if level != "" {
		parts = append(parts, formatLevel(level))
	}
	if msg != "" {
		parts = append(parts, styleMsg.Render(msg))
	}
	if !opts.TimeOnly {
		for k, v := range kvMap {
			parts = append(parts, formatKV(k, v))
		}
	}

	if len(parts) == 0 {
		return formatPlain(raw, opts)
	}
	return strings.Join(parts, " ")
}

// ── Plain ─────────────────────────────────────────────────────────────────────

func formatPlain(raw string, opts Options) string {
	return styleUnknown.Render(raw)
}

// ── Prefix format (LEVEL: message) ───────────────────────────────────────────

// knownLevels maps prefixes we recognise to their canonical level string.
var knownLevelPrefixes = []string{
	"TRACE", "DEBUG", "DBG", "INFO", "INF",
	"WARN", "WARNING", "WRN", "ERROR", "ERR",
	"FATAL", "CRIT", "CRITICAL",
}

// tryPrefixFormat detects lines like:
//
//	INFO: message
//	[ERROR] message
//	DEBUG - message
//	WARN  message
func tryPrefixFormat(raw string) (level, msg string, ok bool) {
	upper := strings.ToUpper(raw)
	for _, lvl := range knownLevelPrefixes {
		if !strings.HasPrefix(upper, lvl) {
			continue
		}
		rest := raw[len(lvl):]
		// Accept separators: ": ", " - ", "] ", "  ", " "
		rest = strings.TrimLeft(rest, ": -][ \t")
		if rest == "" && len(raw) == len(lvl) {
			rest = ""
		}
		return lvl, rest, true
	}
	return "", "", false
}
