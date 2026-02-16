package state

import (
	"regexp"
	"strconv"
	"strings"
)

// parseReportField extracts the value for a given field name from dokku report output.
// Report format:  "       FieldName:      value"
func parseReportField(output, fieldName string) string {
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		prefix := fieldName + ":"
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}

var exportRegex = regexp.MustCompile(`^export ([^=]+)='(.*)'$`)

// parseExportLines parses `export KEY='VALUE'` lines from config:export output.
func parseExportLines(output string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if m := exportRegex.FindStringSubmatch(line); m != nil {
			result[m[1]] = m[2]
		}
	}
	return result
}

// parseScaleOutput parses ps:scale output with "proctype count" rows.
func parseScaleOutput(output string) map[string]int {
	result := map[string]int{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "----->") ||
			strings.HasPrefix(strings.ToLower(trimmed), "proctype") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 2 {
			if count, err := strconv.Atoi(fields[1]); err == nil {
				result[fields[0]] = count
			}
		}
	}
	return result
}

// parsePortsList parses ports:list output with "scheme host-port container-port" rows.
// Returns map of "scheme:host-port" -> "container-port".
func parsePortsList(output string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "----->") ||
			strings.HasPrefix(trimmed, "scheme") {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 3 {
			key := fields[0] + ":" + fields[1]
			result[key] = fields[2]
		}
	}
	return result
}

// splitCommaList splits a comma-separated string and trims whitespace.
// Returns nil for empty input.
func splitCommaList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// parseAppsList parses apps:list output, skipping the header line.
func parseAppsList(output string) []string {
	var apps []string
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "=====>") {
			continue
		}
		apps = append(apps, trimmed)
	}
	return apps
}

// parseServiceList parses <type>:list output, skipping the header line.
// Header format: "NAME  VERSION  STATUS"
func parseServiceList(output string) []string {
	var services []string
	first := true
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if first {
			first = false
			continue // skip header
		}
		fields := strings.Fields(trimmed)
		if len(fields) >= 1 {
			services = append(services, fields[0])
		}
	}
	return services
}
