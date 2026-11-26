package main

import (
	"net/url"
	"strings"
	"testing"
)

func TestHLSKeyPolicy(t *testing.T) {
	base,_:=url.Parse("https://example.org/live/master.m3u8")
	cfg:=exampleCfg()
	for _,tt:=range []struct{tag,reason string}{
		{`#EXT-X-KEY:METHOD=NONE`,""},
		{`#EXT-X-KEY:METHOD=AES-128,URI="key.bin"`,""},
		{`#EXT-X-KEY:METHOD=AES-128,URI="key.bin",KEYFORMAT="identity",KEYFORMATVERSIONS="1"`,""},
		{`#EXT-X-SESSION-KEY:METHOD=AES-128,URI="key.bin",KEYFORMAT="identity"`,""},
		{`#EXT-X-KEY:METHOD=SAMPLE-AES,URI="key.bin",KEYFORMAT="identity"`,"unsupported-hls-encryption-method"},
		{`#EXT-X-KEY:METHOD=SAMPLE-AES,URI="skd://license",KEYFORMAT="com.apple.streamingkeydelivery"`,"protected-hls-keyformat"},
		{`#EXT-X-KEY:METHOD=AES-128,URI="key.bin",KEYFORMAT="unknown"`,"protected-hls-keyformat"},
		{`#EXT-X-KEY:METHOD=AES-128,URI="key.bin",KEYFORMAT="identity",KEYFORMATVERSIONS="2"`,"unsupported-hls-key-version"},
		{`#EXT-X-KEY:METHOD=AES-128,URI="http://127.0.0.1/key"`,"invalid-or-blocked-hls-reference"},
		{`#EXT-X-KEY:METHOD=AES-128,URI="skd://license"`,"invalid-or-blocked-hls-reference"},
		{`#EXT-X-KEY:METHOD=AES-128,URI=key.bin`,"unsupported-hls-directive"},
		{`#EXT-X-KEY:METHOD=AES-128,URI="a",URI="b"`,"invalid-hls-key-tag"},
		{`#EXT-X-KEY:METHOD=NONE,URI="key.bin"`,"invalid-hls-key-tag"},
		{`#EXT-X-SESSION-KEY:METHOD=NONE`,"invalid-hls-key-tag"},
	}{
		t.Run(tt.tag,func(t *testing.T){
			out,err:=rewriteRelayPlaylist([]byte("#EXTM3U\n"+tt.tag+"\n#EXTINF:4,\nsegment.ts\n"),base,cfg,"fixture")
			if tt.reason!="" {
				if err==nil||hlsFailureReason(err)!=tt.reason{t.Fatalf("want %s, got %v",tt.reason,err)}
				return
			}
			if err!=nil{t.Fatal(err)}
			if strings.Contains(tt.tag,`URI="key.bin"`) {
				if !strings.Contains(string(out),"/resource.key?") || !strings.Contains(string(out),"METHOD=AES-128") {t.Fatal("key reference not safely preserved")}
				if strings.Contains(tt.tag,`KEYFORMAT="identity"`) && !strings.Contains(string(out),`KEYFORMAT="identity"`) {t.Fatal("key format was removed")}
			}
		})
	}
}
