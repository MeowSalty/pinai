package proxy

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"syscall"
	"time"
)

var (
	errSchemeNotAllowed = errors.New("仅允许 http 和 https 协议")
	errPrivateAddress   = errors.New("目标地址为私有或保留地址")
)

var privateCIDRs []*net.IPNet

func init() {
	cidrs := []string{
		// IPv4
		"0.0.0.0/8",
		"10.0.0.0/8",
		"100.64.0.0/10",
		"127.0.0.0/8",
		"169.254.0.0/16",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"198.18.0.0/15",
		// IPv6
		"::1/128",
		"fc00::/7",
		"fe80::/10",
	}
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			panic("无法解析 CIDR: " + cidr)
		}
		privateCIDRs = append(privateCIDRs, ipNet)
	}
}

// validateURLScheme 校验 URL 仅使用 http 或 https 协议。
func validateURLScheme(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errSchemeNotAllowed
	}
	return nil
}

// isPrivateIP 检查 IP 是否属于私有或保留地址段。
func isPrivateIP(ip net.IP) bool {
	// To4() 归一化，覆盖 ::ffff:127.0.0.1 等 IPv4-mapped IPv6
	if ip4 := ip.To4(); ip4 != nil {
		ip = ip4
	}
	for _, cidr := range privateCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// checkConnection 是 net.Dialer.Control 回调，在 TCP connect 前校验已解析的 IP。
func checkConnection(network, addr string, conn syscall.RawConn) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return errors.New("无法解析 IP 地址: " + host)
	}
	if isPrivateIP(ip) {
		return errPrivateAddress
	}
	return nil
}

// newSafeClient 构建带 SSRF 防护的 HTTP Client。
func newSafeClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   checkConnection,
	}
	transport := &http.Transport{
		DialContext: dialer.DialContext,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
				return errSchemeNotAllowed
			}
			if len(via) >= 10 {
				return errors.New("重定向次数过多")
			}
			return nil
		},
	}
}
