package domain

import (
	"net"
	"strings"
)

// BlockedHost reports whether a URL hostname must never be monitored.
// Covers literals (loopback, private, link-local incl. cloud metadata
// 169.254.169.254, unspecified, multicast) and localhost names.
// Non-literal hostnames pass here; infrastructure re-checks the resolved
// IPs at check time (DNS TOCTOU-safe), with PULSE_ALLOW_HOSTS exceptions.
func BlockedHost(host string) bool {
	h := strings.ToLower(strings.TrimSpace(host))
	if h == "localhost" || strings.HasSuffix(h, ".localhost") {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return BlockedIP(ip)
	}
	return false
}

// BlockedIP reports whether a resolved IP is off-limits for monitoring.
func BlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast()
}
