package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"foundryprotocol/protocol"
	"foundryprotocol/server"
)

// TestGatewayBrowseAndPlayTogether covers the full start-to-finish journey a
// player sees on the client: start a world server, register it with a gateway,
// browse the gateway's server list over HTTP, connect two clients through the
// listed ws_url, and verify one player observes the other's chat. This is the
// same path the Godot client's Browse-Servers screen takes.
func TestGatewayBrowseAndPlayTogether(t *testing.T) {
	// 1. Start a real world server.
	worldLogger := zerolog.Nop()
	world, err := server.New(server.Config{
		Addr:       "127.0.0.1:0",
		WorldName:  "integration",
		ContentDir: "../content",
		AssetDir:   "../assets",
		SaveDir:    t.TempDir(),
		TPS:        20,
		Dev:        true,
	}, worldLogger)
	if err != nil {
		t.Fatalf("start world server: %v", err)
	}
	if err := world.Listen(); err != nil {
		t.Fatalf("world listen: %v", err)
	}
	worldCtx, cancelWorld := context.WithCancel(context.Background())
	defer cancelWorld()
	go func() { _ = world.Serve(worldCtx) }()

	// 2. Start the gateway in-process on an ephemeral port.
	reg, err := LoadRegistry(t.TempDir() + "/servers.yaml")
	if err != nil {
		t.Fatalf("load registry: %v", err)
	}
	gw := NewServer(Config{APIKey: "test-key"}, reg, zerolog.Nop())
	ts := httptest.NewServer(gw.Handler())
	defer ts.Close()

	wsURL := "ws://" + world.Addr() + "/ws"
	gwURL := strings.TrimPrefix(ts.URL, "http://")

	// 3. Register the world server with the gateway (the flow cmd/world will use).
	body, _ := json.Marshal(ServerInfo{
		ID:         "integ",
		Name:       "Integration World",
		WSURL:      wsURL,
		Owner:      "tester",
		MaxPlayers: 8,
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/servers", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-key")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("register status %d", resp.StatusCode)
	}

	// 4. Browse the gateway's server list, the same thing the client does.
	listResp, err := http.Get(ts.URL + "/servers")
	if err != nil {
		t.Fatalf("list servers: %v", err)
	}
	defer listResp.Body.Close()
	var list struct {
		Servers []ServerInfo `json:"servers"`
	}
	if err := json.NewDecoder(listResp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list.Servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(list.Servers))
	}
	if list.Servers[0].WSURL != wsURL {
		t.Fatalf("unexpected ws_url %q", list.Servers[0].WSURL)
	}
	t.Logf("browsed gateway %s, found %q at %s", gwURL, list.Servers[0].Name, wsURL)

	// 5. Two players connect through the listed URL and meet each other.
	alice := dialPlayer(t, wsURL, "Alice")
	bob := dialPlayer(t, wsURL, "Bob")

	bob.Socket.WriteJSON(protocol.Message{Type: protocol.TypeChat, Text: "hello alice"})
	got := waitForType(t, alice.Socket, protocol.TypeChatReceived, func(m protocol.Message) bool {
		return m.Text == "hello alice"
	})
	if got.PlayerName != "Bob" {
		t.Fatalf("expected Alice to see Bob's message, got sender %q", got.PlayerName)
	}
	alice.Socket.WriteJSON(protocol.Message{Type: protocol.TypeChat, Text: "hey bob"})
	check := waitForType(t, bob.Socket, protocol.TypeChatReceived, func(m protocol.Message) bool {
		return m.Text == "hey bob"
	})
	if check.PlayerName != "Alice" {
		t.Fatalf("expected Bob to see Alice's message, got sender %q", check.PlayerName)
	}
}

type connPair struct {
	Socket *websocket.Conn
	Name   string
}

func dialPlayer(t *testing.T, wsURL, name string) *connPair {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", name, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.WriteJSON(protocol.Message{Type: protocol.TypeHello, Name: name}); err != nil {
		t.Fatalf("%s hello: %v", name, err)
	}
	pair := &connPair{Socket: conn, Name: name}
	waitForType(t, conn, protocol.TypeWelcome, func(m protocol.Message) bool { return m.PlayerName == name })
	return pair
}

func waitForType(t *testing.T, conn *websocket.Conn, wantType string, check func(protocol.Message) bool) protocol.Message {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			if time.Now().Before(deadline) {
				continue
			}
			t.Fatalf("read: %v", err)
		}
		var msg protocol.Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}
		if msg.Type == wantType && check(msg) {
			return msg
		}
	}
	t.Fatalf("timed out waiting for %q", wantType)
	return protocol.Message{}
}
