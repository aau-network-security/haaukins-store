package util

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/aau-network-security/haaukins-store/database"
	"github.com/aau-network-security/haaukins-store/model"
	pb "github.com/aau-network-security/haaukins-store/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v2"
)

type server struct {
	store database.Store
	auth  Authenticator
	tls   bool
	pb.UnimplementedStoreServer
}

type certificate struct {
	cPath    string
	cKeyPath string
	caPath   string
}

var (
	Running   = State(0)
	Booked    = State(1) // todo: will be added
	Suspended = State(2)
	Closed    = State(3)
	Error     = State(4)
)

type State int32

func (s server) AddProfile(ctx context.Context, in *pb.AddProfileRequest) (*pb.InsertResponse, error) {
	result, err := s.store.AddProfile(in)
	if err != nil {
		log.Printf("ERR: Error Add Profile %s", err.Error())
		return &pb.InsertResponse{ErrorMessage: err.Error()}, nil
	}
	log.Printf("Profile %s Saved", in.Name)
	return &pb.InsertResponse{Message: result}, nil
}

func (s server) GetProfiles(ctx context.Context, in *pb.EmptyRequest) (*pb.GetProfilesResp, error){
	result, err := s.store.GetProfiles()
	if err != nil {
		log.Printf("ERR: Error Getting Profiles %s", err.Error())
		return &pb.GetProfilesResp{ErrorMessage: err.Error()}, nil
	}
	type Challenge struct {
		Tag string `json:"tag"`
		Name string `json:"name"`
	}
	var profiles []*pb.GetProfilesResp_Profile
	var challenges []Challenge
	for _, p := range result {
		var chals []*pb.GetProfilesResp_Profile_Challenge
		err := json.Unmarshal([]byte(p.Challenges), &challenges)
		if err != nil {
			//log.Printf("Error while marshalling: %s", err)
			return &pb.GetProfilesResp{ErrorMessage: err.Error()}, nil
		}
		for _, c := range challenges {
			//log.Printf("Looping through challenges... Found: %s in profile: %s", c.Name, p.Name)
			chals = append(chals, &pb.GetProfilesResp_Profile_Challenge{
				Tag: c.Tag,
				Name: c.Name,
			})
		}
		profiles = append(profiles, &pb.GetProfilesResp_Profile{
			Name: p.Name,
			Challenges: chals,
		})
	}
	return &pb.GetProfilesResp{Profiles: profiles}, nil
}

func (s server) UpdateProfile(ctx context.Context, in *pb.UpdateProfileRequest) (*pb.UpdateResponse, error) {
	result, err := s.store.UpdateProfile(in)
	if err != nil {
		log.Printf("ERR: Error Update Profile %s", err.Error())
		return &pb.UpdateResponse{ErrorMessage: err.Error()}, nil
	}
	log.Printf("Profile %s Updated", in.Name)
	return &pb.UpdateResponse{Message: result}, nil
}

func (s server) DeleteProfile(ctx context.Context, in *pb.DelProfileRequest) (*pb.DelProfileResp, error) {
	result, err := s.store.DeleteProfile(in)
	if err != nil {
		log.Printf("ERR: Error Delete Profile %s", err.Error())
		return &pb.DelProfileResp{ErrorMessage: err.Error()}, nil
	}
	log.Printf("Profile %s Deleted", in.Name)
	return &pb.DelProfileResp{Message: result}, nil
}

func (s server) AddEvent(ctx context.Context, in *pb.AddEventRequest) (*pb.InsertResponse, error) {
	result, err := s.store.AddEvent(in)
	if err != nil {
		log.Printf("ERR: Error Add Event %s", err.Error())
		return &pb.InsertResponse{ErrorMessage: err.Error()}, nil
	}
	log.Printf("Event %s Saved", in.Tag)
	return &pb.InsertResponse{Message: result}, nil

}

