package fishaudio

import (
    "context"
    "crypto/tls"
    "net/http"
    "strings"
    "time"
    "github.com/gorilla/websocket"
    "github.com/vmihailenco/msgpack/v5"
)

type RealtimeConnection struct {
    Open  chan struct{}
    Audio chan []byte
    Error chan error
    Close chan struct{}
    ws    *websocket.Conn
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
    ws, _, err := d.DialContext(ctx, u, h)
    if err != nil {
        return nil, err
    }
    conn := &RealtimeConnection{Open: make(chan struct{}, 1), Audio: make(chan []byte, 256), Error: make(chan error, 1), Close: make(chan struct{}, 1), ws: ws}
    conn.Open <- struct{}{}
    start := map[string]interface{}{"event": "start", "request": req}
    sb, _ := msgpack.Marshal(start)
    _ = ws.WriteMessage(websocket.BinaryMessage, sb)
    go func() {
        for t := range texts {
            m := map[string]interface{}{"event": "text", "text": t}
            b, _ := msgpack.Marshal(m)
            _ = ws.WriteMessage(websocket.BinaryMessage, b)
            f := map[string]interface{}{"event": "flush"}
            fb, _ := msgpack.Marshal(f)
            _ = ws.WriteMessage(websocket.BinaryMessage, fb)
        }
        st := map[string]interface{}{"event": "stop"}
        b, _ := msgpack.Marshal(st)
        _ = ws.WriteMessage(websocket.BinaryMessage, b)
    }()
    go func() {
        defer func() { close(conn.Close) ; _ = ws.Close() }()
        for {
            _, data, err := ws.ReadMessage()
            if err != nil {
                select { case conn.Error <- err: default: }
                return
            }
            var dec map[string]interface{}
            _ = msgpack.Unmarshal(data, &dec)
            ev, _ := dec["event"].(string)
            if ev == "audio" {
                if a, ok := dec["audio"].([]byte); ok { conn.Audio <- a }
            } else if ev == "finish" {
                r, _ := dec["reason"].(string)
                if r == "error" { msg, _ := dec["message"].(string); select { case conn.Error <- &finishError{msg}: default: } }
                return
            }
        }
    }()
    return conn, nil
}

type finishError struct{ s string }
func (e *finishError) Error() string { return e.s }