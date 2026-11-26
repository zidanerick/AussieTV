package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const artworkMaxEntries = 2048
const artworkDiskMaxBytes = 4 << 20

var artworkDisk struct {
	sync.Mutex
	path string
}
var artworkPacer struct {
	sync.Mutex
	next time.Time
}

// Share one request budget across web and background enrichment, including
// optional show images. Never hold the artwork cache lock during network IO.
func pacedArtworkRequest(req *http.Request) (*http.Response,error) {
	artworkPacer.Lock()
	wait:=time.Until(artworkPacer.next)
	if wait>0 {time.Sleep(wait)}
	artworkPacer.next=time.Now().Add(700*time.Millisecond)
	artworkPacer.Unlock()
	return artworkClient.Do(req)
}

func lookupArtworkBackdrop(showID int) string {
	ctx,cancel:=context.WithTimeout(context.Background(),3*time.Second)
	defer cancel()
	req,err:=http.NewRequestWithContext(ctx,"GET",fmt.Sprintf("https://api.tvmaze.com/shows/%d/images",showID),nil)
	if err!=nil {return ""}
	req.Header.Set("User-Agent","AussieTV/0.1.0-beta.1 (TVmaze artwork attribution)")
	resp,err:=pacedArtworkRequest(req)
	if err!=nil {return ""}
	defer resp.Body.Close()
	if resp.StatusCode!=200 {return ""}
	var rows []struct{
		Type string `json:"type"`
		Main bool `json:"main"`
		Resolutions map[string]struct{
			URL string `json:"url"`
			Width int `json:"width"`
			Height int `json:"height"`
		} `json:"resolutions"`
	}
	if json.NewDecoder(io.LimitReader(resp.Body,512<<10)).Decode(&rows)!=nil {return ""}
	var selected string
	for _,row:=range rows {
		if row.Type!="background" {continue}
		for _,size:=range []string{"medium","original"} {
			img:=row.Resolutions[size]
			if img.Width<=img.Height || img.Height<=0 || !trustedPosterURL(img.URL) {continue}
			if selected=="" || row.Main {selected=img.URL}
			if row.Main {return selected}
			break
		}
	}
	return selected
}

func prioritizedArtworkProgrammes(channels []Channel) []Program {
	var list []Program
	now:=time.Now().Unix()
	for _,ch:=range channels {list=append(list,ch.Programs...)}
	// Prioritise all current shows, then the next guide entries, then old
	// entries. Historical fixtures remain supported and order is stable.
	rank:=func(p Program)int {if p.Start<=now && p.End>now{return 0};if p.Start>now{return 1};return 2}
	sort.SliceStable(list,func(i,j int)bool{
		a,b:=rank(list[i]),rank(list[j])
		if a!=b {return a<b}
		if a==2 {return list[i].Start>list[j].Start}
		return list[i].Start<list[j].Start
	})
	return list
}

func boundedArtworkEntries(entries map[string]artworkCacheEntry) map[string]artworkCacheEntry {
	type pair struct{key string;entry artworkCacheEntry}
	list:=make([]pair,0,len(entries))
	for key,entry:=range entries {
		if key=="" || len(key)>400 || !artworkCacheIsFresh(entry) || entry.At.After(time.Now().Add(time.Minute)) {continue}
		if entry.Art.Poster!="" && !trustedPosterURL(entry.Art.Poster) {continue}
		if !trustedPosterURL(entry.Art.Backdrop) {entry.Art.Backdrop=""}
		if !trustedTVmazeURL(entry.Art.Page) {entry.Art.Page=""}
		if entry.Art.Poster!="" {entry.Art.Credit="TVmaze"} else {entry.Art=Artwork{}}
		list=append(list,pair{key,entry})
	}
	sort.Slice(list,func(i,j int)bool{
		if list[i].entry.At.Equal(list[j].entry.At) {return list[i].key<list[j].key}
		return list[i].entry.At.After(list[j].entry.At)
	})
	out:=make(map[string]artworkCacheEntry)
	for _,row:=range list[:min(len(list),artworkMaxEntries)] {out[row.key]=row.entry}
	return out
}

func readArtworkDisk(path string) map[string]artworkCacheEntry {
	f,err:=os.Open(path)
	if err!=nil {return nil}
	defer f.Close()
	b,err:=io.ReadAll(io.LimitReader(f,artworkDiskMaxBytes+1))
	if err!=nil || len(b)>artworkDiskMaxBytes {return nil}
	var entries map[string]artworkCacheEntry
	if json.Unmarshal(b,&entries)!=nil {return nil}
	return boundedArtworkEntries(entries)
}

func writeArtworkDisk(path string,entries map[string]artworkCacheEntry) error {
	b,err:=json.Marshal(boundedArtworkEntries(entries))
	if err!=nil {return err}
	if len(b)>artworkDiskMaxBytes {return fmt.Errorf("artwork cache exceeds size limit")}
	if err:=os.MkdirAll(filepath.Dir(path),0700);err!=nil {return err}
	f,err:=os.CreateTemp(filepath.Dir(path),".artwork-*.json")
	if err!=nil {return err}
	defer os.Remove(f.Name())
	if _,err=f.Write(b);err!=nil {f.Close();return err}
	if err=f.Close();err!=nil {return err}
	return os.Rename(f.Name(),path)
}

func initArtworkDiskCache() {
	base,err:=os.UserCacheDir()
	if err!=nil {return}
	artworkDisk.Lock()
	defer artworkDisk.Unlock()
	artworkDisk.path=filepath.Join(base,"AussieTV","programme-artwork.json")
	entries:=readArtworkDisk(artworkDisk.path)
	artMu.Lock()
	defer artMu.Unlock()
	for k,v:=range entries {artCache[k]=v}
}

func storeArtworkCacheEntry(key string,result Artwork) {
	// Serialize snapshots and renames so an older writer cannot overwrite a
	// newer cache. No credentials, stream URLs or image binaries are stored.
	artworkDisk.Lock()
	defer artworkDisk.Unlock()
	artMu.Lock()
	artCache[key]=artworkCacheEntry{Art:result,At:time.Now()}
	if len(artCache)>artworkMaxEntries {artCache=boundedArtworkEntries(artCache)}
	var snapshot map[string]artworkCacheEntry
	if artworkDisk.path!="" {
		snapshot=make(map[string]artworkCacheEntry,len(artCache))
		for k,v:=range artCache {snapshot[k]=v}
	}
	artMu.Unlock()
	if snapshot!=nil {_=writeArtworkDisk(artworkDisk.path,snapshot)}
}

func cachedProgrammeImages(title string)(Artwork,bool){
	clean:=artworkTitle(title)
	if clean=="" {return Artwork{},false}
	artMu.Lock()
	entry,found:=artCache[canonicalTitle(clean)]
	artMu.Unlock()
	if !found || !artworkCacheIsFresh(entry) {return Artwork{},false}
	if !trustedPosterURL(entry.Art.Poster) {return Artwork{},true}
	if !trustedPosterURL(entry.Art.Backdrop) {entry.Art.Backdrop=""}
	return entry.Art,true
}

func warmProgrammeArtwork(ctx context.Context,region string) {
	ticker:=time.NewTicker(45*time.Second)
	defer ticker.Stop()
	for {
		if ctx.Err()!=nil {return}
		scheduleProgrammeArtwork(feedChannels(loadChannels(region)))
		select {case <-ctx.Done():return;case <-ticker.C:}
	}
}