func (s server) AddTeam(ctx context.Context, in *pb.AddTeamRequest) (*pb.InsertResponse, error) {
	// check event
	// status : closed means invert, will check, running, suspended and booked ones
	ok, err := s.store.IsEventExists(&pb.GetEventByTagReq{EventTag: in.EventTag, Status: int32(Closed)})
	if !ok {
		return &pb.InsertResponse{ErrorMessage: "No event found by given tag"}, fmt.Errorf("Event %v does not exists", in.EventTag)
	}

	result, err := s.store.AddTeam(in)
	if err != nil {
		log.Printf("ERR: Error Add Team %s", err.Error())
		return &pb.InsertResponse{ErrorMessage: err.Error()}, nil
	}
	log.Printf("Team %s Saved for the Event %s", in.Id, in.EventTag)
	return &pb.InsertResponse{Message: result}, nil
}

func (s server) GetEvents(ctx context.Context, in *pb.GetEventRequest) (*pb.GetEventResponse, error) {
	result, err := s.store.GetEvents(in)
	if err != nil {
		log.Printf("ERR: Error Get Events %s", err.Error())
		return &pb.GetEventResponse{ErrorMessage: err.Error()}, nil
	}
	events := getEventsResponse(result)
	return &pb.GetEventResponse{Events: events}, nil
}

func (s server) IsEventExists(ctx context.Context, in *pb.GetEventByTagReq) (*pb.GetEventByTagResp, error) {
	isExist, err := s.store.IsEventExists(in)
	if err != nil {
		return &pb.GetEventByTagResp{}, err
	}
	return &pb.GetEventByTagResp{IsExist: isExist}, nil
}

// this might be somehow handled by GetEvents too
// however it is much easy to create new function
func (s server) GetEventByUser(ctx context.Context, in *pb.GetEventByUserReq) (*pb.GetEventResponse, error) {
	result, err := s.store.GetEventByUser(in)
	if err != nil {
		log.Printf("ERR: get events by user %s", err.Error())
		return &pb.GetEventResponse{ErrorMessage: err.Error()}, nil
	}
	events := getEventsResponse(result)
	return &pb.GetEventResponse{Events: events}, nil
}

func (s server) DropEvent(ctx context.Context, in *pb.DropEventReq) (*pb.DropEventResp, error) {
	isDropped, err := s.store.DropEvent(in)
	if err != nil {
		return &pb.DropEventResp{}, err
	}
	return &pb.DropEventResp{IsDropped: isDropped}, nil
}

func (s server) GetEventStatus(ctx context.Context, in *pb.GetEventStatusRequest) (*pb.EventStatusStore, error) {
	result, err := s.store.GetEventStatus(in)
	if err != nil {
		return &pb.EventStatusStore{Status: int32(Error)}, err
	}
	log.Printf("Event status returned ! [Status: %d , Event: %s] ", result, in.EventTag)
	return &pb.EventStatusStore{Status: result}, nil

}

func (s server) SetEventStatus(ctx context.Context, in *pb.SetEventStatusRequest) (*pb.EventStatusStore, error) {
	log.Printf("Set event status for event %s to %d", in.EventTag, in.Status)
	result, err := s.store.SetEventStatus(in)
	if err != nil {
		return &pb.EventStatusStore{Status: int32(Error)}, err
	}

	log.Printf("Event status updated ! [Status: %d , Event: %s] ", result, in.EventTag)

	return &pb.EventStatusStore{Status: result}, nil
}

func (s server) GetTimeSeries(ctx context.Context, r *pb.EmptyRequest) (*pb.GetTimeSeriesResponse, error) {
	log.Printf("Calculating costs in timeline")
	m, err := s.store.GetCostsInTime()
	if err != nil {
		return nil, fmt.Errorf("error on calculating costs %v", err)
	}
	return &pb.GetTimeSeriesResponse{Timeseries: m}, nil
}

