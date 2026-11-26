package main
import (
	"encoding/json"
	"encoding/xml"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProviderSetImportAndUnifiedFeed(t *testing.T){
	dir:=t.TempDir()
	write:=func(name,content string){t.Helper();if err:=os.WriteFile(filepath.Join(dir,name),[]byte(content),0600);err!=nil{t.Fatal(err)}}
	write("custom.m3u",`#EXTM3U
#EXTINF:-1 tvg-id="my.channel" tvg-logo="https://example.org/logo.png",Science, TV
https://example.org/live/master.m3u8
`)
	write("custom.xml",`<tv><channel id="my.channel"><display-name>Science</display-name></channel><programme channel="my.channel" start="20261126080000 +1100" stop="20261126090000 +1100"><title>Science Show</title></programme><programme channel="other" start="20261126080000 +1100" stop="20261126090000 +1100"><title>Wrong Show</title></programme></tv>`)
	manifest:=providerManifest{Version:1,Providers:[]providerSpec{{ID:"science",Name:"Science",Enabled:true,M3U:"custom.m3u",XMLTV:"custom.xml",NumberStart:2000}}}
	data,_:=json.Marshal(manifest);write("providers.json",string(data))
	set,err:=loadProviderSet(filepath.Join(dir,"providers.json"));if err!=nil{t.Fatal(err)}
	if len(set.Channels)!=1 || len(set.Channels[0].Programs)!=1{t.Fatalf("incorrect import: %+v",set)}
	ch:=set.Channels[0]
	if ch.Name!="Science, TV"||ch.Number!="2000"||ch.Programs[0].Title!="Science Show"||!strings.HasPrefix(ch.ID,"ext-science-"){t.Fatalf("incorrect channel: %+v",ch)}
	old:=installedProviders;installedProviders=set;defer func(){installedProviders=old}()
	combined:=appendProviderChannels(sampleFeed())
	if len(combined.Channels)!=len(sampleFeed().Channels)+1{t.Fatal("merge failed")}
	if !logoChannelAllowed(ch){t.Fatal("external logo not recognised")}
	cfg:=exampleCfg()
	channels:=feedChannels(combined)
	if !strings.Contains(m3uFeed(channels,cfg),xmlTVID(cfg.Region,ch)){t.Fatal("provider missing in M3U")}
	b,err:=xmltvFeed(channels,cfg.Region);if err!=nil{t.Fatal(err)}
	var doc xmltvDocument
	if xml.Unmarshal(b,&doc)!=nil{t.Fatal("invalid XMLTV")}
	found:=false
	for _,p:=range doc.Programmes{if p.Channel==xmlTVID(cfg.Region,ch)&&p.Title.Value=="Science Show"{found=true}}
	if !found{t.Fatal("programme mapping missing")}
}

func TestProviderRejectsUnsafeInputs(t *testing.T){
	spec:=providerSpec{ID:"test",Name:"Test",NumberStart:2000}
	for _,s:=range []string{"http://example.org/x.m3u8","https://127.0.0.1/x.m3u8","https://100.64.0.1/x.m3u8","https://example.org:444/x.m3u8","https://user:pass@example.org/x.m3u8","https://example.org/x.m3u8?token=secret","file:///etc/passwd","https://example.org/movie.mp4","https://localhost./x.m3u8"}{
		if _,err:=parseProviderM3U([]byte("#EXTM3U\n#EXTINF:-1 tvg-id=\"a\",Example\n"+s+"\n"),spec);err==nil{t.Fatalf("accepted unsafe source %s",s)}
	}
	if _,err:=parseProviderM3U([]byte("#EXTM3U\n#EXTINF:-1 tvg-id=\"a\",Example\n#KODIPROP:inputstream.adaptive.license_key=x\nhttps://example.org/x.m3u8\n"),spec);err==nil{t.Fatal("accepted playback directive")}
	if _,err:=providerFile(t.TempDir(),"../secret");err==nil{t.Fatal("accepted traversal")}
}

func TestNetworkGuardSpecialAddresses(t *testing.T){
	for _,s:=range []string{"100.64.0.1","198.18.0.1","192.0.2.1","169.254.169.254","127.0.0.1","::ffff:10.0.0.1","64:ff9b::a00:1","2002:a00:1::"}{
		if publicMediaIP(net.ParseIP(s)){t.Fatalf("allowed %s",s)}
	}
	for _,s:=range []string{"8.8.8.8","2606:4700:4700::1111"}{if !publicMediaIP(net.ParseIP(s)){t.Fatalf("blocked public %s",s)}}
}

func TestExtraAustralianLineupOptIn(t *testing.T){
	old:=extraAustralianEnabled;defer func(){extraAustralianEnabled=old}()
	entries:=map[string]Entry{"mjh-c31":{Name:"Channel 31",Network:"Community",Master:"https://example.org/c31.m3u8"},"mjh-rush":{Name:"9Rush",Network:"Nine",Master:"https://example.org/rush.m3u8"}}
	extraAustralianEnabled=false
	if len(makeData(entries,"Melbourne").Channels)!=0{t.Fatal("extras enabled by default")}
	extraAustralianEnabled=true
	if got:=makeData(entries,"Melbourne");len(got.Channels)!=2{t.Fatalf("missing opt-in extras: %+v",got)}
}
