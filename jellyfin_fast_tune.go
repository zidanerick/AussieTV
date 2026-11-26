package main

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// Fast tune shares a single validated copy-only MPEG-TS producer per channel
// and feed generation. Memory, sessions, clients and idle retention are bounded.
const (
	fastTunePackets = 32768 // ~6 MiB per active channel
	fastTuneChannels = 4
	fastTuneClients = 8 // per channel
	fastTuneIdle = 20 * time.Second
	fastTuneJoinTimeout = 24 * time.Second
	fastTuneWriteTimeout = 10 * time.Second
)

var fastTuneStarts, fastTuneReuses, fastTuneLagDrops atomic.Int64

type fastProducer func(http.ResponseWriter, *http.Request, Channel)

type fastTunePool struct {
	mu sync.Mutex
	sessions map[string]*fastTuneSession
	closed bool
	produce fastProducer
	packets int
	idle time.Duration
}

func newFastTunePool(produce fastProducer) *fastTunePool {
	return &fastTunePool{sessions: make(map[string]*fastTuneSession), produce: produce, packets:fastTunePackets, idle:fastTuneIdle}
}

type fastTuneSession struct {
	mu sync.Mutex
	ctx context.Context
	cancel context.CancelFunc
	ring []byte
	next, join, pat uint64 // absolute packet sequence numbers, not byte offsets
	hasJoin, hasPAT bool
	probe tsAudioProbe
	pendingJoin, pendingAudio bool
	candidate uint64
	changed chan struct{}
	ended bool
	status int
	clients int // pool lock
	timer *time.Timer // pool lock
}

func (s *fastTuneSession) notify() {
	close(s.changed)
	s.changed = make(chan struct{})
}

func (s *fastTuneSession) oldest() uint64 {
	capacity := uint64(len(s.ring)/188)
	if s.next > capacity { return s.next-capacity }
	return 0
}

func (s *fastTuneSession) appendPacket(p []byte) {
	seq := s.next
	off := int(seq % uint64(len(s.ring)/188))*188
	copy(s.ring[off:off+188],p)
	s.next++
	// A PAT followed by PMT, random-access video and actual audio is the
	// conservative join boundary. Random-access signalling is a hint, not
	// proof that a particular downstream decoder can show its first frame.
	if p[0]!=0x47 || p[1]&0x80!=0 { return }
	pid := uint16(p[1]&0x1f)<<8 | uint16(p[2])
	if pid==0 && p[1]&0x40!=0 {
		s.pat=seq
		s.hasPAT=true
	}
	s.probe.add(p)
	_, video := s.probe.videoPIDs[pid]
	_, audio := s.probe.audioPIDs[pid]
	randomAccess := p[3]&0x20!=0 && p[4]>0 && p[4]<183 && p[5]&0x40!=0
	if video && randomAccess && p[1]&0x40!=0 && s.hasPAT && s.pat>=s.oldest() && s.probe.hasAudio {
		s.candidate=s.pat
		s.pendingJoin=true
		s.pendingAudio=false
	}
	if audio && p[1]&0x40!=0 && p[3]&0x10!=0 && s.pendingJoin {
		s.pendingAudio=true
	}
	if s.pendingJoin && s.pendingAudio && s.candidate>=s.oldest() {
		s.join=s.candidate
		s.hasJoin=true
		s.pendingJoin=false
	}
}

type fastTuneWriter struct {
	s *fastTuneSession
	header http.Header
	tail []byte
}

func (w *fastTuneWriter) Header() http.Header { return w.header }
func (w *fastTuneWriter) WriteHeader(status int) {
	w.s.mu.Lock()
	defer w.s.mu.Unlock()
	if w.s.status==0 { w.s.status=status; w.s.notify() }
}
func (w *fastTuneWriter) SetWriteDeadline(time.Time) error { return nil }
func (w *fastTuneWriter) Flush() {}
func (w *fastTuneWriter) Write(p []byte) (int,error) {
	if err:=w.s.ctx.Err(); err!=nil { return 0,err }
	w.s.mu.Lock()
	defer w.s.mu.Unlock()
	if w.s.status==0 { w.s.status=200 }
	if w.s.status!=200 { return len(p),nil }
	n:=len(p)
	if len(w.tail)>0 {
		take:=min(188-len(w.tail),len(p))
		w.tail=append(w.tail,p[:take]...)
		p=p[take:]
		if len(w.tail)==188 {
			if w.tail[0]!=0x47 { return 0,errors.New("unaligned transport stream") }
			w.s.appendPacket(w.tail)
			w.tail=w.tail[:0]
		}
	}
	for len(p)>=188 {
		if p[0]!=0x47 { return 0,errors.New("unaligned transport stream") }
		w.s.appendPacket(p[:188])
		p=p[188:]
	}
	w.tail=append(w.tail,p...)
	w.s.notify()
	return n,nil
}

