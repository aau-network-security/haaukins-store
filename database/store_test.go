package database

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	pb "github.com/aau-network-security/haaukins-store/proto"
	"github.com/golang-jwt/jwt/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
)

const (
	AUTH_KEY       = "au"
	AUTH_KEY_VALUE = "authkey"
	SIGNIN_VALUE   = "signkey"
	HOST           = "localhost:50051"
)

var (
	testCertPath    = os.Getenv("CERT")
	testCertKeyPath = os.Getenv("CERT_KEY")
	testCAPath      = os.Getenv("CA")
)

type Creds struct {
	Token    string
	Insecure bool
}

func (c Creds) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		"token": string(c.Token),
	}, nil
}

func (c Creds) RequireTransportSecurity() bool {
	return !c.Insecure
}

func TestStoreConnection(t *testing.T) {

	tokenCorret := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		AUTH_KEY: AUTH_KEY_VALUE,
	})

	tokenError := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		AUTH_KEY: "wrong-token",
	})

	tt := []struct {
		name  string
		token *jwt.Token
		err   string
	}{
		{name: "Test Normal Authentication", token: tokenCorret},
		{name: "Test Unauthorized", token: tokenError, err: "Invalid Authentication Key"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			tokenString, err := tc.token.SignedString([]byte(SIGNIN_VALUE))
			if err != nil {
				t.Fatalf("Error creating the token")
			}

			authCreds := Creds{Token: tokenString}

			// Load the client certificates from disk
			certificate, err := tls.LoadX509KeyPair(testCertPath, testCertKeyPath)
			if err != nil {
				t.Fatalf("could not load client key pair: %s", err)
			}

			// Create a certificate pool from the certificate authority
			certPool := x509.NewCertPool()
			ca, err := ioutil.ReadFile(testCAPath)
			if err != nil {
				t.Fatalf("could not read ca certificate: %s", err)
			}

			// Append the certificates from the CA
			if ok := certPool.AppendCertsFromPEM(ca); !ok {
				t.Fatalf("failed to append ca certs")
			}

			creds := credentials.NewTLS(&tls.Config{
				ServerName:   HOST,
				Certificates: []tls.Certificate{certificate},
				RootCAs:      certPool,
			})

			dialOpts := []grpc.DialOption{
				grpc.WithTransportCredentials(creds),
				grpc.WithPerRPCCredentials(authCreds),
			}
			// Create a connection with the TLS credentials

			conn, err := grpc.Dial(HOST, dialOpts...)
			if err != nil {
				t.Fatalf("Connection error: %v", err)
			}
			defer conn.Close()

			c := pb.NewStoreClient(conn)

			_, err = c.GetEvents(context.Background(), &pb.GetEventRequest{})

			if err != nil {
				st, ok := status.FromError(err)
				if ok {
					err = fmt.Errorf(st.Message())
				}

				if tc.err != "" {
					if tc.err != err.Error() {
						t.Fatalf("unexpected error (expected: %s) received: %s", tc.err, err.Error())
					}
					return
				}
				t.Fatalf("expected no error, but received: %s", err)
			}

			if tc.err != "" {
				t.Fatalf("expected error, but received none")
			}
		})
	}
}

