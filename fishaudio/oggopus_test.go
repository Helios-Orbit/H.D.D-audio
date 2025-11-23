package fishaudio

import "testing"

func page(laces []byte, payload []byte) []byte {
    h := make([]byte, 27)
    copy(h[:4], []byte("OggS"))
    h[26] = byte(len(laces))
    seg := append([]byte{}, laces...)
    return append(append(h, seg...), payload...)
}

func TestDemuxBasic(t *testing.T) {
    d := NewOggOpusDemux()
    p1 := []byte("OpusHead")
    p2 := []byte("OpusTags")
    a1 := []byte("AAAA")
    a2 := make([]byte, 300)
    for i := range a2 { a2[i] = 1 }
    l1 := []byte{byte(len(p1)), byte(len(p2)), byte(len(a1)), 255}
    pay1 := append(append(append([]byte{}, p1...), p2...), a1...)
    pay1 = append(pay1, a2[:255]...)
    g1 := page(l1, pay1)
    l2 := []byte{45}
    g2 := page(l2, a2[255:])
    out1 := d.Push(g1)
    if len(out1) != 3 { t.Fatalf("got %d", len(out1)) }
    if string(out1[0]) != "OpusHead" { t.Fatalf("bad head") }
    if string(out1[1]) != "OpusTags" { t.Fatalf("bad tags") }
    if string(out1[2]) != "AAAA" { t.Fatalf("bad a1") }
    out2 := d.Push(g2)
    if len(out2) != 1 { t.Fatalf("got %d", len(out2)) }
    if len(out2[0]) != 300 { t.Fatalf("bad a2 %d", len(out2[0])) }
}