package middlewares

import (
	"net"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

func GrpcCheckMiddleware(checker *IPChecker) grpc.StreamServerInterceptor {
	if checker.trustedSubnet == nil {
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		p, ok := peer.FromContext(ss.Context())
		if !ok {
			return status.Error(codes.Internal, "could not get peer from context")
		}

		var ip net.IP
		if tcpAddr, ok := p.Addr.(*net.TCPAddr); ok {
			ip = tcpAddr.IP
		} else {
			host, _, err := net.SplitHostPort(p.Addr.String())
			if err != nil {
				return status.Errorf(codes.Internal, "could not parse peer address: %v", err)
			}
			ip = net.ParseIP(host)
		}

		if ip == nil {
			return status.Errorf(codes.Internal, "could not parse peer IP address")
		}

		if !checker.IsAllowed(ip) {
			log.Warn().Str("ip", ip.String()).Msg("gRPC request from non-trusted IP")
			return status.Error(codes.PermissionDenied, "forbidden")
		}

		return handler(srv, ss)
	}
}
