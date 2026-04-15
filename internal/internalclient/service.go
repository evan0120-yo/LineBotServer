package internalclient

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"linebot-backend/internal/infra"
	grpcpb "linebot-backend/internal/internalclient/pb"
)

// Service wraps Internal AI Copilot gRPC client.
type Service struct {
	conn   *grpc.ClientConn
	client grpcpb.IntegrationServiceClient
}

// NewService creates a new Internal AI Copilot gRPC client.
func NewService(grpcAddr string) (*Service, error) {
	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("grpc.NewClient: %w", err)
	}

	client := grpcpb.NewIntegrationServiceClient(conn)

	return &Service{
		conn:   conn,
		client: client,
	}, nil
}

// Close closes the gRPC connection.
func (s *Service) Close() error {
	return s.conn.Close()
}

// LineTaskConsult calls Internal AI Copilot LineTaskConsult gRPC endpoint.
func (s *Service) LineTaskConsult(ctx context.Context, command LineTaskConsultCommand) (LineTaskConsultResult, error) {
	request := &grpcpb.LineTaskConsultRequest{
		AppId:              command.AppID,
		BuilderId:          int32(command.BuilderID),
		MessageText:        command.MessageText,
		ReferenceTime:      command.ReferenceTime,
		TimeZone:           command.TimeZone,
		SupportedTaskTypes: command.SupportedTaskTypes,
		ClientIp:           command.ClientIP,
	}

	response, err := s.client.LineTaskConsult(ctx, request)
	if err != nil {
		return LineTaskConsultResult{}, infra.NewInternalGRPCError(err)
	}

	result := LineTaskConsultResult{
		TaskType:      response.TaskType,
		Operation:     response.Operation,
		Summary:       response.Summary,
		StartAt:       response.StartAt,
		EndAt:         response.EndAt,
		Location:      response.Location,
		MissingFields: response.MissingFields,
	}

	return result, nil
}
