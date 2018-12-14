package websocket

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
	"golang.org/x/net/websocket"
)

/*
GobwasWSDriver is a WSDriver implementation based on gobwas/ws
*/
type GobwasWSDriver struct{}

// NewGobwasWSDriver creates a gobwas/ws WSDriver
func NewGobwasWSDriver() WSDriver {
	return &GobwasWSDriver{}
}

// HTTPHandler implements WSDriver.Handler
func (d *GobwasWSDriver) HTTPHandler(ctx context.Context, onConnect func(context.Context, WSConnection)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, _, _, err := ws.UpgradeHTTP(r, w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := r.WithContext(ctx).Context()
		conn := &GobwasWSConn{Conn: c, cookies: r.Cookies()}
		onConnect(ctx, conn)
	})
}

// GobwasWSConn wraps gobwas/ws connections
type GobwasWSConn struct {
	net.Conn
	cookies []*http.Cookie
}

// Send implements WSConnection.Send
func (c *GobwasWSConn) Send(payload []byte) error {
	return wsutil.WriteServerBinary(c, payload)
}

// Read implements WSConnection.Read
func (c *GobwasWSConn) Read(buff *[]byte) (int, error) {
	header, err := ws.ReadHeader(c.Conn)
	if err != nil {
		return 0, fmt.Errorf("readMessage(header): %s", err.Error())
	}
	if header.OpCode == ws.OpClose {
		return 0, fmt.Errorf("readMessage(closed)")
	}
	length := int(header.Length)
	if _, err := io.ReadAtLeast(c.Conn, *buff, length); err != nil {
		return 0, fmt.Errorf("readMessage(body): %s", err.Error())
	}
	if header.Masked {
		ws.Cipher(*buff, header.Mask, 0)
	}
	return length, nil
}

// Cookies returns ws connection request cookies
func (c *GobwasWSConn) Cookies() []*http.Cookie {
	return c.cookies
}

/*
XNetWSDriver is a WSDriver implementation based on x/net/websocket
*/
type XNetWSDriver struct{}

// NewXNetWSDriver creates a x/net/websocket WSDriver
func NewXNetWSDriver() WSDriver {
	return &XNetWSDriver{}
}

// HTTPHandler implements WSDriver.Handler
func (d *XNetWSDriver) HTTPHandler(ctx context.Context, onConnect func(context.Context, WSConnection)) http.Handler {
	return websocket.Server{
		Handler: func(c *websocket.Conn) {
			conn := &XNetWSConn{Conn: c}
			ctx := c.Request().WithContext(ctx).Context()
			onConnect(ctx, conn)
		},
	}
}

// XNetWSConn wraps x/net/websocket connections
type XNetWSConn struct {
	*websocket.Conn
}

// Read implements WSConnection.Read
func (c *XNetWSConn) Read(buff *[]byte) (int, error) {
	err := websocket.Message.Receive(c.Conn, buff)
	return len(*buff), err
}

// Send implements WSConnection.Send
func (c *XNetWSConn) Send(payload []byte) error {
	return websocket.Message.Send(c.Conn, payload)
}

// Cookies returns ws connection request cookies
func (c *XNetWSConn) Cookies() []*http.Cookie {
	return c.Conn.Request().Cookies()
}
