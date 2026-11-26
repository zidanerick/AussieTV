package main

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type publicProbeTransport func(*http.Request)(*http.Response,error)
func (f publicProbeTransport) RoundTrip(r *http.Request)(*http.Response,error){return f(r)}

func TestPublicBroadcasterProbe(t *testing.T){
	ch:=publicBroadcasterCandidates[0]
	master:="#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1200000\nvariant.m3u8\n"
	live:="#EXTM3U\n#EXT-X-TARGETDURATION:6\n#EXTINF:6,\nsegment.ts\n"
	for _,tc:=range []struct{name,child,want string;code int}{
		{"valid",live,"manifest-ready",200},
		{"geo or login denied","","http-403",403},
		{"protected","#EXTM3U\n#EXT-X-KEY:METHOD=SAMPLE-AES,URI=\"key\",KEYFORMAT=\"com.apple.streamingkeydelivery\"\n#EXTINF:6,\nsegment.ts\n","unsupported-playlist",200},
		{"private","#EXTM3U\n#EXTINF:6,\nhttp://127.0.0.1/secret.ts\n","unsupported-playlist",200},
		{"vod",live+"#EXT-X-ENDLIST\n","no-live-media-playlist",200},
		{"html","<html>not video</html>","unsupported-playlist",200},
		{"empty","#EXTM3U\n","no-live-media-playlist",200},
	}{
		t.Run(tc.name,func(t *testing.T){
			calls:=0
			client:=&http.Client{Transport:publicProbeTransport(func(r *http.Request)(*http.Response,error){
				calls++
				if calls>2{t.Fatal("unexpected media or key fetch")}
				if r.Header.Get("Authorization")!=""||r.Header.Get("Cookie")!=""{t.Fatal("credentials sent")}
				body,code:=master,200
				if calls==2{body,code=tc.child,tc.code}
				return &http.Response{StatusCode:code,Body:io.NopCloser(strings.NewReader(body)),Request:r,Header:make(http.Header)},nil
			})}
			if got:=probePublicBroadcaster(context.Background(),client,ch);got!=tc.want{t.Fatalf("got %s want %s",got,tc.want)}
			if calls!=2{t.Fatalf("expected master and media playlist requests, got %d",calls)}
		})
	}
}

func TestPublicBroadcasterLineupGate(t *testing.T){
	old:=extraAustralianEnabled
	publicSources.Lock()
	oldReady,oldStatus:=publicSources.ready,publicSources.status
	publicSources.ready=map[string]bool{}
	publicSources.status=map[string]publicSourceStatus{}
	publicSources.Unlock()
	defer func(){extraAustralianEnabled=old;publicSources.Lock();publicSources.ready=oldReady;publicSources.status=oldStatus;publicSources.Unlock()}()
	ch:=publicBroadcasterCandidates[0]
	extraAustralianEnabled=true
	if _,ok:=externalProviderChannel(ch.ID);ok{t.Fatal("unverified source exposed")}
	publicSources.Lock()
	publicSources.ready[ch.ID]=true
	publicSources.Unlock()
	if got,ok:=externalProviderChannel(ch.ID);!ok||got.ID!=ch.ID||!requiresGuardedRelay(ch){t.Fatal("ready source not routed")}
	base:=ChannelData{}
	if got:=appendProviderChannels(base);len(got.Channels)!=len(installedProviders.Channels)+1{t.Fatal("ready channel not appended")}
	extraAustralianEnabled=false
	if _,ok:=externalProviderChannel(ch.ID);ok{t.Fatal("disabled source exposed")}
}
