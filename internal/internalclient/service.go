package internalclient

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/idtoken"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	grpcoauth "google.golang.org/grpc/credentials/oauth"

	"linebot-backend/internal/infra"
	grpcpb "linebot-backend/internal/internalclient/pb"
)

// Service wraps Internal AI Copilot gRPC client.
type Service struct {
	conn   *grpc.ClientConn
	client grpcpb.IntegrationServiceClient
}

// NewService creates a new Internal AI Copilot gRPC client.
// When insecureConn is true, plain TCP is used (local dev).
// When false, TLS + OIDC token are used (Cloud Run service-to-service).
func NewService(grpcAddr string, insecureConn bool) (*Service, error) {
	var opts []grpc.DialOption
	if insecureConn {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
		// Cloud Run service-to-service: attach OIDC token so the target service
		// can verify the caller's identity via Cloud Run IAM (roles/run.invoker).
		host := strings.SplitN(grpcAddr, ":", 2)[0]
		audience := "https://" + host
		ts, err := idtoken.NewTokenSource(context.Background(), audience)
		if err != nil {
			return nil, fmt.Errorf("idtoken.NewTokenSource: %w", err)
		}
		opts = append(opts, grpc.WithPerRPCCredentials(grpcoauth.TokenSource{TokenSource: ts}))
	}

	conn, err := grpc.NewClient(grpcAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("grpc.NewClient: %w", err)
	}

	return &Service{
		conn:   conn,
		client: grpcpb.NewIntegrationServiceClient(conn),
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
