package middlewares

import (
	"fmt"
	"net"
	"net/http"

	"github.com/rs/zerolog/log"
)

// IPChecker инкапсулирует логику проверки IP-адреса на принадлежность к доверенной подсети.
type IPChecker struct {
	trustedSubnet *net.IPNet
}

// NewIPChecker создает новый экземпляр IPChecker.
func NewIPChecker(trustedSubnetCIDR string) (*IPChecker, error) {
	if trustedSubnetCIDR == "" {
		return &IPChecker{trustedSubnet: nil}, nil
	}

	_, ipNet, err := net.ParseCIDR(trustedSubnetCIDR)
	if err != nil {
		return nil, fmt.Errorf("failed to parse trusted subnet CIDR: %w", err)
	}

	return &IPChecker{trustedSubnet: ipNet}, nil
}

// IsAllowed проверяет, разрешен ли доступ для данного IP-адреса.
func (ic *IPChecker) IsAllowed(ip net.IP) bool {
	return ic.trustedSubnet == nil || (ip != nil && ic.trustedSubnet.Contains(ip))
}

type IPCheckMiddleware struct {
	checker *IPChecker
}

func NewIPCheckMiddleware(checker *IPChecker) *IPCheckMiddleware {
	return &IPCheckMiddleware{checker: checker}
}

func (icm *IPCheckMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		realIPStr := r.Header.Get("X-Real-IP")
		if icm.checker.trustedSubnet != nil && realIPStr == "" {
			log.Warn().Msg("X-Real-IP header is missing, but trusted_subnet is configured. Denying access.")
			http.Error(w, "Forbidden: X-Real-IP header is missing", http.StatusForbidden)
			return
		}

		ip := net.ParseIP(realIPStr)
		if !icm.checker.IsAllowed(ip) {
			http.Error(w, "Forbidden: IP address not in trusted subnet", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