func (s server) GetEventTeams(ctx context.Context, in *pb.GetEventTeamsRequest) (*pb.GetEventTeamsResponse, error) {
	result, err := s.store.GetTeams(in.EventTag)
	if err != nil {
		log.Printf("ERR: Error Get teams for Event %s : %s", in.EventTag, err.Error())
		return &pb.GetEventTeamsResponse{ErrorMessage: err.Error()}, nil
	}

	var teams []*pb.GetEventTeamsResponse_Teams
	for _, t := range result {
		teams = append(teams, &pb.GetEventTeamsResponse_Teams{
			Id:               t.Tag,
			Email:            t.Email,
			Name:             t.Name,
			HashPassword:     t.Password,
			CreatedAt:        t.CreatedAt,
			LastAccess:       t.LastAccess,
			SolvedChallenges: t.SolvedChallenges,
		})
	}
	log.Printf("Get Teams for the Event %s", in.EventTag)
	return &pb.GetEventTeamsResponse{Teams: teams}, nil
}

func (s server) UpdateCloseEvent(ctx context.Context, in *pb.UpdateEventRequest) (*pb.UpdateResponse, error) {
	result, err := s.store.UpdateCloseEvent(in)
	if err != nil {
		log.Printf("ERR: Error Update Close Event %s finish time: %s", in.OldTag, err.Error())
		return &pb.UpdateResponse{ErrorMessage: err.Error()}, nil
	}
	log.Printf("Event %s Stopped", in.OldTag)
	return &pb.UpdateResponse{Message: result}, nil
}

func (s server) UpdateTeamSolvedChallenge(ctx context.Context, in *pb.UpdateTeamSolvedChallengeRequest) (*pb.UpdateResponse, error) {
	result, err := s.store.UpdateTeamSolvedChallenge(in)
	if err != nil {
		log.Printf("ERR: Error Update team %s solve challenge: %s", in.TeamId, err.Error())
		return &pb.UpdateResponse{ErrorMessage: err.Error()}, nil
	}
	log.Printf("Team %s solved %s challenge", in.TeamId, in.Tag)
	return &pb.UpdateResponse{Message: result}, nil
}

func (s server) UpdateTeamLastAccess(ctx context.Context, in *pb.UpdateTeamLastAccessRequest) (*pb.UpdateResponse, error) {
	result, err := s.store.UpdateTeamLastAccess(in)
	if err != nil {
		log.Printf("ERR: Error Update team %s last access: %s", in.TeamId, err.Error())
		return &pb.UpdateResponse{ErrorMessage: err.Error()}, nil
	}
	return &pb.UpdateResponse{Message: result}, nil
}

func (s server) DeleteTeam(ctx context.Context, req *pb.DelTeamRequest) (*pb.DelTeamResp, error) {
	result, err := s.store.DelTeam(req)
	if err != nil {
		log.Printf("ERR: Error delete team %s from event %s, err: %s", req.TeamId, req.EvTag, err.Error())
		return &pb.DelTeamResp{Message: err.Error()}, nil
	}
	return &pb.DelTeamResp{Message: result}, nil
}

func (s server) UpdateExercises(ctx context.Context, req *pb.UpdateExerciseRequest) (*pb.UpdateExerciseResponse, error) {
	resp, err := s.store.UpdateExercises(req)
	if err != nil {
		return &pb.UpdateExerciseResponse{Message: err.Error()}, err
	}

	return &pb.UpdateExerciseResponse{Message: resp}, nil
}

func (s server) UpdateTeamPassword(ctx context.Context, req *pb.UpdateTeamPassRequest) (*pb.UpdateResponse, error) {
	if err := s.store.UpdateTeamPassword(req); err != nil {
		log.Printf("ERR: Error Update team %s password: %s", req.TeamID, err.Error())
		return &pb.UpdateResponse{ErrorMessage: err.Error()}, err
	}

	return &pb.UpdateResponse{}, nil
}

