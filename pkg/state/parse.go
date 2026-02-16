package state

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"

	"github.com/deanmarano/dokkufile/pkg/schema"
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

// parseAppJSON parses an app.json file and extracts healthchecks and cron jobs.
func parseAppJSON(content string) (map[string][]schema.HealthcheckConfig, []schema.CronJob) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, nil
	}

	var healthchecks map[string][]schema.HealthcheckConfig
	if hcRaw, ok := raw["healthchecks"]; ok {
		if err := json.Unmarshal(hcRaw, &healthchecks); err != nil {
			healthchecks = nil
		}
	}

	var cron []schema.CronJob
	if cronRaw, ok := raw["cron"]; ok {
		if err := json.Unmarshal(cronRaw, &cron); err != nil {
			cron = nil
		}
	}

	return healthchecks, cron
}

// parseReportConfigFields extracts key=value config fields from a report-style output.
// Looks for lines like "  Config key:  value" and returns a map.
func parseReportConfigFields(output string) map[string]string {
	result := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "Config ") {
			continue
		}
		rest := strings.TrimPrefix(trimmed, "Config ")
		idx := strings.Index(rest, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(rest[:idx])
		val := strings.TrimSpace(rest[idx+1:])
		if key != "" && val != "" {
			result[key] = val
		}
	}
	return result
}

// parseOIDCClients parses auth:oidc:list output into OIDCClient structs.
// Expected format: "ID  SECRET  REDIRECT_URI" rows after a header.
func parseOIDCClients(output string) []schema.OIDCClient {
	var clients []schema.OIDCClient
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
			client := schema.OIDCClient{ID: fields[0]}
			if len(fields) >= 2 {
				client.Secret = fields[1]
			}
			if len(fields) >= 3 {
				client.RedirectURI = fields[2]
			}
			clients = append(clients, client)
		}
	}
	return clients
}
