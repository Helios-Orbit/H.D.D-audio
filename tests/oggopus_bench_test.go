package tests

import (
    "testing"
    fa "github.com/Helios-Orbit/H.D.D-audio/fishaudio"
)

func makePages(n int, segs int, segLen int) []byte {
    var s []byte
    for i := 0; i < n; i++ {
        h := make([]byte, 27)
        copy(h[:4], []byte("OggS"))
        h[26] = byte(segs)
        l := make([]byte, segs)
        for j := range l { l[j] = byte(segLen) }
        p := make([]byte, segs*segLen)
        s = append(s, h...)
        s = append(s, l...)
        s = append(s, p...)
    }
    return s
}

func BenchmarkDemuxPush(b *testing.B) {
    data := makePages(2048, 4, 200)
    chunks := 1024
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        d := fa.NewOggOpusDemux()
        for off := 0; off < len(data); off += chunks {
            end := off + chunks
            if end > len(data) { end = len(data) }
            _ = d.Push(data[off:end])
        }
    }
}

func TestDemuxMaxBuf(t *testing.T) {
    d := fa.NewOggOpusDemux()
    d.MaxBuf = 1024
    data := makePages(100, 4, 300)
    for i := 0; i < len(data); i += 500 {
        end := i + 500
        if end > len(data) { end = len(data) }
        _ = d.Push(data[i:end])
        if len(data[i:end]) > d.MaxBuf {
            if len(data[i:end]) < d.MaxBuf { t.Fatalf("unexpected") }
        }
    }
    d.Reset()
    if len(d.Push([]byte{})) != 0 { t.Fatalf("bad reset") }
}