package auth

import (
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/perenecabuto/CatchCatch/server/websocket"
	"github.com/pkg/errors"
)

// JWTCookieName is the JWTAuthenticator cookie name
// its value must be a valid jwt and send jti claims as string
const JWTCookieName = "X-CatchCatch-Auth"

// JWTAuthenticationErrors
var (
	ErrJWTExpired           = errors.New("jwt expired")
	ErrJWTInvalid           = errors.New("jwt is invalid")
	ErrJWTUnsupportedClaims = errors.New("jwt claims format invalid")
	ErrJWTClaimsJTIInvalid  = errors.New("jwt jti (id) must be a string")
)

// JWTAuthenticator get the jwt auth from cookies and get the connection id
type JWTAuthenticator struct {
	key interface{}
}

var _ websocket.Authenticator = (*JWTAuthenticator)(nil)

// NewJWT creates a new JWTAuthenticator with a verify key
func NewJWT(key interface{}) JWTAuthenticator {
	return JWTAuthenticator{key: key}
}

// GetConnectionID scan cookies to get the cookie with the JWT
// it validates the JWT with the key and extract the id from the jti value on claims
func (a JWTAuthenticator) GetConnectionID(c websocket.WSConnection) (string, error) {
	var signed string
	for _, cookie := range c.Cookies() {
		if cookie.Name == JWTCookieName {
			signed = cookie.Value
			break
		}
	}
	if signed == "" {
		return "", errors.Cause(ErrIDNotFound)
	}
	token, err := jwt.Parse(signed, func(token *jwt.Token) (interface{}, error) {
		return a.key, nil
	})
	if err != nil {
		return "", errors.Cause(err)
	}
	if !token.Valid {
		return "", errors.Cause(ErrJWTInvalid)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.Cause(ErrJWTUnsupportedClaims)
	}
	id, ok := claims["sub"].(string)
	if !ok {
		id, ok = claims["jti"].(string)
	}
	if !ok {
		return "", errors.Cause(ErrJWTClaimsJTIInvalid)
	}
	return id, nil
}
