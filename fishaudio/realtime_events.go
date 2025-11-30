package fishaudio

import (
    "bytes"
    "sync"
    "github.com/vmihailenco/msgpack/v5"
    "github.com/gorilla/websocket"
)

type StartEvent struct {
    Event   string    `msgpack:"event"`
    Request TTSRequest `msgpack:"request"`
}

type TextEvent struct {
    Event string `msgpack:"event"`
    Text  string `msgpack:"text"`
}

type FlushEvent struct {
    Event string `msgpack:"event"`
}

type StopEvent struct {
    Event string `msgpack:"event"`
}

type BaseEvent struct {
    Event   string `msgpack:"event"`
    Reason  string `msgpack:"reason,omitempty"`
    Message string `msgpack:"message,omitempty"`
    Audio   []byte `msgpack:"audio,omitempty"`
}

var bufPool = sync.Pool{New: func() interface{} { return new(bytes.Buffer) }}

func writeEvent(ws *websocket.Conn, v interface{}) error {
    b := bufPool.Get().(*bytes.Buffer)
    b.Reset()
    enc := msgpack.NewEncoder(b)
    if err := enc.Encode(v); err != nil {
        bufPool.Put(b)
        return err
    }
    err := ws.WriteMessage(websocket.BinaryMessage, b.Bytes())
    b.Reset()
    bufPool.Put(b)
    return err
}

func decodeEvent(data []byte, out *BaseEvent) error {
    return msgpack.Unmarshal(data, out)
}