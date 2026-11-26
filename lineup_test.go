package main

import (
	"strings"
	"testing"
)

func TestRoundTwoLineup(t *testing.T) {
	old:=extraAustralianEnabled
	defer func(){extraAustralianEnabled=old}()
	extraAustralianEnabled=true
	if len(extraAustralian)!=38 {t.Fatalf("unexpected shortlist count %d",len(extraAustralian))}
	entries:=map[string]Entry{}
	for i,s:=range extraAustralian {
		id:="fixture-"+s.number
		entries[id]=Entry{Name:s.name,Network:s.net,Master:"https://example.org/live/master.m3u8",Headers:map[string]string{"user-agent":"device-specific"},Programs:[][]any{{float64(1795676400),"Example show"},{float64(1795680000),"Next show"}}}
		_ = i
	}
	entries["blocked"]=Entry{Name:"7Bravo",Network:"Seven",Master:"https://example.org/live/master.m3u8"}
	entries["old-single-show"]=Entry{Name:"Seinfeld",Network:"Nine",Master:"https://example.org/live/master.m3u8"}
	entries["radio"]=Entry{Name:"SBS Radio 1",Network:"SBS",Master:"https://example.org/live/master.m3u8"}
	entries["bbc-duplicate"]=Entry{Name:"BBC News",Network:"Nine",Master:"https://example.org/live/master.m3u8"}
	got:=makeData(entries,"Melbourne")
	if len(got.Channels)!=38 {t.Fatalf("expected only 38 approved entries, got %d",len(got.Channels))}
	seen:=map[string]bool{}
	for _,ch:=range got.Channels {
		if seen[ch.Number] || ch.Category=="" || len(ch.Headers)!=0 || !requiresGuardedRelay(ch) || !logoChannelAllowed(ch) {t.Fatalf("invalid shortlist entry %+v",ch)}
		seen[ch.Number]=true
	}
	cfg:=exampleCfg()
	feed:=m3uFeed(feedChannels(got),cfg)
	if !strings.Contains(feed,`group-title="News & business"`) || strings.Contains(feed,"7Bravo") {t.Fatal("wrong M3U groups")}
	// Missing and unsafe sources are omitted, not shown as broken placeholders.
	delete(entries,"fixture-1000")
	entries["fixture-1001"]=Entry{Name:"YouTV",Network:"Ten",Master:"https://127.0.0.1/live.m3u8"}
	if n:=len(makeData(entries,"Melbourne").Channels);n!=36 {t.Fatalf("unsafe or absent candidate was exported: %d",n)}
	extraAustralianEnabled=false
	if len(makeData(entries,"Melbourne").Channels)!=0 {t.Fatal("opt-out did not remove extras")}
}
