package tests

import (
    "testing"
    "github.com/vmihailenco/msgpack/v5"
    fa "github.com/Helios-Orbit/H.D.D-audio/fishaudio"
)

func BenchmarkEncodeStruct(b *testing.B) {
    ev := fa.TextEvent{Event: "text", Text: "hello"}
    for i := 0; i < b.N; i++ {
        _, _ = msgpack.Marshal(ev)
    }
}

func BenchmarkEncodeMap(b *testing.B) {
    m := map[string]interface{}{"event": "text", "text": "hello"}
    for i := 0; i < b.N; i++ {
        _, _ = msgpack.Marshal(m)
    }
}

func BenchmarkDecodeStruct(b *testing.B) {
    ev := fa.TextEvent{Event: "text", Text: "hello"}
    bs, _ := msgpack.Marshal(ev)
    var out fa.BaseEvent
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        _ = msgpack.Unmarshal(bs, &out)
    }
}
