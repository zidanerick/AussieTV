package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testBackdropURL = "https://static.tvmaze.com/uploads/images/original_untouched/123/background.jpg"

func TestArtworkDiskRoundTripAndValidation(t *testing.T) {
	now:=time.Now()
	path:=filepath.Join(t.TempDir(),"artwork.json")
	input:=map[string]artworkCacheEntry{
		"good":{Art:Artwork{Poster:testPosterURL,Backdrop:testBackdropURL,Page:"https://www.tvmaze.com/shows/123/example"},At:now},
		"stale":{Art:Artwork{Poster:testPosterURL},At:now.Add(-13*time.Hour)},
		"bad":{Art:Artwork{Poster:"http://127.0.0.1/image"},At:now},
		"miss":{Art:Artwork{},At:now},
		"bad backdrop":{Art:Artwork{Poster:testPosterURL,Backdrop:"https://example.com/fake.jpg"},At:now},
	}
	if err:=writeArtworkDisk(path,input);err!=nil {t.Fatal(err)}
	got:=readArtworkDisk(path)
	if len(got)!=3 || got["good"].Art.Backdrop!=testBackdropURL || got["good"].Art.Poster!=testPosterURL || got["bad backdrop"].Art.Backdrop!="" {t.Fatalf("bad restored cache: %+v",got)}
	if _,ok:=got["miss"];!ok {t.Fatal("negative cache lost")}
}

func TestArtworkBackdropLookupAndXMLTV(t *testing.T) {
	old:=artworkClient
	defer func(){artworkClient=old}()
	calls:=0
	artworkClient=&http.Client{Transport:artworkTransport(func(req *http.Request)(*http.Response,error){
		calls++
		var data string
		switch req.URL.Path {
		case "/search/shows":
			data=`[{"show":{"id":123,"name":"The Simpsons","url":"https://www.tvmaze.com/shows/123/example","image":{"medium":"`+testPosterURL+`"}}}]`
		case "/shows/123/images":
			data=`[{"type":"poster","main":true,"resolutions":{"original":{"url":"`+testPosterURL+`","width":400,"height":600}}},{"type":"background","main":true,"resolutions":{"original":{"url":"`+testBackdropURL+`","width":1280,"height":720}}}]`
		default:t.Fatalf("unexpected request %s",req.URL.Path)
		}
		return &http.Response{StatusCode:200,Body:io.NopCloser(strings.NewReader(data)),Header:make(http.Header),Request:req},nil
	})}
	art:=lookupArtwork("The Simpsons")
	if calls!=2 || art.Poster!=testPosterURL || art.Backdrop!=testBackdropURL {t.Fatalf("wrong art: %+v calls=%d",art,calls)}
	artMu.Lock()
	oldCache:=artCache
	artCache=map[string]artworkCacheEntry{canonicalTitle("The Simpsons"):{Art:art,At:time.Now()}}
	artMu.Unlock()
	defer func(){artMu.Lock();artCache=oldCache;artMu.Unlock()}()
	feed:=sampleFeed()
	feed.Channels[0].Programs=[]Program{{Start:1731924000,End:1731927600,Title:"The Simpsons"}}
	data,err:=xmltvFeed(feedChannels(feed),"Melbourne");if err!=nil {t.Fatal(err)}
	var doc xmltvDocument
	if err=xml.Unmarshal(data,&doc);err!=nil {t.Fatal(err)}
	p:=doc.Programmes[0]
	if p.Icon==nil || p.Icon.Src!=testPosterURL || len(p.Images)!=2 || p.Images[0].Type!="poster" || p.Images[1].Type!="backdrop" || p.Images[1].URL!=testBackdropURL {t.Fatalf("wrong artwork export: %+v",p)}
}

func TestArtworkGuidePrioritisesNowAndNext(t *testing.T) {
	now:=time.Now().Unix()
	data:=[]Channel{{Programs:[]Program{{Start:now-7200,End:now-3600,Title:"old"},{Start:now+3600,End:now+7200,Title:"next"}}},{Programs:[]Program{{Start:now-100,End:now+100,Title:"now"}}}}
	p:=prioritizedArtworkProgrammes(data)
	if len(p)!=3 || p[0].Title!="now" || p[1].Title!="next" {t.Fatal("wrong artwork priority")}
}

func TestJellyfinCuratedLogoEndpoint(t *testing.T) {
	ch:=Channel{ID:"abc-correct",Network:"ABC",Name:"ABC NEWS",Number:"24",Logo:"https://logos.example.net/genuine.png",Stream:"https://streams.example.org/live.m3u8"}
	cfg:=exampleCfg()
	feed:=ChannelData{Channels:[]Channel{ch}}
	h:=newJellyfinHandler(cfg,func(string)ChannelData{return feed})
	id:=base64.RawURLEncoding.EncodeToString([]byte(ch.ID))
	path:="/jellyfin/logo/"+id
	key:="?key="+url.QueryEscape(cfg.Key)
	if requestJellyfin(t,h,path,"GET").Code!=401 {t.Fatal("logo auth bypass")}
	// Reuse the same curated image cache as the browser.
	logoCache.Lock()
	old:=logoCache.items
	img:=[]byte{137,80,78,71,13,10,26,10}
	logoCache.items=map[string]cachedLogo{ch.Logo:{data:img,mime:"image/png",stored:time.Now()}}
	logoCache.Unlock()
	defer func(){logoCache.Lock();logoCache.items=old;logoCache.Unlock()}()
	r:=requestJellyfin(t,h,path+key,"GET")
	if r.Code!=200 || !bytes.Equal(r.Body.Bytes(),img) || r.Header().Get("Content-Type")!="image/png" {t.Fatal("logo proxy failure")}
	if !strings.Contains(m3uFeed([]Channel{ch},cfg),cfg.BaseURL+path+key) {t.Fatal("M3U logo URL missing")}
	x,err:=xmltvFeedWithConfig([]Channel{ch},cfg.Region,&cfg);if err!=nil {t.Fatal(err)}
	var doc xmltvDocument
	if xml.Unmarshal(x,&doc)!=nil || doc.Channels[0].Icon==nil || doc.Channels[0].Icon.Src!=cfg.BaseURL+path+key {t.Fatal("XMLTV logo mismatch")}
}
