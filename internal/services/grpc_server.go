package services

import (
	"io"

	"github.com/rs/zerolog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"ypMetrics/internal/store"
	"ypMetrics/models"
	pb "ypMetrics/proto"
)

// MetricGRPCServer реализует gRPC сервер для метрик.
type MetricGRPCServer struct {
	pb.UnimplementedMetricsServer
	storage store.Storage
	log     zerolog.Logger
}

// NewMetricGRPCServer создает новый экземпляр MetricGRPCServer.
func NewMetricGRPCServer(storage store.Storage, log zerolog.Logger) *MetricGRPCServer {
	return &MetricGRPCServer{
		storage: storage,
		log:     log,
	}
}

// Update обрабатывает потоковую передачу метрик от агента.
func (s *MetricGRPCServer) Update(stream grpc.ClientStreamingServer[pb.Metric, pb.UpdateResponse]) error {
	var metricsToUpdate []models.Metrics
	for {
		metric, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			s.log.Error().Err(err).Msg("Error receiving metric from stream")
			return status.Errorf(codes.Internal, "cannot receive stream message: %v", err)
		}

		s.log.Info().
			Str("id", metric.GetId()).
			Str("type", metric.GetType()).
			Msg("Received metric via gRPC")

		m := models.Metrics{
			ID:    metric.GetId(),
			MType: metric.GetType(),
		}

		switch metric.GetType() {
		case "gauge":
			v := metric.GetValue()
			m.Value = &v
		case "counter":
			v := metric.GetDelta()
			m.Delta = &v
		default:
			s.log.Warn().Str("type", metric.GetType()).Msg("Unknown metric type received via gRPC")
			continue
		}
		metricsToUpdate = append(metricsToUpdate, m)
	}

	if len(metricsToUpdate) > 0 {
		err := s.storage.UpdateMetricsBatch(stream.Context(), metricsToUpdate)
		if err != nil {
			s.log.Error().Err(err).Msg("Error saving metrics from gRPC stream")
			return status.Errorf(codes.Internal, "cannot save metrics: %v", err)
		}
	}

	if err := stream.SendAndClose(&pb.UpdateResponse{}); err != nil {
		return status.Errorf(codes.Internal, "cannot send response: %v", err)
	}

	return nil
}
