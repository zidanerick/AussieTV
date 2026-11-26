package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestArtworkSyncRepairsOnlyMissingExactProgrammes(t *testing.T) {
	now:=time.Now().UTC().Truncate(time.Second)
	feed:=exampleCfg()
	channelID:="11111111111111111111111111111111"
	missingID:="22222222222222222222222222222222"
	existingID:="33333333333333333333333333333333"
	wrongID:="44444444444444444444444444444444"
	unrelatedID:="55555555555555555555555555555555"
	ch:=Channel{ID:"test",Name:"9Gem",Number:"92",Programs:[]Program{
		{Title:"The Simpsons",Start:now.Add(-time.Hour).Unix(),End:now.Add(time.Hour).Unix()},
	}}
	artMu.Lock()
	old:=artCache
	artCache=map[string]artworkCacheEntry{canonicalTitle("The Simpsons"):{At:now,Art:Artwork{Poster:testPosterURL}}}
	artMu.Unlock()
	defer func(){artMu.Lock();artCache=old;artMu.Unlock()}()
	var img bytes.Buffer
	if err:=png.Encode(&img,image.NewRGBA(image.Rect(0,0,2,3)));err!=nil{t.Fatal(err)}
	uploads,checks,downloads:=0,0,0
	multi:=false
	api:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		if r.Header.Get("X-Emby-Token")!="separate-test-api-token" || strings.Contains(r.URL.String(),"separate-test-api-token") {t.Error("unsafe API authorization");w.WriteHeader(401);return}
		switch r.URL.Path {
		case "/System/Configuration/livetv":
			tuners:=[]map[string]string{{"Type":"m3u","Url":feed.BaseURL+"/jellyfin/channels.m3u?key="+feed.Key}}
			if multi {tuners=append(tuners,map[string]string{"Type":"hdhomerun","Url":"http://other"})}
			json.NewEncoder(w).Encode(map[string]any{"TunerHosts":tuners,"ListingProviders":[]map[string]string{{"Type":"xmltv","Path":feed.BaseURL+"/jellyfin/guide.xml?key="+feed.Key}}})
		case "/LiveTv/Channels":
			json.NewEncoder(w).Encode(syncItems{Items:[]syncItem{{ID:channelID,Name:ch.Name,Number:ch.Number,Type:"TvChannel"}}})
		case "/LiveTv/Programs":
			if r.URL.Query().Get("channelIds")!=channelID {t.Error("unscoped programme query")}
			item:=syncItem{ID:missingID,Name:"The Simpsons",Type:"LiveTvProgram",ChannelID:channelID,StartDate:now.Add(-time.Hour),EndDate:now.Add(time.Hour)}
			existing:=item;existing.ID=existingID;existing.ImageTags=map[string]string{"Primary":"already"}
			wrong:=item;wrong.ID=wrongID;wrong.StartDate=wrong.StartDate.Add(time.Minute)
			unrelated:=item;unrelated.ID=unrelatedID;unrelated.Type="Movie"
			json.NewEncoder(w).Encode(syncItems{Items:[]syncItem{item,existing,wrong,unrelated}})
		case "/Items/"+missingID+"/Images/Primary":
			if r.Method=="HEAD" {checks++;if uploads==0{w.WriteHeader(404)}else{w.WriteHeader(200)};return}
			if r.Method!="POST" || r.Header.Get("Content-Type")!="image/png" {t.Error("invalid image upload");w.WriteHeader(400);return}
			raw,_:=io.ReadAll(r.Body)
			decoded,err:=base64.StdEncoding.DecodeString(string(raw))
			if err!=nil || !bytes.Equal(decoded,img.Bytes()) {t.Error("upload is not base64 image bytes");w.WriteHeader(400);return}
			uploads++;w.WriteHeader(204)
		default:t.Errorf("unexpected request %s %s",r.Method,r.URL.Path);w.WriteHeader(404)
		}
	}))
	defer api.Close()
	poster:=&http.Client{Transport:artworkTransport(func(req *http.Request)(*http.Response,error){
		downloads++
		if req.URL.String()!=testPosterURL || req.Header.Get("X-Emby-Token")!="" {t.Error("untrusted poster request")}
		return &http.Response{StatusCode:200,Body:io.NopCloser(bytes.NewReader(img.Bytes())),Header:make(http.Header),Request:req},nil
	})}
	c:=&artworkSyncConfig{base:api.URL,token:"separate-test-api-token",feed:feed,client:api.Client(),posterClient:poster}
	report:=c.runOnce(context.Background(),[]Channel{ch},now)
	if report.Status!="ok" || report.Matched!=2 || report.Existing!=1 || report.Uploaded!=1 || report.Verified!=1 || uploads!=1 || downloads!=1 || checks!=2 {t.Fatalf("unexpected report %+v uploads=%d downloads=%d checks=%d",report,uploads,downloads,checks)}
	multi=true
	report=c.runOnce(context.Background(),[]Channel{ch},now)
	if report.Status=="ok" || uploads!=1 || downloads!=1 {t.Fatal("multi tuner scope not rejected")}
}

func TestArtworkSyncConfigAndRedirectIsolation(t *testing.T) {
	feed:=exampleCfg()
	t.Setenv("AUSSIETV_JELLYFIN_ARTWORK_SYNC","0")
	if c,err:=artworkSyncFromEnv(feed);c!=nil||err!=nil{t.Fatal("not opt in")}
	t.Setenv("AUSSIETV_JELLYFIN_ARTWORK_SYNC","1")
	t.Setenv("AUSSIETV_JELLYFIN_API_KEY_FILE","")
	t.Setenv("AUSSIETV_JELLYFIN_API_KEY",feed.Key)
	t.Setenv("AUSSIETV_JELLYFIN_SERVER_URL","http://localhost:8096")
	if _,err:=artworkSyncFromEnv(feed);err==nil{t.Fatal("feed key accepted as administrator key")}
	t.Setenv("AUSSIETV_JELLYFIN_API_KEY","separate-test-admin-token")
	reached:=false
	target:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){reached=true;w.WriteHeader(200)}))
	defer target.Close()
	api:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){http.Redirect(w,r,target.URL,302)}))
	defer api.Close()
	t.Setenv("AUSSIETV_JELLYFIN_SERVER_URL",api.URL)
	c,err:=artworkSyncFromEnv(feed);if err!=nil{t.Fatal(err)}
	var data any
	if c.readJSON(context.Background(),"/System/Configuration/livetv",nil,&data)==nil || reached{t.Fatal("redirect followed")}
}
