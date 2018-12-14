package auth

import (
	"github.com/pkg/errors"

	"github.com/perenecabuto/CatchCatch/server/websocket"
)

// StringIDCookieName is the StringAuthenticator cookie with the user id value
const StringIDCookieName = "X-CatchCatch-ID"

// StringAuthenticator extract the connection id from StringIDCookieName value
type StringAuthenticator struct{}

var _ websocket.Authenticator = (*StringAuthenticator)(nil)

// NewString creates a new StringAuthenticator
func NewString() StringAuthenticator {
	return StringAuthenticator{}
}

// GetConnectionID get connection id from StringIDCookieName cookie value
func (a StringAuthenticator) GetConnectionID(c websocket.WSConnection) (string, error) {
	for _, cookie := range c.Cookies() {
		if cookie.Name == StringIDCookieName {
			return cookie.Value, nil
		}
	}
	return "", errors.Cause(ErrIDNotFound)
}
