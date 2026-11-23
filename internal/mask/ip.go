package mask

import (
	"net"
	"path/filepath"
	"strings"
)

// ipSkipRule holds a parsed skip rule for IP masking.
type ipSkipRule struct {
	single   net.IP
	cidr     *net.IPNet
	wildcard string
}

// parseIPSkipRules parses raw skip string rules into structured IP skip rules.
func parseIPSkipRules(rawRules []string) []ipSkipRule {
	rules := make([]ipSkipRule, 0, len(rawRules))
	for _, raw := range rawRules {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || trimmed == "false" || trimmed == "true" {
			continue
		}
		if strings.Contains(trimmed, "/") {
			if _, cidr, err := net.ParseCIDR(trimmed); err == nil {
				rules = append(rules, ipSkipRule{cidr: cidr})
				continue
			}
		}
		if strings.Contains(trimmed, "*") {
			rules = append(rules, ipSkipRule{wildcard: trimmed})
			continue
		}
		if ip := net.ParseIP(trimmed); ip != nil {
			rules = append(rules, ipSkipRule{single: ip})
			continue
		}
	}
	return rules
}

// matchIPWildcard checks if an IP string matches a wildcard pattern such as 192.168.*.*.
func matchIPWildcard(pattern string, ipStr string) bool {
	if !strings.Contains(pattern, "*") {
		return false
	}
	patternParts := strings.Split(pattern, ".")
	ipParts := strings.Split(ipStr, ".")
	if len(patternParts) == len(ipParts) {
		matched := true
		for i := range patternParts {
			if patternParts[i] != "*" && patternParts[i] != ipParts[i] {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	matched, err := filepath.Match(pattern, ipStr)
	return err == nil && matched
}

// isIPSkipped returns true if the parsed IP matches any of the skip rules.
func isIPSkipped(ip net.IP, ipStr string, rules []ipSkipRule) bool {
	for _, rule := range rules {
		if rule.single != nil && rule.single.Equal(ip) {
			return true
		}
		if rule.cidr != nil && rule.cidr.Contains(ip) {
			return true
		}
		if rule.wildcard != "" && matchIPWildcard(rule.wildcard, ipStr) {
			return true
		}
	}
	return false
}

func isHexOrColon(b byte) bool {
	return b == ':' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')
}

// maskIPAddress handles unified IPv4 and IPv6 masking.
func maskIPAddress(match string, skipRules []ipSkipRule) string {
	prefix := ""
	clean := match
	if len(match) > 0 && !isHexOrColon(match[0]) {
		prefix = string(match[0])
		clean = match[1:]
	}
	ip := net.ParseIP(clean)
	if ip == nil {
		return match
	}
	if isIPSkipped(ip, clean, skipRules) {
		return match
	}
	return prefix + "[MASKED_IP]"
}
