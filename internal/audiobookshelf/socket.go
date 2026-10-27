package audiobookshelf

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// Socket events that mean the user's listening progress may have changed.
// Playback sends user_item_progress_updated; manual changes such as marking a
// book finished send user_updated.
var progressEvents = map[string]bool{
	"user_item_progress_updated": true,
	"user_updated":               true,
}

const (
	minReconnectDelay = time.Second
	maxReconnectDelay = 5 * time.Minute
	handshakeTimeout  = 30 * time.Second
	// Used if the server's open packet has no heartbeat settings.
	defaultHeartbeat = 2 * time.Minute
	// user_updated carries all of the user's progress; it is streamed and discarded, not buffered.
	maxMessageSize = 256 << 20
)

// WatchProgress connects to the Audiobookshelf socket and calls onChange whenever
// listening progress may have changed. It reconnects with backoff until ctx is done,
// and calls onChange after every reconnect, since changes may have been missed meanwhile.
func (c *Client) WatchProgress(ctx context.Context, log *slog.Logger, onChange func()) {
	delay := minReconnectDelay
	for connected := false; ; {
		start := time.Now()
		err := c.listen(ctx, func() {
			log.Info("listening for progress updates from Audiobookshelf")
			if connected {
				onChange()
			}
			connected = true
		}, onChange)
		if ctx.Err() != nil {
			return
		}

		// A connection that stayed up for a while was healthy; start backing off afresh.
		if time.Since(start) > maxReconnectDelay {
			delay = minReconnectDelay
		}
		log.Warn("Audiobookshelf socket disconnected, reconnecting", "error", err, "in", delay)

		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}
		delay = min(delay*2, maxReconnectDelay)
	}
}

// listen runs one socket.io session: it connects, authenticates, then reads
// events until the connection fails. onReady is called once authenticated.
func (c *Client) listen(ctx context.Context, onReady, onChange func()) error {
	dialCtx, cancel := context.WithTimeout(ctx, handshakeTimeout)
	defer cancel()
	conn, _, err := websocket.Dial(dialCtx, socketURL(c.baseURL), nil)
	if err != nil {
		return err
	}
	defer conn.CloseNow()
	conn.SetReadLimit(maxMessageSize)

	// Engine.IO opens with its heartbeat settings; the server pings every
	// pingInterval and the connection is dead if a ping is pingTimeout late.
	msg, err := readEvent(ctx, conn, handshakeTimeout)
	if err != nil {
		return err
	}
	var open struct {
		PingInterval int `json:"pingInterval"`
		PingTimeout  int `json:"pingTimeout"`
	}
	if !strings.HasPrefix(msg, "0") || json.Unmarshal([]byte(msg[1:]), &open) != nil {
		return fmt.Errorf("unexpected engine.io handshake %q", truncate(msg))
	}
	heartbeat := time.Duration(open.PingInterval+open.PingTimeout) * time.Millisecond
	if heartbeat <= 0 {
		heartbeat = defaultHeartbeat
	}

	// Join the default socket.io namespace, then authenticate with the API key.
	if err := conn.Write(ctx, websocket.MessageText, []byte("40")); err != nil {
		return err
	}
	auth, err := json.Marshal([]string{"auth", c.apiKey})
	if err != nil {
		return err
	}
	authSent := false

	for {
		msg, err := readEvent(ctx, conn, heartbeat)
		if err != nil {
			return err
		}
		switch {
		case msg == "2": // engine.io ping
			if err := conn.Write(ctx, websocket.MessageText, []byte("3")); err != nil {
				return err
			}
		case msg == "1":
			return errors.New("server closed the connection")
		case strings.HasPrefix(msg, "40") && !authSent: // namespace joined
			if err := conn.Write(ctx, websocket.MessageText, append([]byte("42"), auth...)); err != nil {
				return err
			}
			authSent = true
		case strings.HasPrefix(msg, "44"):
			return fmt.Errorf("socket.io connection refused: %s", truncate(msg[2:]))
		case strings.HasPrefix(msg, "41"):
			return errors.New("server disconnected the socket")
		case msg == `42"init"`:
			onReady()
		case msg == `42"auth_failed"`:
			return errors.New("authentication failed, check ABS_API_KEY")
		case strings.HasPrefix(msg, `42"`) && progressEvents[strings.Trim(msg[2:], `"`)]:
			onChange()
		}
	}
}

// readEvent reads one message. Socket.io events ("42[name, data]") are reduced to
// `42"name"`, discarding the payload without buffering it. Other messages are returned whole.
func readEvent(ctx context.Context, conn *websocket.Conn, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, r, err := conn.Reader(ctx)
	if err != nil {
		return "", err
	}

	prefix := make([]byte, 2)
	n, err := io.ReadFull(r, prefix)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	if n < 2 || !bytes.Equal(prefix, []byte("42")) {
		rest, err := io.ReadAll(io.LimitReader(r, 64<<10))
		if err != nil {
			return "", err
		}
		_, _ = io.Copy(io.Discard, r)
		return string(prefix[:n]) + string(rest), nil
	}

	var name string
	dec := json.NewDecoder(r)
	if tok, err := dec.Token(); err == nil && tok == json.Delim('[') {
		if tok, err := dec.Token(); err == nil {
			name, _ = tok.(string)
		}
	}
	if _, err := io.Copy(io.Discard, io.MultiReader(dec.Buffered(), r)); err != nil {
		return "", err
	}
	return `42"` + name + `"`, nil
}

// socketURL turns the server URL (which may include a base path) into its socket.io endpoint.
func socketURL(baseURL string) string {
	u := baseURL
	switch {
	case strings.HasPrefix(u, "https://"):
		u = "wss://" + strings.TrimPrefix(u, "https://")
	case strings.HasPrefix(u, "http://"):
		u = "ws://" + strings.TrimPrefix(u, "http://")
	}
	return u + "/socket.io/?EIO=4&transport=websocket"
}

func truncate(s string) string {
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}
