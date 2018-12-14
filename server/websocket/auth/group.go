package auth

import (
	"github.com/pkg/errors"

	"github.com/perenecabuto/CatchCatch/server/websocket"
)

// GroupAuthenticator group many authenticators and return the first id found
type GroupAuthenticator struct {
	group []websocket.Authenticator
}

var _ websocket.Authenticator = (*GroupAuthenticator)(nil)

// NewGroup creates a new GroupAuthenticator
func NewGroup(auth ...websocket.Authenticator) GroupAuthenticator {
	return GroupAuthenticator{group: auth}
}

// GetConnectionID get the first id from the auth group
// it returns the first error received
// when id is not found try the next authenticator
// if none if found return ErrIDNotFound
func (a GroupAuthenticator) GetConnectionID(c websocket.WSConnection) (string, error) {
	for _, auth := range a.group {
		id, err := auth.GetConnectionID(c)
		if err == ErrIDNotFound {
			continue
		}
		if err != nil {
			return "", errors.WithStack(err)
		}
		if id != "" {
			return id, nil
		}
	}
	return "", errors.WithStack(ErrIDNotFound)
}
