package streams

import (
	"github.com/bokysan/socketace/v2/internal/util"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"io"
)

// Implement a ReadWriteCloser over a websocket connection
type WebsocketReadWriteCloser struct {
	conn *websocket.Conn
}

func NewWebsocketReadWriteCloser(conn *websocket.Conn) *WebsocketReadWriteCloser {
	return &WebsocketReadWriteCloser{
		conn: conn,
	}
}

func (wsc *WebsocketReadWriteCloser) Read(p []byte) (int, error) {
	messageType, message, err := wsc.conn.ReadMessage()
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
func (wsc *WebsocketReadWriteCloser) Write(p []byte) (int, error) {
	dataLen := len(p)
	for {
		if len(p) > util.BufferSize {
			err := wsc.conn.WriteMessage(websocket.BinaryMessage, p[:util.BufferSize])
			if err != nil {
				return 0, errors.WithStack(err)
			}
			p = p[util.BufferSize:]
		} else {
			err := wsc.conn.WriteMessage(websocket.BinaryMessage, p)
			if err != nil {
				return 0, errors.WithStack(err)
			}
			break
		}
	}
	return dataLen, nil
}

func (wsc *WebsocketReadWriteCloser) Close() error {
	return wsc.conn.Close()
}
