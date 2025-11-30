package fishaudio

import "bytes"

var oggSig = []byte("OggS")

type OggOpusDemux struct {
    buf    []byte
    cur    []byte
    MaxBuf int
}

func NewOggOpusDemux() *OggOpusDemux { return &OggOpusDemux{MaxBuf: 2 << 20} }

func (d *OggOpusDemux) Reset() { d.buf = nil; d.cur = nil }

func (d *OggOpusDemux) Push(b []byte) [][]byte {
    d.buf = append(d.buf, b...)
    if d.MaxBuf > 0 && len(d.buf) > d.MaxBuf {
        i := bytes.LastIndex(d.buf, oggSig)
        if i >= 0 { d.buf = d.buf[i:] }
        if len(d.buf) > d.MaxBuf { d.buf = d.buf[len(d.buf)-d.MaxBuf:] }
    }
    var out [][]byte
    for {
        i := bytes.Index(d.buf, oggSig)
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
            seg := int(v)
            d.cur = append(d.cur, payload[off:off+seg]...)
            off += seg
            if v < 255 {
                out = append(out, d.cur)
                d.cur = nil
            }
        }
        d.buf = d.buf[hlen+plen:]
    }
    return out
}