package main

import (
	"context"
	"fmt"
	"github.com/aau-network-security/haaukins-store/database"
	pb "github.com/aau-network-security/haaukins-store/proto"
	_ "github.com/lib/pq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

const (
	port = ":50051"
)



type server struct {
	store 	database.Store
	auth 	Authenticator
	tls		bool
	cert	string
	certKey string
}

func (s server) AddEvent(ctx context.Context, in *pb.AddEventRequest) (*pb.InsertResponse, error) {
	result, err := s.store.AddEvent(in)
	if err != nil {
		return &pb.InsertResponse{ErrorMessage: err.Error()}, nil
	}
	return &pb.InsertResponse{ Message: result }, nil

}

func (s server) AddTeam(ctx context.Context, in *pb.AddTeamRequest) (*pb.InsertResponse, error) {
	result, err := s.store.AddTeam(in)
	if err != nil {
		return &pb.InsertResponse{ErrorMessage: err.Error()}, nil
	}
	return &pb.InsertResponse{Message: result}, nil
}

func (s server) GetEvents(context.Context, *pb.EmptyRequest) (*pb.GetEventResponse, error) {
	result, err := s.store.GetEvents()
	if err != nil {
		return &pb.GetEventResponse{ErrorMessage: err.Error()}, nil
	}

	var events []*pb.GetEventResponse_Events
	for _, e := range result{
		events = append(events, &pb.GetEventResponse_Events{
			Name:                 e.Name,
			Tag:                  e.Tag,
			Frontends:            e.Frontends,
			Exercises:            e.Exercises,
			Available:            int32(e.Available),
			Capacity:             int32(e.Capacity),
			StartedAt:            e.StartedAt,
			ExpectedFinishTime:   e.ExpectedFinishTime,
			FinishedAt:           e.FinishedAt,
		})
	}

	return &pb.GetEventResponse{Events:events}, nil

}

func (s server) GetEventTeams(ctx context.Context,in *pb.GetEventTeamsRequest) (*pb.GetEventTeamsResponse, error) {
	result, err := s.store.GetTeams(in.EventTag)
	if err != nil {
		return &pb.GetEventTeamsResponse{ErrorMessage: err.Error()}, nil
	}

	var teams []*pb.GetEventTeamsResponse_Teams
	for _, t := range result{
		teams = append(teams, &pb.GetEventTeamsResponse_Teams{
			Id:                   t.Id,
			Email:                t.Email,
			Name:                 t.Name,
			HashPassword:         t.Password,
			CreatedAt:            t.CreatedAt,
			LastAccess:           t.LastAccess,
			SolvedChallenges:     t.SolvedChallenges,
		})
	}

	return &pb.GetEventTeamsResponse{Teams:teams}, nil
}

func (s server) UpdateEventFinishDate(ctx context.Context, in *pb.UpdateEventRequest) (*pb.UpdateResponse, error) {
	result, err := s.store.UpdateEventFinishDate(in)
	if err != nil {
		return &pb.UpdateResponse{ErrorMessage: err.Error()}, nil
	}
	return &pb.UpdateResponse{Message: result}, nil
}

func (s server) UpdateTeamSolvedChallenge(ctx context.Context, in *pb.UpdateTeamSolvedChallengeRequest) (*pb.UpdateResponse, error) {
 	result, err := s.store.UpdateTeamSolvedChallenge(in)
 	if err != nil {
 		return &pb.UpdateResponse{ErrorMessage: err.Error()}, nil
	}
	return &pb.UpdateResponse{Message: result}, nil
}

func (s server) UpdateTeamLastAccess(ctx context.Context, in *pb.UpdateTeamLastAccessRequest) (*pb.UpdateResponse, error) {
	result, err := s.store.UpdateTeamLastAccess(in)
	if err != nil {
		return &pb.UpdateResponse{ErrorMessage: err.Error()}, nil
	}
	return &pb.UpdateResponse{Message: result}, nil
}

func (s server) grpcOpts() ([]grpc.ServerOption, error) {
	if s.tls {
		creds, err := credentials.NewServerTLSFromFile(s.cert, s.certKey)
		if err != nil {
			return []grpc.ServerOption{}, err
		}
		return []grpc.ServerOption{grpc.Creds(creds)}, nil
	}
	return []grpc.ServerOption{}, nil
}

func (s server) GetGRPCServer(opts ...grpc.ServerOption) *grpc.Server {

	streamInterceptor := func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if err := s.auth.AuthenticateContext(stream.Context()); err != nil {
			return err
		}
		return handler(srv, stream)
	}

	unaryInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := s.auth.AuthenticateContext(ctx); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	}

	opts = append([]grpc.ServerOption{
		grpc.StreamInterceptor(streamInterceptor),
		grpc.UnaryInterceptor(unaryInterceptor),
	}, opts...)
	return grpc.NewServer(opts...)
}

func readContent(path string) error {
	cont, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
		return err
	}
	_, err = io.Copy(os.Stdout, cont)
	if err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func InitilizegRPCServer()*server {

	store, err := database.NewStore()
	certificate := os.Getenv("CERT")
	certificateKey:=  os.Getenv("CERT_KEY")

	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}

	if err := readContent(certificate); err !=nil {
		log.Fatalf(err.Error())
	}
	if err:= readContent(certificateKey); err !=nil {
		log.Fatalf(err.Error())
	}

	tls := true
	mode := os.Getenv("SSL_OFF")
	if strings.ToLower(mode) == "true" {
		tls = false
	}

	s := &server{
		store:		store,
		auth: 		NewAuthenticator(os.Getenv("SIGNIN_KEY")),
		tls: 		tls,
		cert:   	certificate,
		certKey:    certificateKey,
	}

	return s
}

func main() {

	s := InitilizegRPCServer()
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	opts, err := s.grpcOpts()
	if err != nil {
		log.Fatal("failed to retrieve server options")
	}

	gRPCServer := s.GetGRPCServer(opts...)
	pb.RegisterStoreServer(gRPCServer, s)
	fmt.Println("waiting client")
	if err := gRPCServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}

}
