package streams

import (
	"github.com/bokysan/socketace/v2/internal/util/buffers"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"io"
	"net"
	"time"
)

// WebsocketConnection implements a ReadWriteCloser over a websocket connection
type WebsocketTunnelConnection struct {
	*websocket.Conn
	closed bool
}

func NewWebsocketTunnelConnection(conn *websocket.Conn) *WebsocketTunnelConnection {
	return &WebsocketTunnelConnection{
		Conn: conn,
	}
}

func (wstc *WebsocketTunnelConnection) Read(p []byte) (int, error) {
	messageType, message, err := wstc.Conn.ReadMessage()
	if messageType == websocket.CloseMessage || messageType == -1 {
		return 0, io.EOF
	} else if messageType != websocket.BinaryMessage {
		return 0, errors.Errorf("Invalid message type: %v", messageType)
	} else if err != nil {
		return 0, errors.WithStack(err)
	}

	msgLen := len(message)
	if len(p) < msgLen {
		return 0, errors.Errorf("Buffer to small: message size is %v, but buffer size is %v", msgLen, len(p))
	}

	copy(p, message)

	return msgLen, nil
}

// Write will take a stream of bytes and send it over a websocket connection.
func (wstc *WebsocketTunnelConnection) Write(p []byte) (int, error) {
	dataLen := len(p)
	for {
		if len(p) > buffers.BufferSize {
			err := wstc.Conn.WriteMessage(websocket.BinaryMessage, p[:buffers.BufferSize])
			if err != nil {
				return 0, errors.WithStack(err)
			}
			p = p[buffers.BufferSize:]
		} else {
			err := wstc.Conn.WriteMessage(websocket.BinaryMessage, p)
			if err != nil {
				return 0, errors.WithStack(err)
			}
			break
		}
	}
	return dataLen, nil
}

func (wstc *WebsocketTunnelConnection) Close() error {
	if wstc.closed {
		return nil
	}
	err := LogClose(wstc.Conn)
	wstc.closed = true

	return err
}

func (wstc *WebsocketTunnelConnection) Closed() bool {
	return wstc.closed
}

func (wstc *WebsocketTunnelConnection) LocalAddr() net.Addr {
	return wstc.Conn.LocalAddr()
}

// RemoteAddr returns the remote network address.
func (wstc *WebsocketTunnelConnection) RemoteAddr() net.Addr {
	return wstc.Conn.RemoteAddr()
}

func (wstc *WebsocketTunnelConnection) SetDeadline(t time.Time) error {
	return wstc.Conn.UnderlyingConn().SetDeadline(t)
}

func (wstc *WebsocketTunnelConnection) SetReadDeadline(t time.Time) error {
	return wstc.Conn.SetReadDeadline(t)
}

func (wstc *WebsocketTunnelConnection) SetWriteDeadline(t time.Time) error {
	return wstc.Conn.SetWriteDeadline(t)
}

// Unwrap returns the embedded net.Conn
func (wstc *WebsocketTunnelConnection) Unwrap() net.Conn {
	return wstc.Conn.UnderlyingConn()
}
