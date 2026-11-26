package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Fixed candidates for freely offered broadcaster programming. Their CDN
// URLs were reported by public stream indexes; availability in Australia,
// programme identity and third-party player support still require live checks.
// Never synthesize credentials, spoof devices or retry via another country.
var publicBroadcasterCandidates = []Channel{
	{ID:"public-nhk-world", Name:"NHK World-Japan", Number:"1220", Network:"NHK", Category:"News / Business", Stream:"https://media-tyo.hls.nhkworld.jp/hls/w/live/master.m3u8", Official:"https://www3.nhk.or.jp/nhkworld/en/live/", Programs:[]Program{}},
	{ID:"public-cbs-news", Name:"CBS News 24/7", Number:"1221", Network:"CBS", Category:"News / Business", Stream:"https://cbsnews.akamaized.net/hls/live/2020607/cbsnlineup_8/master.m3u8", Official:"https://www.cbsnews.com/live/", Programs:[]Program{}},
	{ID:"public-abc-news-us", Name:"ABC News Live (US)", Number:"1222", Network:"ABC US", Category:"News / Business", Stream:"https://abcnews-streams.akamaized.net/hls/live/2023560/abcnewshudson1/master.m3u8", Official:"https://abcnews.com/Live", Programs:[]Program{}},
	{ID:"public-nbc-news-now", Name:"NBC News NOW", Number:"1223", Network:"NBC", Category:"News / Business", Stream:"https://nbcnews2.akamaized.net/hls/live/723426/nbc-news0529092111-ua/master.m3u8", Official:"https://www.nbcnews.com/now", Programs:[]Program{}},
}

type publicSourceStatus struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Number string `json:"number"`
	Status string `json:"status"`
	CheckedAt string `json:"checkedAt,omitempty"`
}

var publicSources = struct {
	sync.RWMutex
	ready map[string]bool
	status map[string]publicSourceStatus
}{ready:map[string]bool{}, status:map[string]publicSourceStatus{}}

func publicBroadcasterChannels() []Channel {
	if !extraAustralianEnabled {return nil}
	publicSources.RLock()
	defer publicSources.RUnlock()
	var out []Channel
	for _,ch:=range publicBroadcasterCandidates {
		if publicSources.ready[ch.ID] {out=append(out,ch)}
	}
	return out
}

func publicBroadcasterChannel(id string)(Channel,bool){
	for _,ch:=range publicBroadcasterChannels(){if ch.ID==id{return ch,true}}
	return Channel{},false
}

func publicBroadcasterReport() []publicSourceStatus {
	publicSources.RLock()
	defer publicSources.RUnlock()
	out:=make([]publicSourceStatus,0,len(publicBroadcasterCandidates))
	for _,ch:=range publicBroadcasterCandidates {
		if s,ok:=publicSources.status[ch.ID];ok {out=append(out,s);continue}
		state:="disabled"
		if extraAustralianEnabled {state="pending"}
		out=append(out,publicSourceStatus{ID:ch.ID,Name:ch.Name,Number:ch.Number,Status:state})
	}
	return out
}

// Manifest-only probe: bounded body, public HTTPS addresses, existing
// playlist policy. No media downloads, key requests or synthetic guide.
func fetchPublicManifest(ctx context.Context, client *http.Client, raw string, ch Channel)([]byte,*url.URL,string){
	u,err:=validateRemote(raw)
	if err!=nil||u.Scheme!="https" {return nil,nil,"blocked-address"}
	req,err:=http.NewRequestWithContext(ctx,http.MethodGet,raw,nil)
	if err!=nil{return nil,nil,"invalid-request"}
	setProviderRequestHeaders(req,ch)
	res,err:=client.Do(req)
	if err!=nil{return nil,nil,"network-error"}
	defer res.Body.Close()
	if res.StatusCode!=http.StatusOK{return nil,nil,fmt.Sprintf("http-%d",res.StatusCode)}
	if res.Request==nil||res.Request.URL==nil{return nil,nil,"invalid-response"}
	base:=res.Request.URL
	if _,err:=validateRemote(base.String());err!=nil||base.Scheme!="https"{return nil,nil,"blocked-redirect"}
	body,err:=io.ReadAll(io.LimitReader(res.Body,(512<<10)+1))
	if err!=nil||len(body)>512<<10{return nil,nil,"manifest-read-error"}
	if _,err:=rewriteRelayPlaylist(body,base,jellyfinConfig{},ch.ID);err!=nil{return nil,nil,"unsupported-playlist"}
	return body,base,""
}

