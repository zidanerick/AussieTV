package main

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

// Simulate the Windows HTTP.sys/another-server port 80 conflict without
// requiring privileged ports or depending on which ports CI already uses.
func TestDiscoveryUDPStillStartsIfHTTP80Unavailable(t *testing.T) {
	cfg := exampleCfg()
	udp, listener, note := openHDHRListeners(cfg,
		func(string, string) (net.Listener, error) { return nil, errors.New("port 80 already in use") },
		func(string, *net.UDPAddr) (*net.UDPConn, error) {
			return net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
		})
	if listener != nil {
		listener.Close()
		t.Fatal("unexpected HTTP listener")
	}
	if udp == nil {
		t.Fatal("HTTP conflict incorrectly disabled UDP discovery")
	}
	done := make(chan struct{})
	go func() { runHDHRDiscovery(udp, cfg); close(done) }()
	t.Cleanup(func() {
		udp.Close()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("UDP goroutine did not exit")
		}
	})
	if !strings.Contains(note, "TCP 80 not listening") {
		t.Fatalf("missing TCP 80 warning: %s", note)
	}
	caller, err := net.DialUDP("udp4", nil, udp.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer caller.Close()
	caller.SetDeadline(time.Now().Add(time.Second))
	if _, err = caller.Write(hdhrProbeRequest(cfg)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1500)
	if _, err = caller.Read(buf); err != nil {
		t.Fatalf("UDP response lost under HTTP failure: %v", err)
	}
}

func TestDiscoveryRejectsInvalidLANAddressBeforeOpeningUDP(t *testing.T) {
	cfg := exampleCfg()
	cfg.BaseURL = "http://127.0.0.1:45782"
	opened := false
	udp, tcp, note := openHDHRListeners(cfg,
		func(string, string) (net.Listener, error) { opened = true; return nil, errors.New("unexpected") },
		func(string, *net.UDPAddr) (*net.UDPConn, error) { opened = true; return nil, errors.New("unexpected") })
	if opened || udp != nil || tcp != nil || !strings.Contains(note, "Auto-detection unavailable") {
		t.Fatalf("should fail closed: %v %v %s", udp, tcp, note)
	}
}
