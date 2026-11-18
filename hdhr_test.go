package main

import (
	"encoding/binary"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHDHRDiscoveryPackets(t *testing.T) {
	cfg := exampleCfg()
	request := hdhrProbeRequest(cfg)
	if !hdhrParseRequest(request, hdhrDeviceID(cfg)) {
		t.Fatal("valid broadcast discovery rejected")
	}
	request[4] ^= 1
	if hdhrParseRequest(request, hdhrDeviceID(cfg)) {
		t.Fatal("corrupt request accepted")
	}
	response := hdhrResponse(cfg)
	if len(response) < 30 || response[1] != hdhrTypeReply {
		t.Fatal("invalid response")
	}
}
func TestHDHRLineupAndPrivateAccess(t *testing.T) {
	cfg := exampleCfg()
	h := hdhrHTTP(cfg, func(string) ChannelData { return sampleFeed() })
	for _, p := range []string{"/discover.json", "/lineup.json", "/lineup_status.json"} {
		r := httptest.NewRequest("GET", p, nil)
		r.RemoteAddr = "192.168.1.9:55001"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s returned %d", p, w.Code)
		}
	}
	r := httptest.NewRequest("GET", "/lineup.json", nil)
	r.RemoteAddr = "192.168.1.9:55001"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var channels []map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &channels); err != nil || len(channels) != 1 {
		t.Fatalf("bad lineup: %v %s", err, w.Body.String())
	}
	if !strings.Contains(channels[0]["URL"], "/auto/v24") {
		t.Fatalf("bad stream URL: %v", channels)
	}
	r = httptest.NewRequest("GET", "/auto/v24", nil)
	r.RemoteAddr = "192.168.1.9:55001"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("stream redirect %d", w.Code)
	}
	r = httptest.NewRequest("GET", "/lineup.json", nil)
	r.RemoteAddr = "8.8.8.8:52222"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatal("public requester allowed")
	}
	r = httptest.NewRequest("POST", "/lineup.json", nil)
	r.RemoteAddr = "192.168.1.9:55001"
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatal("write request allowed")
	}
}

func TestHDHRDeviceIDChecksumAndPrivateHTTPBind(t *testing.T) {
	cfg := exampleCfg()
	id := hdhrDeviceID(cfg)
	lookup := [16]byte{10, 5, 15, 6, 7, 12, 1, 11, 9, 2, 8, 13, 4, 3, 14, 0}
	var sum byte
	for shift := 28; shift >= 0; shift -= 4 {
		nibble := byte((id >> uint(shift)) & 15)
		if shift%8 == 4 {
			sum ^= lookup[nibble]
		} else {
			sum ^= nibble
		}
	}
	if sum != 0 {
		t.Fatalf("HDHR ID %08x has bad checksum %d", id, sum)
	}
	bind, err := hdhrHTTPBind(cfg)
	if err != nil || bind != "192.168.1.50:80" {
		t.Fatalf("LAN HTTP discovery address: %s %v", bind, err)
	}
	for _, x := range []string{"http://127.0.0.1:45782", "http://8.8.8.8:45782", "https://192.168.1.50:45782", "http://192.168.1.50:9000"} {
		cfg.BaseURL = x
		if _, err := hdhrHTTPBind(cfg); err == nil {
			t.Fatalf("accepted invalid discovery host %s", x)
		}
	}
	packet := hdhrResponse(exampleCfg())
	if binary.BigEndian.Uint16(packet[:2]) != hdhrTypeReply {
		t.Fatal("reply type")
	}
}

func TestHDHRUDPListenerActualPacketExchange(t *testing.T) {
	cfg := exampleCfg()
	listener, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { runHDHRDiscovery(listener, cfg); close(done) }()
	t.Cleanup(func() {
		listener.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("listener did not exit")
		}
	})
	remote := listener.LocalAddr().(*net.UDPAddr)
	client, err := net.DialUDP("udp4", nil, remote)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if err := client.SetDeadline(time.Now().Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Write(hdhrProbeRequest(cfg)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1500)
	n, err := client.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if n < 12 || binary.BigEndian.Uint16(buf[:2]) != hdhrTypeReply {
		t.Fatalf("bad UDP discovery response: %x", buf[:n])
	}
}
