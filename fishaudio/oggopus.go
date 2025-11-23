package fishaudio

import "bytes"

type OggOpusDemux struct {
    buf []byte
    cur []byte
}

func NewOggOpusDemux() *OggOpusDemux { return &OggOpusDemux{} }

func (d *OggOpusDemux) Push(b []byte) [][]byte {
    d.buf = append(d.buf, b...)
    var out [][]byte
    sig := []byte("OggS")
    for {
        i := bytes.Index(d.buf, sig)
        if i < 0 { break }
        if i > 0 { d.buf = d.buf[i:] }
        if len(d.buf) < 27 { break }
        ps := int(d.buf[26])
        hlen := 27 + ps
        if len(d.buf) < hlen { break }
        l := d.buf[27:hlen]
        plen := 0
        for _, v := range l { plen += int(v) }
        if len(d.buf) < hlen+plen { break }
        payload := d.buf[hlen : hlen+plen]
        off := 0
        for _, v := range l {
            if v == 0 { continue }
            d.cur = append(d.cur, payload[off:off+int(v)]...)
            off += int(v)
            if v < 255 {
                out = append(out, d.cur)
                d.cur = nil
            }
        }
        d.buf = d.buf[hlen+plen:]
    }
    return out
}