package main

import (
	"strconv"
	"strings"
)

// Limit known H.264/AAC master playlists to one variant for the experimental
// shared TS producer. Never invent codecs, remove the selected audio group,
// or modify a media playlist. Unknown/malformed layouts pass through intact.
func fastTuneAttributes(raw string) (map[string]string,bool) {
	values:=make(map[string]string)
	for raw!="" {
		eq:=strings.IndexByte(raw,'=')
		if eq<=0 {return nil,false}
		key:=strings.TrimSpace(raw[:eq])
		raw=raw[eq+1:]
		if key=="" || raw=="" {return nil,false}
		var val string
		if raw[0]=='"' {
			end:=strings.IndexByte(raw[1:],'"')
			if end<0 {return nil,false}
			val=raw[1:end+1]
			raw=raw[end+2:]
		} else {
			i:=strings.IndexByte(raw,',')
			if i<0 {val=raw;raw=""} else {val=raw[:i];raw=raw[i:]}
		}
		if _,exists:=values[key];exists {return nil,false}
		values[key]=val
		if raw!="" {
			if raw[0]!=',' {return nil,false}
			raw=raw[1:]
		}
	}
	return values,true
}

func selectFastTuneVariant(body []byte) []byte {
	lines:=strings.Split(string(body),"\n")
	type variant struct{ tag,uri int; bandwidth int64; compatible bool }
	var variants []variant
	for i,line:=range lines {
		line=strings.TrimSpace(line)
		if !strings.HasPrefix(line,"#EXT-X-STREAM-INF:") {continue}
		// Only prune straightforward adjacent URI pairs.
		if i+1>=len(lines) || strings.TrimSpace(lines[i+1])=="" || strings.HasPrefix(strings.TrimSpace(lines[i+1]),"#") {return body}
		attrs,ok:=fastTuneAttributes(strings.TrimPrefix(line,"#EXT-X-STREAM-INF:"))
		if !ok {return body}
		bw,err:=strconv.ParseInt(attrs["BANDWIDTH"],10,64)
		if err!=nil || bw<=0 {return body}
		h264,aac,unknown:=false,false,false
		for _,codec:=range strings.Split(attrs["CODECS"],",") {
			codec=strings.ToLower(strings.TrimSpace(codec))
			switch {
			case strings.HasPrefix(codec,"avc1."),strings.HasPrefix(codec,"avc3."):h264=true
			case strings.HasPrefix(codec,"mp4a.40."):aac=true
			default:unknown=true
			}
		}
		variants=append(variants,variant{i,i+1,bw,h264&&aac&&!unknown})
	}
	if len(variants)<2 {return body}
	// Prefer the highest declared bandwidth at or below 6 Mbit/s, otherwise
	// the lowest compatible variant. All original rendition groups remain.
	chosen:=-1
	for i,v:=range variants {
		if !v.compatible {continue}
		if chosen<0 {chosen=i;continue}
		current:=variants[chosen]
		if (v.bandwidth<=6000000 && (current.bandwidth>6000000 || v.bandwidth>current.bandwidth)) ||
		   (v.bandwidth>6000000 && current.bandwidth>6000000 && v.bandwidth<current.bandwidth) {chosen=i}
	}
	if chosen<0 {return body}
	drop:=make(map[int]bool)
	for i,v:=range variants {if i!=chosen {drop[v.tag]=true;drop[v.uri]=true}}
	out:=make([]string,0,len(lines))
	for i,line:=range lines {if !drop[i] {out=append(out,line)}}
	return []byte(strings.Join(out,"\n"))
}
