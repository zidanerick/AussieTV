package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestFastTuneManifestSelection(t *testing.T) {
	master:=`#EXTM3U
#EXT-X-MEDIA:TYPE=AUDIO,GROUP-ID="eng",NAME="English, main",DEFAULT=YES,URI="audio.m3u8"
#EXT-X-STREAM-INF:BANDWIDTH=1500000,CODECS="avc1.4d401f,mp4a.40.2",AUDIO="eng"
low.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=5000000,CODECS="avc1.640028,mp4a.40.2",AUDIO="eng"
high.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=9000000,CODECS="avc1.640032,mp4a.40.2",AUDIO="eng"
ultra.m3u8
#EXT-X-STREAM-INF:BANDWIDTH=3000000,CODECS="hvc1.1.6.L120,mp4a.40.2",AUDIO="eng"
hevc.m3u8
`
	got:=string(selectFastTuneVariant([]byte(master)))
	for _,required:=range []string{"#EXTM3U","audio.m3u8","English, main","high.m3u8","AUDIO=\"eng\""} {
		if !strings.Contains(got,required) {t.Fatalf("missing %q",required)}
	}
	for _,removed:=range []string{"low.m3u8","ultra.m3u8","hevc.m3u8"} {
		if strings.Contains(got,removed) {t.Fatalf("unexpected %q",removed)}
	}
	media:=[]byte("#EXTM3U\n#EXTINF:4,\nsegment.ts\n")
	if !bytes.Equal(selectFastTuneVariant(media),media) {t.Fatal("media playlist modified")}
	unknown:=[]byte(strings.ReplaceAll(strings.ReplaceAll(master,"avc1.","unknown."),"hvc1.","unknown."))
	if !bytes.Equal(selectFastTuneVariant(unknown),unknown) {t.Fatal("unknown codecs pruned")}
	malformed:=[]byte(strings.Replace(master,"low.m3u8","#comment\nlow.m3u8",1))
	if !bytes.Equal(selectFastTuneVariant(malformed),malformed) {t.Fatal("malformed master pruned")}
}
