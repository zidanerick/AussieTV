package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Administrator-supplied local files only. No remote playlist discovery,
// credentials, cookies, header directives, DRM or geographic bypass.
const maxProviderChannels = 200
const maxProviderFiles = 8
const maxProviderGuideBytes = 8 << 20
const maxProviderPrograms = 20000

type providerSpec struct {
	ID string `json:"id"`
	Name string `json:"name"`
	Enabled bool `json:"enabled"`
	M3U string `json:"m3u"`
	XMLTV string `json:"xmltv,omitempty"`
	NumberStart int `json:"numberStart"`
}
type providerManifest struct {
	Version int `json:"version"`
	Providers []providerSpec `json:"providers"`
}
type providerSet struct {
	Channels []Channel
	IDs map[string]bool
}

var installedProviders = providerSet{}
var providerID = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
var m3uAttribute = regexp.MustCompile(`([A-Za-z0-9_-]+)="([^"]*)"`)

func readProviderFile(path string,limit int64)([]byte,error){
	f,err:=os.Open(path)
	if err!=nil{return nil,errors.New("provider file unavailable")}
	defer f.Close()
	info,err:=f.Stat()
	if err!=nil||!info.Mode().IsRegular()||info.Size()>limit{return nil,errors.New("provider file is not a bounded regular file")}
	b,err:=io.ReadAll(io.LimitReader(f,limit+1))
	if err!=nil||int64(len(b))>limit{return nil,errors.New("provider file exceeds size limit")}
	return b,nil
}
func providerFile(root,name string)(string,error){
	if name==""||filepath.Base(name)!=name||strings.ContainsAny(name,"/\\:")||name=="."||name==".." {return "",errors.New("provider files must be simple local filenames")}
	path:=filepath.Join(root,name)
	info,err:=os.Lstat(path)
	if err!=nil||!info.Mode().IsRegular(){return "",errors.New("provider file missing or not regular")}
	return path,nil
}
func providerText(s string,n int)bool {
	return strings.TrimSpace(s)!=""&&len(s)<=n&&!strings.ContainsAny(s,"\r\n\x00")&&singleLine(s)==s
}

func providerURL(raw string,playlist bool)bool {
	u,err:=validateRemote(raw)
	if err!=nil||u.Scheme!="https"||u.Port()!=""||u.Fragment!=""||len(raw)>2048||u.RawQuery!=""||u.ForceQuery||!strings.Contains(u.Hostname(),"."){return false}
	if playlist&&!strings.HasSuffix(strings.ToLower(u.Path),".m3u8"){return false}
	return true
}

type providerChannel struct {Channel Channel; GuideID string}

func parseProviderM3U(body []byte,spec providerSpec)([]providerChannel,error){
	sc:=bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte,4096),16384)
	if !sc.Scan()||!strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(sc.Text(),"\ufeff")),"#EXTM3U"){return nil,errors.New("missing M3U header")}
	var rows []providerChannel
	var pending *providerChannel
	seen:=map[string]bool{}
	for sc.Scan(){
		line:=strings.TrimSpace(sc.Text())
		if line=="" {continue}
		if strings.HasPrefix(line,"#EXTINF:"){
			if pending!=nil{return nil,errors.New("missing stream URL")}
			// Find the separator outside quoted attributes, allowing commas
			// in legitimate channel names without interpreting directives.
			quoted:=false;sep:=-1
			for i,r:=range line {if r=='"'{quoted=!quoted};if r==','&&!quoted{sep=i;break}}
			if sep<0{return nil,errors.New("invalid EXTINF")}
			attrs:=map[string]string{}
			for _,m:=range m3uAttribute.FindAllStringSubmatch(line[:sep],-1){
				key:=strings.ToLower(m[1]);if _,ok:=attrs[key];ok{return nil,errors.New("duplicate M3U attribute")};attrs[key]=m[2]
			}
			name:=strings.TrimSpace(line[sep+1:])
			if !providerText(name,120){return nil,errors.New("invalid channel name")}
			guide:=attrs["tvg-id"]
			if !providerText(guide,160){return nil,errors.New("channel requires a stable tvg-id")}
			if seen[guide]{return nil,errors.New("duplicate tvg-id")}
			seen[guide]=true
			if len(rows)>=maxProviderChannels{return nil,errors.New("provider channel limit exceeded")}
			number:=spec.NumberStart+len(rows)
			if number>9899{return nil,errors.New("provider channel number limit exceeded")}
			hash:=sha256.Sum256([]byte(guide))
			logo:=attrs["tvg-logo"];if !providerURL(logo,false){logo=""}
			pending=&providerChannel{GuideID:guide,Channel:Channel{ID:"ext-"+spec.ID+"-"+hex.EncodeToString(hash[:12]),Name:name,Number:strconv.Itoa(number),Network:spec.Name,Logo:logo,Programs:[]Program{}}}
			continue
		}
		if strings.HasPrefix(line,"#"){
			// Reject playback directives rather than silently dropping an
			// authentication, DRM, proxy or device requirement.
			if strings.HasPrefix(line,"#EXTVLCOPT")||strings.HasPrefix(line,"#KODIPROP")||strings.HasPrefix(line,"#EXTHTTP")||strings.HasPrefix(line,"#EXT-X-"){return nil,errors.New("unsupported M3U playback directive")}
			continue
		}
		if pending==nil||!providerURL(line,true){return nil,errors.New("only explicit public HTTPS HLS channel URLs without query credentials are accepted")}
		pending.Channel.Stream=line
		rows=append(rows,*pending);pending=nil
	}
	if sc.Err()!=nil||pending!=nil||len(rows)==0{return nil,errors.New("incomplete or empty provider playlist")}
	return rows,nil
}

