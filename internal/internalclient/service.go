package internalclient

import (
	"context"
	"fmt"
	"log"

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
	log.Printf("[INFO] internal grpc line-task request: appID=%s builderID=%d messageText=%q referenceTime=%q timeZone=%q supportedTaskTypes=%v clientIP=%q", command.AppID, command.BuilderID, command.MessageText, command.ReferenceTime, command.TimeZone, command.SupportedTaskTypes, command.ClientIP)
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
		log.Printf("[INFO] internal grpc line-task failed: err=%v", err)
		return LineTaskConsultResult{}, infra.NewInternalGRPCError(err)
	}

	result := LineTaskConsultResult{
		TaskType:      response.TaskType,
		Operation:     response.Operation,
		EventID:       response.EventId,
		Summary:       response.Summary,
		StartAt:       response.StartAt,
		EndAt:         response.EndAt,
		QueryStartAt:  response.QueryStartAt,
		QueryEndAt:    response.QueryEndAt,
		Location:      response.Location,
		MissingFields: response.MissingFields,
	}

	log.Printf("[INFO] internal grpc line-task response: taskType=%s operation=%s eventID=%q summary=%q startAt=%q endAt=%q queryStartAt=%q queryEndAt=%q missingFields=%v", result.TaskType, result.Operation, result.EventID, result.Summary, result.StartAt, result.EndAt, result.QueryStartAt, result.QueryEndAt, result.MissingFields)
	return result, nil
}