func (s server) GetEventID(ctx context.Context, req *pb.GetEventIDReq) (*pb.GetEventIDResp, error) {
	id, err := s.store.GetEventID(req)
	if err != nil {
		return &pb.GetEventIDResp{}, err
	}

	return &pb.GetEventIDResp{EventID: id}, nil
}

func GetCreds(conf *model.Config) (credentials.TransportCredentials, error) {
	log.Printf("Preparing credentials for RPC")

	certificateProps := certificate{
		cPath:    conf.TLS.CertFile,
		cKeyPath: conf.TLS.CertKey,
		caPath:   conf.TLS.CAFile,
	}

	certificate, err := tls.LoadX509KeyPair(certificateProps.cPath, certificateProps.cKeyPath)
	if err != nil {
		return nil, fmt.Errorf("could not load server key pair: %s", err)
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(certificateProps.caPath)
	if err != nil {
		return nil, fmt.Errorf("could not read ca certificate: %s", err)
	}
	// CA file for let's encrypt is located under domain conf as `chain.pem`
	// pass chain.pem location
	// Append the client certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, errors.New("failed to append client certs")
	}

	// Create the TLS credentials
	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:   tls.RequireAndVerifyClientCert,
		Certificates: []tls.Certificate{certificate},
		ClientCAs:    certPool,
	})
	return creds, nil
}

func (s server) GrpcOpts(conf *model.Config) ([]grpc.ServerOption, error) {

	if conf.TLS.Enabled {
		creds, err := GetCreds(conf)

		if err != nil {
			return []grpc.ServerOption{}, errors.New("Error on retrieving certificates: " + err.Error())
		}
		log.Printf("Server is running in secure mode !")
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

func InitilizegRPCServer(conf *model.Config) (*server, error) {

	store, err := database.NewStore(conf)

	if err != nil {
		return nil, err
	}

	s := &server{
		store: store,
		auth:  NewAuthenticator(conf.SigninKey, conf.AuthKey),
		tls:   conf.TLS.Enabled,
	}
	return s, nil
}

func NewConfigFromFile(path string) (*model.Config, error) {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c model.Config
	err = yaml.Unmarshal(f, &c)
	if err != nil {
		return nil, err
	}

	if c.Host == "" {
		log.Println("Host not provided in the configuration file")
		c.Host = "localhost:50051"
	}

	if c.SigninKey == "" {
		log.Println("SigninKey not provided in the configuration file")
		c.Host = "dev-env"
	}

	if c.AuthKey == "" {
		log.Println("AuthKey not provided in the configuration file")
		c.Host = "development-environment"
	}

	if c.DB.Host == "" || c.DB.User == "" || c.DB.Pass == "" || c.DB.Name == "" {
		return nil, errors.New("DB paramenters missing in the configuration file")
	}

	if c.DB.Port == 0 {
		c.DB.Port = 5432
	}

	if c.TLS.Enabled {
		if c.TLS.CAFile == "" || c.TLS.CertKey == "" || c.TLS.CertFile == "" {
			return nil, errors.New("Provide Certificates in the config file")
		}
	}

	return &c, nil
}

func getEventsResponse(result []model.Event) []*pb.GetEventResponse_Events {
	var events []*pb.GetEventResponse_Events
	for _, e := range result {
		log.Printf("Event %s , Secret Key %s ", e.Name, e.SecretKey)
		events = append(events, &pb.GetEventResponse_Events{
			Name:               e.Name,
			Tag:                e.Tag,
			Frontends:          e.Frontends,
			Exercises:          e.Exercises,
			Available:          int32(e.Available),
			Capacity:           int32(e.Capacity),
			StartedAt:          e.StartedAt,
			ExpectedFinishTime: e.ExpectedFinishTime,
			FinishedAt:         e.FinishedAt,
			Status:             e.Status,
			CreatedBy:          e.CreatedBy,
			OnlyVPN:            e.OnlyVPN,
			SecretKey:          e.SecretKey,
			DisabledExercises:  e.DisabledExercises,
		})
	}
	log.Printf("Get Events")
	return events
}
