package fishaudio

import (
    "context"
    "sync"
    "time"
    "net/http"
    "github.com/gorilla/websocket"
)

type ClientOptions struct {
    DefaultPooling bool
    MaxConnsPerKey int
    IdleTTL        time.Duration
    MaxLife        time.Duration
    WSReadTimeout  time.Duration
    WSPingInterval time.Duration
    AudioBuf       int
    PacketsBuf     int
    TextIdleTTL    time.Duration
}

type WSConnPool struct {
    mu              sync.Mutex
    m               map[string]*keyPool
    wsIndex         map[*websocket.Conn]*poolEntry
    maxPerKey       int
    idleTTL         time.Duration
    maxLife         time.Duration
    textIdleTTL     time.Duration
}

type keyPool struct {
    mu      sync.Mutex
    entries []*poolEntry
    waiters []chan *poolEntry
}

type poolEntry struct {
    ws       *websocket.Conn
    busy     bool
    created  time.Time
    lastUsed time.Time
    lastText time.Time
}

func NewWSConnPool(maxPerKey int, idleTTL time.Duration, maxLife time.Duration, textIdleTTL time.Duration) *WSConnPool {
    if maxPerKey <= 0 { maxPerKey = 4 }
    if idleTTL <= 0 { idleTTL = 60 * time.Second }
    if maxLife <= 0 { maxLife = 10 * time.Minute }
    if textIdleTTL <= 0 { textIdleTTL = 2 * time.Minute }
    p := &WSConnPool{m: make(map[string]*keyPool), wsIndex: make(map[*websocket.Conn]*poolEntry), maxPerKey: maxPerKey, idleTTL: idleTTL, maxLife: maxLife, textIdleTTL: textIdleTTL}
    go p.reapLoop()
    return p
}

func (p *WSConnPool) get(key string) *keyPool {
    p.mu.Lock()
    kp := p.m[key]
    if kp == nil { kp = &keyPool{}; p.m[key] = kp }
    p.mu.Unlock()
    return kp
}

func (p *WSConnPool) Acquire(ctx context.Context, key string, dial func() (*websocket.Conn, *http.Response, error)) (*websocket.Conn, func(), func(), error) {
    kp := p.get(key)
    for {
        entry, release, forceClose := p.tryAcquire(kp, key)
        if entry != nil { return entry.ws, release, forceClose, nil }
        kp.mu.Lock()
        if len(kp.entries) < p.maxPerKey {
            kp.mu.Unlock()
            ws, _, err := dial()
            if err != nil { return nil, nil, nil, err }
            e := &poolEntry{ws: ws, busy: true, created: time.Now(), lastUsed: time.Now()}
            kp.mu.Lock()
            kp.entries = append(kp.entries, e)
            kp.mu.Unlock()
            p.mu.Lock()
            p.wsIndex[ws] = e
            p.mu.Unlock()
            release := func() { p.release(key, e) }
            force := func() { p.forceClose(key, e) }
            return ws, release, force, nil
        }
        ch := make(chan *poolEntry, 1)
        kp.waiters = append(kp.waiters, ch)
        kp.mu.Unlock()
        select {
        case e := <-ch:
            if e == nil { continue }
            release := func() { p.release(key, e) }
            force := func() { p.forceClose(key, e) }
            return e.ws, release, force, nil
        case <-ctx.Done():
            return nil, nil, nil, ctx.Err()
        }
    }
}

func (p *WSConnPool) tryAcquire(kp *keyPool, key string) (*poolEntry, func(), func()) {
    now := time.Now()
    kp.mu.Lock()
    var chosen *poolEntry
    var i int
    for i = 0; i < len(kp.entries); i++ {
        e := kp.entries[i]
        if p.expired(e, now) {
            _ = e.ws.Close()
            kp.entries[i] = kp.entries[len(kp.entries)-1]
            kp.entries = kp.entries[:len(kp.entries)-1]
            i--
            continue
        }
        if !e.busy {
            e.busy = true
            e.lastUsed = now
            chosen = e
            break
        }
    }
    kp.mu.Unlock()
    if chosen == nil { return nil, nil, nil }
    release := func() { p.release(key, chosen) }
    force := func() { p.forceClose(key, chosen) }
    return chosen, release, force
}

func (p *WSConnPool) expired(e *poolEntry, now time.Time) bool {
    if p.maxLife > 0 && now.Sub(e.created) > p.maxLife { return true }
    if p.idleTTL > 0 && !e.busy && now.Sub(e.lastUsed) > p.idleTTL { return true }
    if p.textIdleTTL > 0 && !e.busy {
        if !e.lastText.IsZero() && now.Sub(e.lastText) > p.textIdleTTL { return true }
        if e.lastText.IsZero() && now.Sub(e.lastUsed) > p.textIdleTTL { return true }
    }
    return false
}

func (p *WSConnPool) release(key string, e *poolEntry) {
    kp := p.get(key)
    kp.mu.Lock()
    e.busy = false
    e.lastUsed = time.Now()
    if len(kp.waiters) > 0 {
        ch := kp.waiters[0]
        kp.waiters = kp.waiters[1:]
        e.busy = true
        ch <- e
        close(ch)
    }
    kp.mu.Unlock()
}

func (p *WSConnPool) forceClose(key string, e *poolEntry) {
    _ = e.ws.Close()
    kp := p.get(key)
    kp.mu.Lock()
    for i := 0; i < len(kp.entries); i++ {
        if kp.entries[i] == e {
            kp.entries[i] = kp.entries[len(kp.entries)-1]
            kp.entries = kp.entries[:len(kp.entries)-1]
            break
        }
    }
    if len(kp.waiters) > 0 {
        ch := kp.waiters[0]
        kp.waiters = kp.waiters[1:]
        ch <- nil
        close(ch)
    }
    kp.mu.Unlock()
    p.mu.Lock()
    delete(p.wsIndex, e.ws)
    p.mu.Unlock()
}

func (p *WSConnPool) TouchText(ws *websocket.Conn) {
    p.mu.Lock()
    if e, ok := p.wsIndex[ws]; ok { e.lastText = time.Now() }
    p.mu.Unlock()
}

func (p *WSConnPool) reapLoop() {
    t := time.NewTicker(5 * time.Second)
    defer t.Stop()
    for range t.C {
        now := time.Now()
        p.mu.Lock()
        for _, kp := range p.m {
            kp.mu.Lock()
            for i := 0; i < len(kp.entries); i++ {
                e := kp.entries[i]
                if p.expired(e, now) {
                    _ = e.ws.Close()
                    delete(p.wsIndex, e.ws)
                    kp.entries[i] = kp.entries[len(kp.entries)-1]
                    kp.entries = kp.entries[:len(kp.entries)-1]
                    i--
                }
            }
            kp.mu.Unlock()
        }
        p.mu.Unlock()
    }
}