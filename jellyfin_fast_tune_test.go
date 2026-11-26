package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func waitFastTune(t *testing.T, check func() bool) {
	t.Helper()
	deadline:=time.Now().Add(5*time.Second)
	for time.Now().Before(deadline) {
		if check() { return }
		time.Sleep(10*time.Millisecond)
	}
	t.Fatal("fast tune condition did not become true")
}

func TestFastTuneSharedWarmJoinAndIdleCleanup(t *testing.T) {
	fixture:=makeTestTSPackets(t,true)
	var started atomic.Int64
	var stopped atomic.Int64
	p:=newFastTunePool(func(w http.ResponseWriter,r *http.Request,ch Channel){
		started.Add(1)
		defer stopped.Add(1)
		w.WriteHeader(200)
		// Deliberately split packet boundaries.
		if _,err:=w.Write(fixture[:91]);err!=nil { return }
		if _,err:=w.Write(fixture[91:]);err!=nil { return }
		<-r.Context().Done()
	})
	p.idle=150*time.Millisecond
	defer p.close()
	ch:=sampleFeed().Channels[0]
	server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){p.serve(w,r,ch)}))
	defer server.Close()
	client:=&http.Client{Timeout:5*time.Second}
	open:=func() *http.Response {
		t.Helper()
		resp,err:=client.Get(server.URL)
		if err!=nil { t.Fatal(err) }
		if resp.StatusCode!=200 { body,_:=io.ReadAll(resp.Body);resp.Body.Close();t.Fatalf("status %d: %s",resp.StatusCode,body) }
		data:=make([]byte,188*10)
		if _,err=io.ReadFull(resp.Body,data);err!=nil {resp.Body.Close();t.Fatal(err)}
		for i:=0;i<len(data);i+=188 {if data[i]!=0x47 {resp.Body.Close();t.Fatal("packet alignment lost")}}
		if data[1]&0x1f!=0 || data[2]!=0 {resp.Body.Close();t.Fatal("join did not start with PAT")}
		return resp
	}
	a:=open();defer a.Body.Close()
	b:=open();defer b.Body.Close()
	if started.Load()!=1 {t.Fatalf("duplicate producers: %d",started.Load())}
	a.Body.Close();b.Body.Close()
	// The idle grace permits a probe-and-reopen without another producer.
	c:=open();c.Body.Close()
	if started.Load()!=1 {t.Fatal("warm reopen started another producer")}
	waitFastTune(t,func()bool{return stopped.Load()==1})
	waitFastTune(t,func()bool{p.mu.Lock();defer p.mu.Unlock();return len(p.sessions)==0})
}

func TestFastTuneLimitsAndPacketRing(t *testing.T) {
	p:=newFastTunePool(func(w http.ResponseWriter,r *http.Request,ch Channel){<-r.Context().Done()})
	p.packets=8
	defer p.close()
	ch:=sampleFeed().Channels[0]
	s,_,err:=p.acquire(ch);if err!=nil {t.Fatal(err)}
	defer p.release(s)
	for i:=1;i<fastTuneClients;i++ {
		other,reused,err:=p.acquire(ch);if err!=nil||!reused||other!=s {t.Fatalf("reuse %d: %v",i,err)}
		defer p.release(other)
	}
	if _,_,err:=p.acquire(ch);err==nil {t.Fatal("reader limit not enforced")}
	for i:=1;i<fastTuneChannels;i++ {
		next:=ch;next.ID=fmt.Sprintf("test-%d",i)
		other,_,err:=p.acquire(next);if err!=nil {t.Fatal(err)}
		defer p.release(other)
	}
	next:=ch;next.ID="overflow"
	if _,_,err:=p.acquire(next);err==nil {t.Fatal("channel limit not enforced")}
	w:=&fastTuneWriter{s:s,header:make(http.Header)}
	w.WriteHeader(200)
	packet:=make([]byte,188);packet[0]=0x47;packet[1]=0x1f;packet[2]=0xff;packet[3]=0x10
	for i:=0;i<12;i++ {packet[4]=byte(i);if _,err:=w.Write(packet);err!=nil {t.Fatal(err)}}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.ring)!=8*188 || s.oldest()!=4 || s.next!=12 || s.hasJoin {t.Fatal("incorrect bounded ring or unsafe join")}
}

func TestFastTuneAuthenticatedRealFFmpegPath(t *testing.T) {
	fixture:=makeTestTSPackets(t,true)
	previous:=mediaClient
	mediaClient=&http.Client{Transport:&mpegTSFixtureTransport{segment:fixture},Timeout:10*time.Second}
	defer func(){mediaClient=previous}()
	ln,err:=net.Listen("tcp","127.0.0.1:0");if err!=nil {t.Fatal(err)}
	cfg:=exampleCfg()
	cfg.Bind=ln.Addr().String();cfg.BaseURL="http://"+cfg.Bind;cfg.StreamMode="fast-tune"
	ch:=sampleFeed().Channels[0];ch.Stream="https://streams.example.org/live/test.m3u8"
	p:=newFastTunePool(func(w http.ResponseWriter,r *http.Request,ch Channel){serveJellyfinMPEGTS(w,r,cfg,ch)})
	defer p.close()
	h:=newJellyfinHandlerWithPool(cfg,func(string)ChannelData{return ChannelData{Channels:[]Channel{ch}}},p)
	srv:=&http.Server{Handler:h}
	go srv.Serve(ln)
	defer srv.Close()
	id:=base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
	target:=cfg.BaseURL+"/jellyfin/stream/"+id+".ts"
	if !bytes.Contains([]byte(m3uFeed([]Channel{ch},cfg)),[]byte(".ts?key=")) {t.Fatal("wrong tuner extension")}
	client:=&http.Client{Timeout:26*time.Second}
	unauth,err:=client.Get(target);if err!=nil {t.Fatal(err)}
	unauth.Body.Close();if unauth.StatusCode!=401 {t.Fatal("auth bypass")}
	ctx,cancel:=context.WithTimeout(context.Background(),26*time.Second)
	defer cancel()
	req,_:=http.NewRequestWithContext(ctx,"GET",target+"?key="+url.QueryEscape(cfg.Key),nil)
	resp,err:=client.Do(req);if err!=nil {t.Fatal(err)}
	defer resp.Body.Close()
	data,err:=io.ReadAll(io.LimitReader(resp.Body,8<<20));if err!=nil {t.Fatal(err)}
	if resp.StatusCode!=200 {t.Fatalf("status %d: %s",resp.StatusCode,data)}
	if len(data)<188*10 || len(data)%188!=0 {t.Fatalf("bad transport size %d",len(data))}
	var probe tsAudioProbe
	for i:=0;i<len(data);i+=188 {probe.add(data[i:i+188])}
	if !probe.audioReady() {t.Fatal("fast-tune output missing audio/video")}
	waitFastTune(t,func()bool{return activeMPEGTS.Load()==0})
}