func inspectPublicManifest(body []byte, base *url.URL)(string,bool){
	var variant, segment string
	var wantVariant,wantSegment,ended,master bool
	for _,raw:=range strings.Split(string(body),"\n"){
		line:=strings.TrimSpace(raw)
		if line=="" {continue}
		switch {
		case strings.HasPrefix(line,"#EXT-X-STREAM-INF:"):
			master=true;wantVariant=true;wantSegment=false
		case strings.HasPrefix(line,"#EXTINF:"):
			wantSegment=true
		case line=="#EXT-X-ENDLIST":
			ended=true
		case strings.HasPrefix(line,"#"):
			continue
		default:
			if wantVariant && variant=="" {
				variant,_=validatedPlaylistRef(base,line)
			}
			if wantSegment && segment=="" {
				segment,_=validatedPlaylistRef(base,line)
			}
			wantVariant=false;wantSegment=false
		}
	}
	if master {return variant,false}
	return "",segment!=""&&!ended
}

func probePublicBroadcaster(ctx context.Context, client *http.Client,ch Channel) string {
	if !providerURL(ch.Stream,true){return "invalid-source"}
	body,base,status:=fetchPublicManifest(ctx,client,ch.Stream,ch)
	if status!="" {return status}
	variant,live:=inspectPublicManifest(body,base)
	if live {return "manifest-ready"}
	if variant=="" {return "no-live-media-playlist"}
	body,base,status=fetchPublicManifest(ctx,client,variant,ch)
	if status!="" {return status}
	_,live=inspectPublicManifest(body,base)
	if !live {return "no-live-media-playlist"}
	return "manifest-ready"
}

// Check candidates concurrently without holding startup or Australian guide
// loading. Failed candidates can recover on a later refresh; successful ones
// remain stable for this process rather than flapping during a broadcast.
func refreshPublicBroadcasters(ctx context.Context, client *http.Client){
	var wg sync.WaitGroup
	for _,ch:=range publicBroadcasterCandidates {
		ch:=ch
		publicSources.RLock()
		ready:=publicSources.ready[ch.ID]
		publicSources.RUnlock()
		if ready{continue}
		wg.Add(1)
		go func(){
			defer wg.Done()
			attempt,cancel:=context.WithTimeout(ctx,12*time.Second)
			defer cancel()
			status:=probePublicBroadcaster(attempt,client,ch)
			publicSources.Lock()
			publicSources.ready[ch.ID]=status=="manifest-ready"
			publicSources.status[ch.ID]=publicSourceStatus{ID:ch.ID,Name:ch.Name,Number:ch.Number,Status:status,CheckedAt:time.Now().UTC().Format(time.RFC3339)}
			publicSources.Unlock()
			log.Printf("Public broadcaster %s: %s",ch.Name,status)
		}()
	}
	wg.Wait()
}

func runPublicBroadcasterChecks(ctx context.Context){
	if !extraAustralianEnabled{return}
	// Same public-address transport as playback, but do not downgrade
	// HTTPS or follow a long redirect chain during the readiness probe.
	client:=&http.Client{Transport:mediaClient.Transport,Timeout:12*time.Second,CheckRedirect:func(r *http.Request,via []*http.Request)error{
		if len(via)>=5||r.URL.Scheme!="https"{return http.ErrUseLastResponse}
		_,err:=validateRemote(r.URL.String())
		return err
	}}
	refreshPublicBroadcasters(ctx,client)
	ticker:=time.NewTicker(15*time.Minute)
	defer ticker.Stop()
	for {
		select{
		case <-ctx.Done():return
		case <-ticker.C:refreshPublicBroadcasters(ctx,client)
		}
	}
}