func parseProviderTime(value string)(time.Time,error){
	return time.Parse("20060102150405 -0700",strings.TrimSpace(value))
}
func addProviderGuide(body []byte,rows []providerChannel)error{
	byID:=map[string]*Channel{}
	for i:=range rows{byID[rows[i].GuideID]=&rows[i].Channel}
	d:=xml.NewDecoder(bytes.NewReader(body))
	count:=0
	depth:=0
	for {
		tok,err:=d.Token()
		if err==io.EOF{break};if err!=nil{return errors.New("invalid XMLTV")}
		switch t:=tok.(type){
		case xml.Directive:
			// XMLTV commonly has a harmless DOCTYPE; Go never resolves
			// external entities. Reject all custom entity declarations.
			if strings.Contains(strings.ToUpper(string(t)),"ENTITY"){return errors.New("XMLTV entity declaration rejected")}
		case xml.StartElement:
			depth++;if depth>32{return errors.New("XMLTV nesting limit exceeded")}
			if t.Name.Local!="programme"||depth!=2{continue}
			count++;if count>maxProviderPrograms{return errors.New("XMLTV programme limit exceeded")}
			var p struct{Channel string `xml:"channel,attr"`;Start string `xml:"start,attr"`;Stop string `xml:"stop,attr"`;Title string `xml:"title"`}
			if d.DecodeElement(&p,&t)!=nil{return errors.New("invalid XMLTV programme")}
			depth--
			ch:=byID[p.Channel];if ch==nil||!providerText(p.Title,300){continue}
			a,e1:=parseProviderTime(p.Start);b,e2:=parseProviderTime(p.Stop)
			if e1!=nil||e2!=nil||!b.After(a)||b.Sub(a)>48*time.Hour||len(ch.Programs)>=2000{continue}
			ch.Programs=append(ch.Programs,Program{Start:a.Unix(),End:b.Unix(),Title:p.Title})
		case xml.EndElement:depth--
		}
	}
	for _,ch:=range byID{sort.SliceStable(ch.Programs,func(i,j int)bool{return ch.Programs[i].Start<ch.Programs[j].Start})}
	return nil
}

func loadProviderSet(path string)(providerSet,error){
	out:=providerSet{IDs:map[string]bool{}}
	if path==""{return out,nil}
	b,err:=readProviderFile(path,64<<10);if err!=nil{return out,err}
	var manifest providerManifest
	d:=json.NewDecoder(bytes.NewReader(b));d.DisallowUnknownFields()
	if d.Decode(&manifest)!=nil||manifest.Version!=1||len(manifest.Providers)>maxProviderFiles{return out,errors.New("invalid provider manifest version or count")}
	if d.Decode(new(any))!=io.EOF{return out,errors.New("trailing provider manifest content")}
	seen:=map[string]bool{}
	numbers:=map[string]bool{}
	for _,spec:=range activeCurated(){numbers[spec.number]=true}
	if extraAustralianEnabled {for _,ch:=range publicBroadcasterCandidates {numbers[ch.Number]=true}}
	for _,s:=range manifest.Providers{
		if !s.Enabled{continue}
		if !providerID.MatchString(s.ID)||seen[s.ID]||!providerText(s.Name,64)||s.Name=="All"||s.Name=="Favourites"||s.NumberStart<1000||s.NumberStart>9899{return providerSet{},errors.New("invalid provider ID, name or numbering")}
		seen[s.ID]=true
		p,err:=providerFile(filepath.Dir(path),s.M3U);if err!=nil{return providerSet{},err}
		data,err:=readProviderFile(p,1<<20);if err!=nil{return providerSet{},err}
		rows,err:=parseProviderM3U(data,s);if err!=nil{return providerSet{},fmt.Errorf("provider %s: %w",s.ID,err)}
		if s.XMLTV!="" {
			p,err:=providerFile(filepath.Dir(path),s.XMLTV);if err!=nil{return providerSet{},err}
			b,err:=readProviderFile(p,maxProviderGuideBytes);if err!=nil{return providerSet{},err}
			if err=addProviderGuide(b,rows);err!=nil{return providerSet{},err}
		}
		if len(out.Channels)+len(rows)>maxProviderChannels{return providerSet{},errors.New("total provider channel limit exceeded")}
		for _,row:=range rows{
			ch:=row.Channel
			if numbers[ch.Number]{return providerSet{},errors.New("duplicate provider channel number")}
			numbers[ch.Number]=true;out.IDs[ch.ID]=true
			out.Channels=append(out.Channels,ch)
		}
	}
	return out,nil
}

func appendProviderChannels(data ChannelData)ChannelData{
	public:=publicBroadcasterChannels()
	if len(installedProviders.Channels)==0 && len(public)==0{return data}
	channels:=make([]Channel,0,len(data.Channels)+len(installedProviders.Channels)+len(public))
	channels=append(channels,data.Channels...)
	for _,ch:=range installedProviders.Channels{channels=append(channels,ch)}
	channels=append(channels,public...)
	data.Channels=channels
	return data
}
func externalProviderChannel(id string)(Channel,bool){
	if ch,ok:=publicBroadcasterChannel(id);ok{return ch,true}
	if !installedProviders.IDs[id]{return Channel{},false}
	for _,ch:=range installedProviders.Channels{if ch.ID==id{return ch,true}}
	return Channel{},false
}
