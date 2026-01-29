package scout

import (
	"fmt"
	"os"
	"regexp"
	"strconv"

	"gopkg.in/yaml.v3"
)

// patternRe matches brace expansion patterns like "beetle{01..05}".
var patternRe = regexp.MustCompile(`^(.+)\{(\d+)\.\.(\d+)\}$`)

// LoadConfig reads and validates servers.yml from the given path.
func LoadConfig(path string) (*ScoutConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg ScoutConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	if len(cfg.Servers) == 0 {
		return nil, fmt.Errorf("servers.yml must define at least one server")
	}

	// Set default connect timeout
	if cfg.SSH.ConnectTimeout <= 0 {
		cfg.SSH.ConnectTimeout = 10
	}

	// Validate entries
	for i, entry := range cfg.Servers {
		if entry.Name == "" && entry.Pattern == "" {
			return nil, fmt.Errorf("server entry %d must have either 'name' or 'pattern'", i+1)
		}
		if entry.Name != "" && entry.Pattern != "" {
			return nil, fmt.Errorf("server entry %d must have only one of 'name' or 'pattern', not both", i+1)
		}
		if entry.Pattern != "" {
			if !patternRe.MatchString(entry.Pattern) {
				return nil, fmt.Errorf("server entry %d: invalid pattern %q (expected format: prefix{NN..MM})", i+1, entry.Pattern)
			}
		}
	}

	return &cfg, nil
}

// ExpandServers expands all server entries (including brace patterns) into a flat list.
func ExpandServers(cfg *ScoutConfig) ([]Server, error) {
	var servers []Server
	for _, entry := range cfg.Servers {
		if entry.Name != "" {
			servers = append(servers, Server{Name: entry.Name, HasGPU: entry.HasGPU})
			continue
		}

		expanded, err := expandPattern(entry.Pattern)
		if err != nil {
			return nil, err
		}
		if len(expanded) == 0 {
			return nil, fmt.Errorf("pattern %q expanded to zero servers", entry.Pattern)
		}
		for _, name := range expanded {
			servers = append(servers, Server{Name: name, HasGPU: entry.HasGPU})
		}
	}
	return servers, nil
}

// expandPattern expands a brace pattern like "beetle{01..05}" into a list of names.
func expandPattern(pattern string) ([]string, error) {
	matches := patternRe.FindStringSubmatch(pattern)
	if matches == nil {
		return nil, fmt.Errorf("invalid pattern: %q", pattern)
	}

	prefix := matches[1]
	startStr := matches[2]
	endStr := matches[3]

	start, err := strconv.Atoi(startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start in pattern %q: %w", pattern, err)
	}
	end, err := strconv.Atoi(endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end in pattern %q: %w", pattern, err)
	}

	if start > end {
		return nil, fmt.Errorf("pattern %q: start (%d) must be <= end (%d)", pattern, start, end)
	}

	// Determine padding width from the start value string
	padWidth := len(startStr)

	var names []string
	for i := start; i <= end; i++ {
		name := fmt.Sprintf("%s%0*d", prefix, padWidth, i)
		names = append(names, name)
	}
	return names, nil
}