func createTestClientConn() (*grpc.ClientConn, error) {

	tokenCorret := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		AUTH_KEY: AUTH_KEY_VALUE,
	})

	tokenString, err := tokenCorret.SignedString([]byte(SIGNIN_VALUE))
	if err != nil {
		return nil, err
	}

	authCreds := Creds{Token: tokenString}

	// Load the client certificates from disk
	certificate, err := tls.LoadX509KeyPair(testCertPath, testCertKeyPath)
	if err != nil {
		return nil, err
	}

	// Create a certificate pool from the certificate authority
	certPool := x509.NewCertPool()
	ca, err := ioutil.ReadFile(testCAPath)
	if err != nil {
		return nil, err
	}

	// Append the certificates from the CA
	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, err
	}

	creds := credentials.NewTLS(&tls.Config{
		ServerName:   HOST,
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
	})

	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(authCreds),
	}

	// Create a connection with the TLS credentials
	conn, err := grpc.Dial(HOST, dialOpts...)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func TestAddEvent(t *testing.T) {
	dbConn, err := createDBConnection()
	if err != nil {
		t.Fatalf("error on database connection create %v", err)
	}
	if err := cleanRecords(dbConn); err != nil {
		t.Fatalf("error on cleaning records %v", err)
	}
	conn, err := createTestClientConn()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := pb.NewStoreClient(conn)

	req := pb.AddEventRequest{
		Name:               "Test",
		Tag:                "test",
		Frontends:          "kali",
		Exercises:          "ftp,xss",
		Available:          1,
		Capacity:           2,
		StartTime:          "2020-05-20 14:35:01",
		Status:             1,
		ExpectedFinishTime: "2020-05-21 14:35:01",
		FinishedAt:         "0001-01-01 00:00:00", // it means that event is not finished yet
		OnlyVPN:            0,
		SecretKey:          "",
		DisabledExercises:  "",
	}

	resp, err := c.AddEvent(context.Background(), &req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.ErrorMessage != "" {
		t.Fatal(errors.New(resp.ErrorMessage))
	}
	events, err := c.GetEvents(context.Background(), &pb.GetEventRequest{Status: 1})
	if err != nil {
		t.Fatal(err)
	}

	if len(events.Events) != 1 {
		t.Fatal("Error getting the stored events")
	}
}

func TestAddTeam(t *testing.T) {

	conn, err := createTestClientConn()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := pb.NewStoreClient(conn)

	_, err = c.AddTeam(context.Background(), &pb.AddTeamRequest{
		Id:       "team1",
		EventTag: "test",
		Email:    "team1@test.dk",
		Name:     "Team Test 1",
		Password: "password",
	})
	if err != nil {
		t.Logf("Error happened in AddTeam to event %v\n", err)
		t.Fatal()
	}

	teams, err := c.GetEventTeams(context.Background(), &pb.GetEventTeamsRequest{
		EventTag: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(teams.Teams) != 1 {
		t.Fatal("Error getting the stored teams")
	}
}

func TestTeamSolveChallenge(t *testing.T) {
	conn, err := createTestClientConn()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := pb.NewStoreClient(conn)

	_, err = c.UpdateTeamSolvedChallenge(context.Background(), &pb.UpdateTeamSolvedChallengeRequest{
		TeamId:      "team1",
		Tag:         "ftp",
		CompletedAt: "2020-05-21 12:35:01",
	})
	if err != nil {
		t.Fatalf("Error updating the solved challenges: %s", err.Error())
	}
}

func TestTeamUpdateLastAccess(t *testing.T) {

	conn, err := createTestClientConn()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := pb.NewStoreClient(conn)

	_, err = c.UpdateTeamLastAccess(context.Background(), &pb.UpdateTeamLastAccessRequest{
		TeamId:   "team1",
		AccessAt: "2020-05-21 12:35:01",
	})
	if err != nil {
		t.Fatalf("Error updating team last access: %s", err.Error())
	}
}

func TestCloseEvent(t *testing.T) {

	conn, err := createTestClientConn()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := pb.NewStoreClient(conn)
	newTag := fmt.Sprintf("%s-%s", "test", strconv.Itoa(int(time.Now().Unix())))
	_, err = c.UpdateCloseEvent(context.Background(), &pb.UpdateEventRequest{
		OldTag:     "test",
		NewTag:     newTag,
		FinishedAt: "2020-05-21 14:35:00",
	})
	if err != nil {
		t.Fatalf("Error closing event: %s", err.Error())
	}
}

func TestMultipleEventWithSameTag(t *testing.T) {

	t.Log("Testing Multiple Events with same Tags")
	conn, err := createTestClientConn()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	c := pb.NewStoreClient(conn)

	req := pb.AddEventRequest{
		Name:               "Test2",
		Tag:                "test",
		Frontends:          "kali",
		Exercises:          "ftp,xss,wc,jwt",
		Available:          1,
		Status:             1,
		Capacity:           2,
		StartTime:          "2020-06-20 14:35:01",
		ExpectedFinishTime: "2020-06-21 14:35:01",
		OnlyVPN:            0,
		SecretKey:          "",
		DisabledExercises:  "",
	}

	_, err = c.AddEvent(context.Background(), &req)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.AddTeam(context.Background(), &pb.AddTeamRequest{
		Id:       "team1",
		EventTag: "test",
		Email:    "team1@test.dk",
		Name:     "Team Test 1",
		Password: "password",
	})
	if err != nil {
		t.Fatal()
	}

	teams, err := c.GetEventTeams(context.Background(), &pb.GetEventTeamsRequest{
		EventTag: "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(teams.Teams) != 1 {
		t.Fatal("Error getting the stored teams in Testing Multiple Events with same Tags")
	}
}
