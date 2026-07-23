package netx

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

var (
	_, cgnatNet, _  = net.ParseCIDR("100.64.0.0/10")
	_, docNet1, _   = net.ParseCIDR("192.0.2.0/24")
	_, docNet2, _   = net.ParseCIDR("198.51.100.0/24")
	_, docNet3, _   = net.ParseCIDR("203.0.113.0/24")
	_, testNet, _   = net.ParseCIDR("198.18.0.0/15")
	_, zeroNet, _   = net.ParseCIDR("0.0.0.0/8")
	_, protoNet, _  = net.ParseCIDR("192.0.0.0/24")
	_, futureNet, _ = net.ParseCIDR("240.0.0.0/4")
	_, docIPv6, _   = net.ParseCIDR("2001:db8::/32")
)

func IsPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}

	// AWS metadata IP
	if ip.Equal(net.IPv4(169, 254, 169, 254)) {
		return true
	}

	if cgnatNet.Contains(ip) || docNet1.Contains(ip) || docNet2.Contains(ip) || docNet3.Contains(ip) || testNet.Contains(ip) || zeroNet.Contains(ip) || protoNet.Contains(ip) || futureNet.Contains(ip) || docIPv6.Contains(ip) {
		return true
	}

	return false
}

func NewSafeHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("invalid address: %w", err)
			}

			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("dns lookup failed: %w", err)
			}

			if len(ips) == 0 {
				return nil, fmt.Errorf("no IP addresses found for host %s", host)
			}

			for _, ipAddr := range ips {
				if IsPrivateIP(ipAddr.IP) {
					return nil, fmt.Errorf("connection to private/restricted IP blocked: %s", ipAddr.IP.String())
				}
			}

			target := net.JoinHostPort(ips[0].IP.String(), port)
			return dialer.DialContext(ctx, network, target)
		},
		TLSHandshakeTimeout: 5 * time.Second,
	}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return fmt.Errorf("stopped after 3 redirects")
			}
			host := req.URL.Hostname()
			ips, err := net.LookupIP(host)
			if err != nil {
				return fmt.Errorf("redirect target DNS lookup failed: %w", err)
			}
			for _, ip := range ips {
				if IsPrivateIP(ip) {
					return fmt.Errorf("redirect to private IP blocked: %s", ip.String())
				}
			}
			return nil
		},
	}
}
