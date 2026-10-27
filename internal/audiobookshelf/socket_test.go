package audiobookshelf

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// fakeSocketServer speaks just enough Engine.IO/Socket.IO to stand in for Audiobookshelf.
// script runs after the handshake and authentication.
func fakeSocketServer(t *testing.T, apiKey string, script func(ctx context.Context, conn *websocket.Conn)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/abs/socket.io/" || r.URL.Query().Get("EIO") != "4" {
			http.NotFound(w, r)
			return
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.CloseNow()
		ctx := r.Context()

		send := func(msg string) { _ = conn.Write(ctx, websocket.MessageText, []byte(msg)) }
		expect := func(want string) bool {
			_, got, err := conn.Read(ctx)
			if err != nil || string(got) != want {
				t.Errorf("client sent %q (%v), want %q", got, err, want)
				return false
			}
			return true
		}

		send(`0{"sid":"e1","upgrades":[],"pingInterval":25000,"pingTimeout":20000,"maxPayload":1000000}`)
		if !expect("40") {
			return
		}
		send(`40{"sid":"s1"}`)
		if !expect(`42["auth","` + apiKey + `"]`) {
			return
		}
		if apiKey != "key" {
			send(`42["auth_failed",{"message":"Invalid token"}]`)
			return
		}
		send(`42["init",{"userId":"u1","username":"reader"}]`)
		script(ctx, conn)
	}))
}

func TestWatchProgress(t *testing.T) {
	connections := atomic.Int32{}
	server := fakeSocketServer(t, "key", func(ctx context.Context, conn *websocket.Conn) {
		if connections.Add(1) > 1 {
			<-ctx.Done()
			return
		}
		send := func(msg string) { _ = conn.Write(ctx, websocket.MessageText, []byte(msg)) }

		// Pings must be answered.
		send("2")
		if _, msg, err := conn.Read(ctx); err != nil || string(msg) != "3" {
			t.Errorf("ping answered with %q (%v), want 3", msg, err)
		}

		// user_updated carries all progress and can be far larger than the default read limit.
		send(`42["user_updated",{"mediaProgress":"` + strings.Repeat("x", 1<<20) + `"}]`)
		send(`42["library_item_updated",{}]`)
		send(`42["user_item_progress_updated",{"id":"p1","data":{}}]`)
		// Dropping the connection makes the client reconnect.
	})
	defer server.Close()

	changes := make(chan struct{}, 10)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	client := New(server.URL+"/abs", "key", time.Second)
	go client.WatchProgress(ctx, slog.New(slog.NewTextHandler(io.Discard, nil)), func() { changes <- struct{}{} })

	// Two progress events, then one more after reconnecting.
	for i := range 3 {
		select {
		case <-changes:
		case <-time.After(5 * time.Second):
			t.Fatalf("got %d changes, want 3", i)
		}
	}
	select {
	case <-changes:
		t.Error("got an unexpected change")
	case <-time.After(100 * time.Millisecond):
	}
	if n := connections.Load(); n != 2 {
		t.Errorf("connected %d times, want 2", n)
	}
}

func TestListenAuthFailed(t *testing.T) {
	server := fakeSocketServer(t, "wrong", func(context.Context, *websocket.Conn) {})
	defer server.Close()

	err := New(server.URL+"/abs", "wrong", time.Second).listen(t.Context(), func() {
		t.Error("ready after failed authentication")
	}, func() {})
	if err == nil || !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("got error %v", err)
	}
}

func TestSocketURL(t *testing.T) {
	for base, want := range map[string]string{
		"http://abs.local:13378":    "ws://abs.local:13378/socket.io/?EIO=4&transport=websocket",
		"https://example.com/audio": "wss://example.com/audio/socket.io/?EIO=4&transport=websocket",
	} {
		if got := socketURL(base); got != want {
			t.Errorf("socketURL(%q) = %q, want %q", base, got, want)
		}
	}
}