func (p *fastTunePool) acquire(ch Channel) (*fastTuneSession,bool,error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed { return nil,false,errors.New("feed is shutting down") }
	if s:=p.sessions[ch.ID]; s!=nil {
		s.mu.Lock()
		ended:=s.ended
		s.mu.Unlock()
		if !ended && s.ctx.Err()==nil {
			if s.clients>=fastTuneClients { return nil,false,errors.New("channel client limit reached") }
			if s.timer!=nil { s.timer.Stop(); s.timer=nil }
			s.clients++
			fastTuneReuses.Add(1)
			return s,true,nil
		}
		// Do not start a replacement until the old producer has exited.
		if !ended { return nil,false,errors.New("channel is stopping") }
		delete(p.sessions,ch.ID)
	}
	if len(p.sessions)>=fastTuneChannels { return nil,false,errors.New("fast tune channel limit reached") }
	ctx,cancel:=context.WithCancel(context.Background())
	s:=&fastTuneSession{ctx:ctx,cancel:cancel,ring:make([]byte,p.packets*188),changed:make(chan struct{}),clients:1}
	p.sessions[ch.ID]=s
	fastTuneStarts.Add(1)
	go func(){
		defer cancel()
		r,_:=http.NewRequestWithContext(ctx,http.MethodGet,"http://localhost/",nil)
		p.produce(&fastTuneWriter{s:s,header:make(http.Header)},r,ch)
		s.mu.Lock()
		s.ended=true
		s.notify()
		s.mu.Unlock()
		p.mu.Lock()
		if p.sessions[ch.ID]==s { delete(p.sessions,ch.ID) }
		if s.timer!=nil { s.timer.Stop(); s.timer=nil }
		p.mu.Unlock()
	}()
	return s,false,nil
}

func (p *fastTunePool) release(s *fastTuneSession) {
	p.mu.Lock()
	defer p.mu.Unlock()
	s.clients--
	if s.clients!=0 || p.closed || s.ctx.Err()!=nil { return }
	// Identity check prevents a stale timer from stopping a reused session.
	var timer *time.Timer
	timer=time.AfterFunc(p.idle,func(){
		p.mu.Lock()
		defer p.mu.Unlock()
		if s.clients==0 && s.timer==timer { s.timer=nil; s.cancel() }
	})
	s.timer=timer
}

func (p *fastTunePool) close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed=true
	for _,s:=range p.sessions {
		if s.timer!=nil { s.timer.Stop(); s.timer=nil }
		s.cancel()
	}
}

func (p *fastTunePool) serve(w http.ResponseWriter,r *http.Request,ch Channel) {
	start:=time.Now()
	s,reused,err:=p.acquire(ch)
	if err!=nil { http.Error(w,err.Error(),503); return }
	defer p.release(s)
	timer:=time.NewTimer(fastTuneJoinTimeout)
	defer timer.Stop()
	var cursor uint64
	for {
		s.mu.Lock()
		if s.status==200 && s.hasJoin && s.join>=s.oldest() {
			cursor=s.join
			s.mu.Unlock()
			break
		}
		ended,status,changed:=s.ended,s.status,s.changed
		s.mu.Unlock()
		if ended || status>=400 {
			if status<400 { status=502 }
			http.Error(w,"Fast tune stream unavailable; check diagnostics",status)
			recordTune(ch.ID,"fast-tune-unavailable",status,time.Since(start),0)
			return
		}
		select {
		case <-r.Context().Done(): return
		case <-s.ctx.Done(): http.Error(w,"Fast tune stream stopped",502); return
		case <-timer.C:
			http.Error(w,"Fast tune join timed out",504)
			recordTune(ch.ID,"fast-tune-join-timeout",504,time.Since(start),0)
			return
		case <-changed:
		}
	}
	w.Header().Set("Content-Type","video/mp2t")
	w.Header().Set("Cache-Control","no-store")
	w.Header().Set("X-Content-Type-Options","nosniff")
	rc:=http.NewResponseController(w)
	// Every reader has its own cursor. No unbounded per-client queues and no
	// blocking the producer on a slow network connection.
	buf:=make([]byte,188*128)
	first:=true
	for {
		s.mu.Lock()
		if cursor<s.oldest() {
			s.mu.Unlock()
			fastTuneLagDrops.Add(1)
			if first { http.Error(w,"Fast tune buffer expired",503) }
			return
		}
		n:=int(min(s.next-cursor,uint64(len(buf)/188)))
		for i:=0;i<n;i++ {
			off:=int((cursor+uint64(i))%uint64(len(s.ring)/188))*188
			copy(buf[i*188:(i+1)*188],s.ring[off:off+188])
		}
		ended,changed:=s.ended,s.changed
		s.mu.Unlock()
		if n>0 {
			_ = rc.SetWriteDeadline(time.Now().Add(fastTuneWriteTimeout))
			if _,err:=w.Write(buf[:n*188]);err!=nil { return }
			if err:=rc.Flush();err!=nil { return }
			cursor+=uint64(n)
			if first {
				mode:="fast-tune-cold"
				if reused { mode="fast-tune-shared" }
				recordTune(ch.ID,mode,200,time.Since(start),0)
				first=false
			}
			continue
		}
		if ended { return }
		select {
		case <-r.Context().Done(): return
		case <-s.ctx.Done(): return
		case <-changed:
		}
	}
}
