package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestExpandedHLSMetadataCompatibility(t *testing.T) {
	base,_:=url.Parse("https://example.org/live/master.m3u8")
	cfg:=exampleCfg()
	manifest:= "#EXTM3U\n#EXT-X-SESSION-DATA:DATA-ID=\"com.example.title\",VALUE=\"Programme\"\n#EXT-X-STREAM-INF:BANDWIDTH=1000000\nvariant.m3u8\n"
	body,err:=rewriteRelayPlaylist([]byte(manifest),base,cfg,"test")
	if err!=nil || !strings.Contains(string(body),"VALUE=\"Programme\"") || !strings.Contains(string(body),"/jellyfin/media/") {t.Fatalf("inline metadata rejected or not rewritten: %v",err)}
	for _,tag:=range []string{
		"#EXT-X-SESSION-DATA:DATA-ID=\"com.example.title\",URI=\"metadata.json\"",
		"#EXT-X-SESSION-DATA:DATA-ID=\"a\",VALUE=\"b\",URI=\"metadata.json\"",
		"#EXT-X-CONTENT-STEERING:SERVER-URI=\"steering.json\"",
		"#EXT-X-DEFINE:NAME=\"source\",VALUE=\"example.org\"",
	} {
		if _,err:=rewriteRelayPlaylist([]byte("#EXTM3U\n"+tag+"\n"),base,cfg,"test");hlsFailureReason(err)!="unsupported-hls-directive"{t.Fatalf("indirect request allowed: %v",err)}
	}
	if _,err:=rewriteRelayPlaylist([]byte("#EXTM3U\n#EXT-X-KEY:METHOD=SAMPLE-AES,URI=\"key\"\n"),base,cfg,"test");hlsFailureReason(err)!="protected-hls"{t.Fatalf("protected manifest accepted: %v",err)}
}

type compatibilityTransport func(*http.Request)(*http.Response,error)
func (f compatibilityTransport) RoundTrip(r *http.Request)(*http.Response,error){return f(r)}

func TestExpandedHLSKeepsVariantsAndRecordsProviderFailures(t *testing.T) {
	oldClient,oldEnabled:=mediaClient,extraAustralianEnabled
	t.Cleanup(func(){mediaClient=oldClient;extraAustralianEnabled=oldEnabled})
	extraAustralianEnabled=true
	cfg:=exampleCfg();cfg.StreamMode="fast-tune"
	ch:=Channel{ID:"compatibility-test",Name:"A Country Practice",Network:"Seven",Number:"1600",Stream:"https://example.org/master.m3u8"}
	if !requiresGuardedRelay(ch){t.Fatal("test expansion not recognised")}
	manifest:="#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000000,CODECS=\"avc1.64001f,mp4a.40.2\"\nlow.m3u8\n#EXT-X-STREAM-INF:BANDWIDTH=4000000,CODECS=\"avc1.64001f,mp4a.40.2\"\nhigh.m3u8\n"
	status:=200
	mediaClient=&http.Client{Transport:compatibilityTransport(func(r *http.Request)(*http.Response,error){
		return &http.Response{StatusCode:status,Request:r,Header:http.Header{"Content-Type":[]string{"application/vnd.apple.mpegurl"}},Body:io.NopCloser(strings.NewReader(manifest))},nil
	})}
	w:=httptest.NewRecorder()
	relayPlaylistOrSegment(w,httptest.NewRequest("GET","http://localhost/test",nil),cfg,ch,ch.Stream)
	if w.Code!=200 || strings.Count(w.Body.String(),"#EXT-X-STREAM-INF:")!=2 {t.Fatalf("expanded variant removed, status=%d",w.Code)}
	status=403
	w=httptest.NewRecorder()
	relayPlaylistOrSegment(w,httptest.NewRequest("GET","http://localhost/test",nil),cfg,ch,ch.Stream)
	events:=recentTuneFailures()
	if len(events)==0 {t.Fatal("failure missing")}
	last:=events[len(events)-1]
	if w.Code!=502 || last.Channel!=ch.ID || last.UpstreamStatus!=403 || last.Reason!="provider-http-error"{t.Fatalf("unexpected failure %+v",last)}
}
