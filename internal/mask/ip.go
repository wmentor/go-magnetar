package mask

import (
	"net"
	"strings"
)

const (
	// ipv4Pattern matches candidate IPv4 addresses; net.ParseIP validates octets (0-255).
	ipv4Pattern = `\b(?:\d{1,3}\.){3}\d{1,3}\b`
	// ipv6Pattern matches candidate IPv6 addresses (full, compressed, or loopback).
	ipv6Pattern = `(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,5}(?::[0-9a-fA-F]{1,4}){1,2}|(?:[0-9a-fA-F]{1,4}:){1,4}(?::[0-9a-fA-F]{1,4}){1,3}|(?:[0-9a-fA-F]{1,4}:){1,3}(?::[0-9a-fA-F]{1,4}){1,4}|(?:[0-9a-fA-F]{1,4}:){1,2}(?::[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(?::[0-9a-fA-F]{1,4}){1,6}|:(?::[0-9a-fA-F]{1,4}){1,7}|(?:[0-9a-fA-F]{1,4}:){1,7}:|::`
)

type ipSkipRule struct {
	ip       net.IP
	network  *net.IPNet
	wildcard []string
}

func parseIPSkipRules(entries []string) []ipSkipRule {
	rules := make([]ipSkipRule, 0, len(entries))
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "*") {
			parts := strings.Split(entry, ".")
			if len(parts) == 4 {
				rules = append(rules, ipSkipRule{wildcard: parts})
			}
			continue
		}
		if _, network, err := net.ParseCIDR(entry); err == nil {
			rules = append(rules, ipSkipRule{network: network})
			continue
		}
		if ip := net.ParseIP(entry); ip != nil {
			rules = append(rules, ipSkipRule{ip: ip})
		}
	}
	return rules
}

func (r ipSkipRule) matches(ip net.IP) bool {
	if r.ip != nil {
		return r.ip.Equal(ip)
	}
	if r.network != nil {
		return r.network.Contains(ip)
	}
	if len(r.wildcard) == 4 {
		v4 := ip.To4()
		if v4 == nil {
			return false
		}
		parts := strings.Split(v4.String(), ".")
		if len(parts) != 4 {
			return false
		}
		for i := 0; i < 4; i++ {
			if r.wildcard[i] != "*" && r.wildcard[i] != parts[i] {
				return false
			}
		}
		return true
	}
	return false
}

func maskIP(candidate string, skipRules []ipSkipRule) string {
	ip := net.ParseIP(candidate)
	if ip == nil {
		return candidate
	}
	for _, rule := range skipRules {
		if rule.matches(ip) {
			return candidate
		}
	}
	return "[MASKED_IP]"
}
