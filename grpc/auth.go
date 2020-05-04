package grpc

import (
	"context"
	"errors"
	"fmt"
	jwt "github.com/dgrijalva/jwt-go"
	"google.golang.org/grpc/metadata"
	"os"
)

const (
	AUTH_KEY    = "au"
)

var (
	InvalidAuthKey   		 = errors.New("Invalid Authentication Key")
	InvalidTokenFormatErr    = errors.New("Invalid token format")
	MissingKeyErr      	  	 = errors.New("No Authentication Key provided")
)

type Authenticator interface {
	AuthenticateContext(context.Context) error
}

type auth struct {
	key string
}

func NewAuthenticator(key string) Authenticator{
	return &auth{key:key}
}

func (a *auth) AuthenticateContext(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return MissingKeyErr
	}

	if len(md["token"]) == 0 {
		return MissingKeyErr
	}

	token := md["token"][0]
	if token == "" {
		return MissingKeyErr
	}

	jwtToken, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return ctx, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}

		return []byte(a.key), nil
	})
	if err != nil {
		return err
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok || !jwtToken.Valid {
		return InvalidTokenFormatErr
	}

	authKey, ok := claims[AUTH_KEY].(string)
	if !ok {
		return InvalidTokenFormatErr
	}

	if authKey != os.Getenv("AUTH_KEY") {
		return InvalidAuthKey
	}

	return nil
}
