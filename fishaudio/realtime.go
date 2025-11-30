package fishaudio

import (
    "context"
    "crypto/tls"
    "net/http"
    "strings"
    "time"
    "sync/atomic"
    "github.com/gorilla/websocket"
)

type RealtimeConnection struct {
    Open  chan struct{}
    Audio chan []byte
    Packets chan []byte
    Error chan error
    Close chan struct{}
    ws    *websocket.Conn
    release func()
    force   func()
    closed  uint32
}

func (c *Client) ConvertRealtime(ctx context.Context, req TTSRequest, texts <-chan string, backend string) (*RealtimeConnection, error) {
    u := c.BaseURL
    if strings.HasPrefix(strings.ToLower(u), "https://") {
        u = "wss://" + strings.TrimPrefix(u, "https://")
    } else if strings.HasPrefix(strings.ToLower(u), "http://") {
        u = "ws://" + strings.TrimPrefix(u, "http://")
    }
    u += "/v1/tts/live"
    d := websocket.Dialer{HandshakeTimeout: 15 * time.Second, TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
    h := http.Header{}
    h.Set("Authorization", "Bearer "+c.APIKey)
    h.Set("model", backend)
    key := c.BaseURL + "|" + strings.ToLower(backend) + "|"
    if req.Format != nil { key += strings.ToLower(*req.Format) }
    key += "|"
    if req.ReferenceID != nil { key += *req.ReferenceID }
    var ws *websocket.Conn
    var release func()
    var force func()
    if c.Options.DefaultPooling && c.Pool != nil {
        w, r, f, err := c.Pool.Acquire(ctx, key, func() (*websocket.Conn, *http.Response, error) { return d.DialContext(ctx, u, h) })
        if err != nil { return nil, err }
        ws, release, force = w, r, f
    } else {
        w, _, err := d.DialContext(ctx, u, h)
        if err != nil { return nil, err }
        ws = w
        release = func() {}
        force = func() { _ = ws.Close() }
    }
    ab := c.Options.AudioBuf
    pb := c.Options.PacketsBuf
    if ab <= 0 { ab = 256 }
    if pb <= 0 { pb = 1024 }
    conn := &RealtimeConnection{Open: make(chan struct{}, 1), Audio: make(chan []byte, ab), Packets: make(chan []byte, pb), Error: make(chan error, 1), Close: make(chan struct{}, 1), ws: ws, release: release, force: force}
    conn.Open <- struct{}{}
    _ = writeEvent(ws, StartEvent{Event: "start", Request: req})
    go func() {
        for t := range texts {
            if err := writeEvent(ws, TextEvent{Event: "text", Text: t}); err != nil {
                if isAbnormalCloseError(err) { conn.ForceClose() ; return }
                select { case conn.Error <- err: default: }
                return
            }
            if c.Pool != nil { c.Pool.TouchText(ws) }
            if err := writeEvent(ws, FlushEvent{Event: "flush"}); err != nil {
                if isAbnormalCloseError(err) { conn.ForceClose() ; return }
                select { case conn.Error <- err: default: }
                return
            }
        }
    }()
    go func() {
        defer func() { if atomic.CompareAndSwapUint32(&conn.closed, 0, 1) { close(conn.Close) } }()
        var demux *OggOpusDemux
        if req.Format != nil {
            f := strings.ToLower(*req.Format)
            if f == "opus" { demux = NewOggOpusDemux() }
        }
        for {
            _, data, err := ws.ReadMessage()
            if err != nil {
                if isAbnormalCloseError(err) { conn.ForceClose() ; return }
                select { case conn.Error <- err: default: }
                return
            }
            var ev BaseEvent
            if err := decodeEvent(data, &ev); err != nil { select { case conn.Error <- err: default: } ; return }
            if ev.Event == "audio" {
                if ev.Audio != nil {
                    conn.Audio <- ev.Audio
                    if demux != nil { for _, p := range demux.Push(ev.Audio) { conn.Packets <- p } }
                }
            } else if ev.Event == "finish" {
                if ev.Reason == "error" { select { case conn.Error <- &finishError{ev.Message}: default: } }
                if demux != nil { close(conn.Packets) }
                return
            }
        }
    }()
    return conn, nil
}

type finishError struct{ s string }
func (e *finishError) Error() string { return e.s }

func (c *RealtimeConnection) Release() { if c.release != nil { c.release() } }

func (c *RealtimeConnection) ForceClose() {
    _ = writeEvent(c.ws, StopEvent{Event: "stop"})
    if c.force != nil { c.force() }
}

func (c *RealtimeConnection) DoneCh() <-chan struct{} { return c.Close }

func (c *RealtimeConnection) Stop() { _ = writeEvent(c.ws, StopEvent{Event: "stop"}) }

func isAbnormalCloseError(err error) bool {
    if err == nil { return false }
    s := err.Error()
    return strings.Contains(s, "close 1006") || strings.Contains(s, "close 1005") || strings.Contains(s, "unexpected EOF")
}