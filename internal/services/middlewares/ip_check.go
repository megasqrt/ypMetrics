package middlewares

import (
	"net"
	"net/http"

	"github.com/rs/zerolog/log"
)

type IPCheckMiddleware struct {
	trustedSubnet *net.IPNet
}

func NewIPCheckMiddleware(trustedSubnetCIDR string) *IPCheckMiddleware {
	if trustedSubnetCIDR == "" {
		return &IPCheckMiddleware{trustedSubnet: nil}
	}

	_, ipNet, err := net.ParseCIDR(trustedSubnetCIDR)
	if err != nil {
		// This should have been caught at startup, but we handle it defensively.
		log.Fatal().Err(err).Msg("Failed to parse trusted subnet CIDR")
		return &IPCheckMiddleware{trustedSubnet: nil}
	}

	return &IPCheckMiddleware{trustedSubnet: ipNet}
}

func (icm *IPCheckMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if icm.trustedSubnet == nil {
			next.ServeHTTP(w, r)
			return
		}

		realIPStr := r.Header.Get("X-Real-IP")
		if realIPStr == "" {
			log.Warn().Msg("X-Real-IP header is missing, but trusted_subnet is configured. Denying access.")
			http.Error(w, "Forbidden: X-Real-IP header is missing", http.StatusForbidden)
			return
		}

		ip := net.ParseIP(realIPStr)
		if ip == nil || !icm.trustedSubnet.Contains(ip) {
			http.Error(w, "Forbidden: IP address not in trusted subnet", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
