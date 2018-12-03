package auth

import (
	"github.com/perenecabuto/CatchCatch/server/websocket"
	uuid "github.com/satori/go.uuid"
)

// RandomAuthenticator get a unique random id
type RandomAuthenticator struct{}

// NewRandom creates a RandomAuthenticator
func NewRandom() RandomAuthenticator {
	return RandomAuthenticator{}
}

var _ websocket.Authenticator = (*RandomAuthenticator)(nil)

// GetConnectionID return an unique random connection id
func (a RandomAuthenticator) GetConnectionID(c websocket.WSConnection) (string, error) {
	id := uuid.NewV4().String()
	return id, nil
}
