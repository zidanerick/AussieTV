package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// Optional direct image repair for the XMLTV importer in Jellyfin 12.
// The credential is a Jellyfin administrator API key, distinct from the
// read-only AussieTV feed key. It never enters a feed or browser endpoint.
type artworkSyncConfig struct {
	base, token string
	feed jellyfinConfig
	client *http.Client
	posterClient *http.Client
}

const syncBatchLimit = 40
const syncPosterMax = 4<<20

var syncIDPattern = regexp.MustCompile(`(?i)^(?:[0-9a-f]{32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})$`)

type syncReport struct {
	At string `json:"at,omitempty"`
	Status string `json:"status"`
	LastError string `json:"lastError,omitempty"`
	Matched int `json:"matched"`
	Existing int `json:"existing"`
	Uploaded int `json:"uploaded"`
	Verified int `json:"verified"`
	Failed int `json:"failed"`
}
var artworkSyncStatus = struct{
	sync.Mutex
	report syncReport
}{report:syncReport{Status:"disabled"}}

func getArtworkSyncReport() syncReport {
	artworkSyncStatus.Lock()
	defer artworkSyncStatus.Unlock()
	return artworkSyncStatus.report
}

func artworkSyncFromEnv(feed jellyfinConfig) (*artworkSyncConfig,error) {
	if os.Getenv("AUSSIETV_JELLYFIN_ARTWORK_SYNC")!="1" {return nil,nil}
	base:=strings.TrimRight(strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_SERVER_URL")),"/")
	u,err:=url.Parse(base)
	if err!=nil || u==nil || u.Hostname()=="" || (u.Scheme!="http" && u.Scheme!="https") || u.User!=nil || u.RawQuery!="" || u.ForceQuery || u.Fragment!="" || u.Opaque!="" {
		return nil,errors.New("artwork sync requires a valid Jellyfin server URL")
	}
	token:=strings.TrimSpace(os.Getenv("AUSSIETV_JELLYFIN_API_KEY"))
	if file:=os.Getenv("AUSSIETV_JELLYFIN_API_KEY_FILE");file!="" {
		if token!="" {return nil,errors.New("configure only one Jellyfin API key source")}
		f,err:=os.Open(file)
		if err!=nil {return nil,errors.New("cannot read Jellyfin API key file")}
		b,err:=io.ReadAll(io.LimitReader(f,4097));f.Close()
		if err!=nil || len(b)>4096 {return nil,errors.New("invalid Jellyfin API key file")}
		token=strings.TrimSpace(string(b))
	}
	if len(token)<16 || len(token)>512 || !regexp.MustCompile(`^[A-Za-z0-9._~-]+$`).MatchString(token) || token==feed.Key {return nil,errors.New("artwork sync requires a separate Jellyfin administrator API key")}
	return &artworkSyncConfig{base:base,token:token,feed:feed,client:&http.Client{
		Timeout:15*time.Second,Transport:&http.Transport{Proxy:nil,ResponseHeaderTimeout:10*time.Second},
		CheckRedirect:func(*http.Request,[]*http.Request)error{return http.ErrUseLastResponse},
	},posterClient:realLogoClient},nil
}

func (c *artworkSyncConfig) request(ctx context.Context,method,path string,query url.Values,body io.Reader,mime string)(*http.Response,error){
	endpoint:=c.base+path
	if len(query)>0 {endpoint+="?"+query.Encode()}
	req,err:=http.NewRequestWithContext(ctx,method,endpoint,body)
	if err!=nil {return nil,errors.New("invalid Jellyfin request")}
	req.Header.Set("Authorization", `MediaBrowser Token="`+c.token+`"`)
	if mime!="" {req.Header.Set("Content-Type",mime)}
	resp,err:=c.client.Do(req)
	if err!=nil {return nil,errors.New("Jellyfin connection failed")}
	return resp,nil
}
func (c *artworkSyncConfig) readJSON(ctx context.Context,path string,q url.Values,dest any) error {
	resp,err:=c.request(ctx,"GET",path,q,nil,"")
	if err!=nil {return err}
	defer resp.Body.Close()
	if resp.StatusCode!=200 {return fmt.Errorf("Jellyfin API HTTP %d",resp.StatusCode)}
	data,err:=io.ReadAll(io.LimitReader(resp.Body,(8<<20)+1))
	if err!=nil || len(data)>8<<20 {return errors.New("Jellyfin response too large or unreadable")}
	if json.Unmarshal(data,dest)!=nil {return errors.New("invalid Jellyfin JSON")}
	return nil
}

// Fail closed for multiple tuners or unmatched source URLs. Names/numbers
// alone do not prove which tuner a channel came from. No config writes.
func (c *artworkSyncConfig) verifyScope(ctx context.Context) error {
	var cfg struct{
		TunerHosts []struct{Type,URL string}
		ListingProviders []struct{Type,Path string}
	}
	if err:=c.readJSON(ctx,"/System/Configuration/livetv",nil,&cfg);err!=nil{return err}
	match:=func(raw,path string)bool {
		got,err:=url.Parse(raw)
		if err!=nil || got.User!=nil || got.Fragment!="" {return false}
		expected,err:=url.Parse(c.feed.BaseURL+path)
		if err!=nil {return false}
		return strings.EqualFold(got.Scheme,expected.Scheme)&&strings.EqualFold(got.Host,expected.Host)&&got.Path==expected.Path&&got.Query().Get("key")==c.feed.Key
	}
	if len(cfg.TunerHosts)!=1 || !strings.EqualFold(cfg.TunerHosts[0].Type,"m3u") || !match(cfg.TunerHosts[0].URL,"/jellyfin/channels.m3u") {
		return errors.New("artwork sync requires AussieTV as the sole matching M3U tuner")
	}
	if len(cfg.ListingProviders)!=1 || !strings.EqualFold(cfg.ListingProviders[0].Type,"xmltv") || !match(cfg.ListingProviders[0].Path,"/jellyfin/guide.xml") {
		return errors.New("artwork sync requires AussieTV as the sole matching XMLTV provider")
	}
	return nil
}

type syncItem struct {
	ID,Name,Type,Number,ChannelNumber,ChannelID string
	StartDate,EndDate time.Time
	ImageTags map[string]string
}
type syncItems struct {Items []syncItem;TotalRecordCount int}

func normalizedSyncID(s string) string {return strings.ToLower(strings.ReplaceAll(s,"-",""))}
func syncChannelKey(name,number string)string {return strings.ToLower(strings.TrimSpace(name))+"\x00"+strings.TrimSpace(number)}
func (c *artworkSyncConfig) channels(ctx context.Context,data []Channel)(map[string]Channel,error) {
	var list syncItems
	if err:=c.readJSON(ctx,"/LiveTv/Channels",url.Values{"limit":{"512"},"addCurrentProgram":{"false"},"enableImages":{"true"}},&list);err!=nil{return nil,err}
	if list.TotalRecordCount>len(list.Items) {return nil,errors.New("incomplete Jellyfin channel list")}
	candidates:=map[string][]Channel{}
	for _,ch:=range data {
		key:=syncChannelKey(ch.Name,ch.Number)
		candidates[key]=append(candidates[key],ch)
	}
	counts:=map[string]int{}
	for _,ch:=range list.Items {
		number:=ch.Number;if number=="" {number=ch.ChannelNumber}
		counts[syncChannelKey(ch.Name,number)]++
	}
	out:=map[string]Channel{}
	for _,ch:=range list.Items {
		if ch.Type!="TvChannel" && ch.Type!="LiveTvChannel" && ch.Type!="Channel" {continue}
		if !syncIDPattern.MatchString(ch.ID) {continue}
		number:=ch.Number;if number=="" {number=ch.ChannelNumber}
		if number=="" {continue}
		key:=syncChannelKey(ch.Name,number)
		if counts[key]==1 && len(candidates[key])==1 {out[normalizedSyncID(ch.ID)]=candidates[key][0]}
	}
	return out,nil
}

func matchingSyncArtwork(item syncItem,ch Channel) Artwork {
	if item.Type!="Program" && item.Type!="LiveTvProgram" {return Artwork{}}
	if !syncIDPattern.MatchString(item.ID) || item.StartDate.IsZero() || item.EndDate.IsZero() {return Artwork{}}
	var found *Program
	for i:=range ch.Programs {
		p:=&ch.Programs[i]
		if p.Start==item.StartDate.Unix() && p.End==item.EndDate.Unix() && strings.EqualFold(strings.TrimSpace(p.Title),strings.TrimSpace(item.Name)) {
			if found!=nil {return Artwork{}}
			found=p
		}
	}
	if found==nil {return Artwork{}}
	art,_:=cachedProgrammeImages(found.Title)
	return art
}

func (c *artworkSyncConfig) fetchPoster(ctx context.Context,raw string)([]byte,string,error) {
	if !trustedPosterURL(raw) {return nil,"",errors.New("untrusted artwork URL")}
	u,err:=url.Parse(raw)
	if err!=nil || u.Port()!="" || u.Fragment!="" || u.Opaque!="" {return nil,"",errors.New("invalid artwork URL")}
	req,err:=http.NewRequestWithContext(ctx,"GET",raw,nil)
	if err!=nil {return nil,"",errors.New("invalid artwork request")}
	resp,err:=c.posterClient.Do(req)
	if err!=nil {return nil,"",errors.New("artwork download failed")}
	defer resp.Body.Close()
	if resp.StatusCode!=200 || resp.ContentLength>syncPosterMax {return nil,"",errors.New("artwork unavailable")}
	b,err:=io.ReadAll(io.LimitReader(resp.Body,syncPosterMax+1))
	if err!=nil || len(b)>syncPosterMax {return nil,"",errors.New("artwork exceeds limit")}
	cfg,format,err:=image.DecodeConfig(bytes.NewReader(b))
	if err!=nil || cfg.Width<=0 || cfg.Height<=0 || cfg.Width>4096 || cfg.Height>4096 || int64(cfg.Width)*int64(cfg.Height)>16000000 {return nil,"",errors.New("invalid artwork dimensions")}
	mime:="image/jpeg"
	switch format {case "jpeg":case "png":mime="image/png";default:return nil,"",errors.New("unsupported artwork format")}
	return b,mime,nil
}

func (c *artworkSyncConfig) imageStatus(ctx context.Context,id string)(int,error) {
	resp,err:=c.request(ctx,"HEAD","/Items/"+id+"/Images/Primary",nil,nil,"")
	if err!=nil {return 0,err}
	resp.Body.Close()
	return resp.StatusCode,nil
}

func (c *artworkSyncConfig) repair(ctx context.Context,id string,art Artwork)(uploaded,verified bool,err error) {
	b,mime,err:=c.fetchPoster(ctx,art.Poster)
	if err!=nil {return false,false,err}
	// Check again immediately before writing, preserving existing artwork.
	status,err:=c.imageStatus(ctx,id)
	if err!=nil {return false,false,err}
	if status==200 {return false,false,nil}
	if status!=404 {return false,false,fmt.Errorf("Jellyfin image check HTTP %d",status)}
	resp,err:=c.request(ctx,"POST","/Items/"+id+"/Images/Primary",nil,strings.NewReader(base64.StdEncoding.EncodeToString(b)),mime)
	if err!=nil {return false,false,err}
	resp.Body.Close()
	if resp.StatusCode!=204 && resp.StatusCode!=200 {return false,false,fmt.Errorf("Jellyfin image upload HTTP %d",resp.StatusCode)}
	// Do not equate a successful write response with a verified image.
	status,err=c.imageStatus(ctx,id)
	if err!=nil || status!=200 {return true,false,errors.New("uploaded artwork not yet verified")}
	return true,true,nil
}

func (c *artworkSyncConfig) runOnce(ctx context.Context,data []Channel,now time.Time) syncReport {
	report:=syncReport{At:now.UTC().Format(time.RFC3339),Status:"ok"}
	if err:=c.verifyScope(ctx);err!=nil {report.Status=err.Error();return report}
	channels,err:=c.channels(ctx,data)
	if err!=nil {report.Status=err.Error();return report}
	if len(channels)==0 {report.Status="no unique matching channels";return report}
	ids:=make([]string,0,len(channels))
	for id:=range channels {ids=append(ids,id)}
	sort.Strings(ids)
	var list syncItems
	q:=url.Values{"channelIds":{strings.Join(ids,",")},"minEndDate":{now.UTC().Format(time.RFC3339)},"maxStartDate":{now.Add(24*time.Hour).UTC().Format(time.RFC3339)},"limit":{"2048"},"enableImages":{"true"},"enableUserData":{"false"},"sortBy":{"StartDate"}}
	if err:=c.readJSON(ctx,"/LiveTv/Programs",q,&list);err!=nil {report.Status=err.Error();return report}
	sort.SliceStable(list.Items,func(i,j int)bool{return list.Items[i].StartDate.Before(list.Items[j].StartDate)})
	attempts:=0
	seen:=map[string]bool{}
	for _,item:=range list.Items {
		if ctx.Err()!=nil || attempts>=syncBatchLimit {break}
		if !item.EndDate.After(now) || item.StartDate.After(now.Add(24*time.Hour)) {continue}
		ch,ok:=channels[normalizedSyncID(item.ChannelID)]
		if !ok {continue}
		art:=matchingSyncArtwork(item,ch)
		if art.Poster=="" {continue}
		id:=normalizedSyncID(item.ID)
		if seen[id] {continue};seen[id]=true
		report.Matched++
		if item.ImageTags["Primary"]!="" {report.Existing++;continue}
		attempts++
		uploaded,verified,err:=c.repair(ctx,id,art)
		if uploaded {report.Uploaded++}
		if verified {report.Verified++}
		if err!=nil {report.Failed++;report.LastError=err.Error()} else if !uploaded {report.Existing++}
	}
	if report.Failed>0 {report.Status="some images failed; see lastError"}
	return report
}

func (c *artworkSyncConfig) run(ctx context.Context) {
	ticker:=time.NewTicker(2*time.Minute)
	defer ticker.Stop()
	for {
		if ctx.Err()!=nil {return}
		cycle,cancel:=context.WithTimeout(ctx,90*time.Second)
		report:=c.runOnce(cycle,feedChannels(loadChannels(c.feed.Region)),time.Now())
		cancel()
		artworkSyncStatus.Lock()
		artworkSyncStatus.report=report
		artworkSyncStatus.Unlock()
		log.Printf("Jellyfin artwork sync status=%q matched=%d existing=%d uploaded=%d verified=%d failed=%d",report.Status,report.Matched,report.Existing,report.Uploaded,report.Verified,report.Failed)
		select {case <-ctx.Done():return;case <-ticker.C:}
	}
}
