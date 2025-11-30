package tests

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
    "github.com/gorilla/websocket"
    fa "github.com/Helios-Orbit/H.D.D-audio/fishaudio"
)

func TestWSConnPoolAcquireRelease(t *testing.T) {
    up := websocket.Upgrader{}
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        c, err := up.Upgrade(w, r, nil)
        if err != nil { return }
        _ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
    }))
    defer srv.Close()

    u := "ws://" + srv.Listener.Addr().String()
    h := http.Header{}
    p := fa.NewWSConnPool(2, 2*time.Second, 1*time.Minute, 1*time.Minute)
    key := "k"
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    ws1, rel1, force1, err := p.Acquire(ctx, key, func() (*websocket.Conn, *http.Response, error) {
        d := websocket.Dialer{}
        return d.DialContext(ctx, u, h)
    })
    if err != nil { t.Fatalf("dial 1: %v", err) }
    ws2, rel2, _, err := p.Acquire(ctx, key, func() (*websocket.Conn, *http.Response, error) {
        d := websocket.Dialer{}
        return d.DialContext(ctx, u, h)
    })
    if err != nil { t.Fatalf("dial 2: %v", err) }
    if ws1 == ws2 { t.Fatalf("expected different conns") }
    rel1()
    rel2()
    force1()
}
