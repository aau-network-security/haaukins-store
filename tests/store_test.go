package tests

import (
	"context"
	"fmt"
	pb "github.com/aau-network-security/haaukins-store/proto"
	"github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"testing"
	"os"
)

const (
	AUTH_KEY    = "au"
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

func TestStoreConnection(t *testing.T){

	address := os.Getenv("HOST")

	tokenCorret := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		AUTH_KEY: os.Getenv("AUTH_KEY"),
	})

	tokenError := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		AUTH_KEY: "wrong-token",
	})

	tt := []struct{
		name		string
		token 		*jwt.Token
		err			string
	}{
		{name: "Normal Authentication", token: tokenCorret},
		{name: "Unauthorized", token: tokenError, err: "Invalid Authentication Key"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			signin_key := os.Getenv("SIGNIN_KEY")
			cert := os.Getenv("CERT")

			tokenString, err := tc.token.SignedString([]byte(signin_key))
			if err != nil {
				t.Fatalf("Error creating the token")
			}

			authCreds := Creds{Token: tokenString}
			creds, _ := credentials.NewClientTLSFromFile(cert, "")

			dialOpts := []grpc.DialOption{
				grpc.WithTransportCredentials(creds),
				grpc.WithPerRPCCredentials(authCreds),
			}
			conn, err := grpc.Dial(address, dialOpts...)
			if err != nil {
				t.Fatalf("Connection error: %v", err)
			}
			defer conn.Close()

			c := pb.NewStoreClient(conn)

			_, err = c.GetEvents(context.Background(), &pb.EmptyRequest{})

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

//todo add more tests cases and make the CI
//func TestStoreConnectionWithoutToken(t *testing.T){
//
//	conn, err := grpc.Dial(address, grpc.WithInsecure())
//	if err != nil {
//		t.Fatalf("Connection error: %v", err)
//	}
//	defer conn.Close()
//
//	c := pb.NewStoreClient(conn)
//
//	_, err = c.GetEvents(context.Background(), &pb.EmptyRequest{})
//
//	expectedError := "No Authentication Key provided"
//
//	if err != nil {
//		st, ok := status.FromError(err)
//		if ok {
//			err = fmt.Errorf(st.Message())
//		}
//		if err.Error() != expectedError {
//			t.Fatalf("unexpected error (expected: %s) received: %s", expectedError, err.Error())
//		}
//		return
//	}
//	t.Fatalf("expected error, but received none")
//}