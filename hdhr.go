package main

// Optional, LAN-only HDHomeRun HTTP and UDP discovery interoperability.
// This does not provide RF tuning, transcoding or an HDHomeRun hardware DVR.
// The existing authenticated M3U/XMLTV feed is the preferred integration.
import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const hdhrPort = 65001
const hdhrTypeRequest = 0x0002
const hdhrTypeReply = 0x0003
const hdhrTunerType uint32 = 1
const hdhrWildcard uint32 = 0xffffffff

var discoveryProbes atomic.Uint64
var discoveryReplies atomic.Uint64
var discoveryHTTPRequests atomic.Uint64

// Configured discovery is not the same as actively bound discovery sockets.
// Report listener state independently of the counters, which only increase
// when real client traffic reaches the listener.
var discoveryUDPListening atomic.Bool
var discoveryHTTPListening atomic.Bool

func hdhrDeviceID(cfg jellyfinConfig) uint32 {
	// Local virtual device identity, stable for a given advertised URL.
	// The model is virtual, and does not imply actual SiliconDust hardware.
	var h uint32 = 2166136261
	for _, b := range []byte(cfg.BaseURL) {
		h ^= uint32(b)
		h *= 16777619
	}
	// Last nibble is a checksum. Unchecked arbitrary IDs can be rejected by
	// HDHomeRun discovery clients even if the UDP packet is otherwise valid.
	const base = uint32(0x10000000)
	candidate := (h & 0x0ffffff0) | base
	lookup := [16]uint32{10, 5, 15, 6, 7, 12, 1, 11, 9, 2, 8, 13, 4, 3, 14, 0}
	checksum := uint32(0)
	for shift := 28; shift >= 4; shift -= 4 {
		nibble := (candidate >> uint(shift)) & 0xf
		if shift%8 == 4 {
			checksum ^= lookup[nibble]
		} else {
			checksum ^= nibble
		}
	}
	return candidate | checksum
}

// Certain Jellyfin HDHomeRun discovery paths query /discover.json on HTTP
// port 80 of the UDP reply address before they use the advertised BaseURL.
// Bind only the explicitly configured private IPv4 address, never 0.0.0.0.
func hdhrHTTPBind(cfg jellyfinConfig) (string, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil || u.Scheme != "http" || u.Port() != "45782" {
		return "", fmt.Errorf("HDHomeRun discovery requires a LAN http://IP:45782 address")
	}
	ip := net.ParseIP(u.Hostname())
	if ip == nil || ip.To4() == nil || !ip.IsPrivate() {
		return "", fmt.Errorf("HDHomeRun discovery requires a private IPv4 LAN address")
	}
	return net.JoinHostPort(ip.String(), "80"), nil
}

func privateClient(r *http.Request) bool {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(host)
	return ip != nil && (ip.IsPrivate() || ip.IsLoopback())
}
func hdhrLineup(channels []Channel, cfg jellyfinConfig) []map[string]string {
	var rows []map[string]string
	seen := map[string]bool{}
	for _, ch := range channels {
		// HDHR identifies entries by virtual GuideNumber. Skip conflicts rather
		// than accidentally tuning one channel from another row.
		n := strings.TrimSpace(ch.Number)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		rows = append(rows, map[string]string{
			"GuideNumber": n, "GuideName": ch.Name,
			"URL": cfg.BaseURL + "/auto/v" + url.PathEscape(n),
		})
	}
	return rows
}
func hdhrHTTP(cfg jellyfinConfig, load func(string) ChannelData) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /discover.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"FriendlyName": "AussieTV Virtual Tuner", "Manufacturer": "AussieTV", "ModelNumber": "AussieTV-IPTV",
			"FirmwareName": "aussietv", "FirmwareVersion": version,
			"DeviceID": strings.ToUpper(strconv.FormatUint(uint64(hdhrDeviceID(cfg)), 16)),
			"BaseURL":  cfg.BaseURL, "LineupURL": cfg.BaseURL + "/lineup.json", "TunerCount": 1,
		})
	})
	mux.HandleFunc("GET /lineup_status.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ScanInProgress": 0, "ScanPossible": 0, "Source": "Antenna", "SourceList": []string{"Antenna"}})
	})
	mux.HandleFunc("GET /lineup.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(hdhrLineup(feedChannels(load(cfg.Region)), cfg))
	})
	mux.HandleFunc("GET /auto/{channel}", func(w http.ResponseWriter, r *http.Request) {
		target := strings.TrimPrefix(r.PathValue("channel"), "v")
		for _, ch := range feedChannels(load(cfg.Region)) {
			if ch.Number == target {
				http.Redirect(w, r, ch.Stream, http.StatusTemporaryRedirect)
				return
			}
		}
		http.NotFound(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Discovery-enabled mode deliberately allows read-only requests by local
		// network devices. Never permit these endpoints to accept public clients.
		if !privateClient(r) {
			http.Error(w, "LAN only", http.StatusForbidden)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		discoveryHTTPRequests.Add(1)
		mux.ServeHTTP(w, r)
	})
}
func hdhrTLV(tag byte, payload []byte) []byte {
	b := []byte{tag}
	n := len(payload)
	if n < 128 {
		b = append(b, byte(n))
	} else {
		b = append(b, byte(n&127)|128, byte(n>>7))
	}
	return append(b, payload...)
}
func hdhrPacket(typ uint16, payload []byte) []byte {
	packet := make([]byte, 4, 8+len(payload))
	binary.BigEndian.PutUint16(packet[:2], typ)
	binary.BigEndian.PutUint16(packet[2:4], uint16(len(payload)))
	packet = append(packet, payload...)
	var crc [4]byte
	binary.LittleEndian.PutUint32(crc[:], crc32.ChecksumIEEE(packet))
	return append(packet, crc[:]...)
}
func hdhrParseRequest(packet []byte, deviceID uint32) bool {
	if len(packet) < 8 || binary.BigEndian.Uint16(packet[:2]) != hdhrTypeRequest {
		return false
	}
	n := int(binary.BigEndian.Uint16(packet[2:4]))
	if n+8 != len(packet) || crc32.ChecksumIEEE(packet[:4+n]) != binary.LittleEndian.Uint32(packet[4+n:]) {
		return false
	}
	foundType, foundID := false, false
	for b := packet[4 : 4+n]; len(b) >= 2; {
		tag := b[0]
		sz := int(b[1] & 0x7f)
		offset := 2
		if b[1]&0x80 != 0 {
			if len(b) < 3 {
				return false
			}
			sz |= int(b[2]) << 7
			offset = 3
		}
		if sz < 0 || len(b) < offset+sz {
			return false
		}
		value := b[offset : offset+sz]
		if tag == 1 && len(value) == 4 {
			v := binary.BigEndian.Uint32(value)
			foundType = v == hdhrTunerType || v == hdhrWildcard
		}
		if tag == 2 && len(value) == 4 {
			v := binary.BigEndian.Uint32(value)
			foundID = v == deviceID || v == hdhrWildcard
		}
		b = b[offset+sz:]
	}
	return foundType && foundID
}
func hdhrResponse(cfg jellyfinConfig) []byte {
	var v [4]byte
	binary.BigEndian.PutUint32(v[:], hdhrTunerType)
	payload := hdhrTLV(1, v[:])
	binary.BigEndian.PutUint32(v[:], hdhrDeviceID(cfg))
	payload = append(payload, hdhrTLV(2, v[:])...)
	payload = append(payload, hdhrTLV(0x10, []byte{1})...)
	payload = append(payload, hdhrTLV(0x2a, []byte(cfg.BaseURL+"\x00"))...)
	payload = append(payload, hdhrTLV(0x27, []byte(cfg.BaseURL+"/lineup.json\x00"))...)
	return hdhrPacket(hdhrTypeReply, payload)
}
func runHDHRDiscovery(conn *net.UDPConn, cfg jellyfinConfig) {
	defer conn.Close()
	buffer := make([]byte, 1500)
	response := hdhrResponse(cfg)
	for {
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, addr, err := conn.ReadFromUDP(buffer)
		if ne, ok := err.(net.Error); ok && ne.Timeout() {
			continue
		}
		if err != nil {
			return
		}
		if addr.IP == nil || !(addr.IP.IsPrivate() || addr.IP.IsLoopback()) {
			continue
		}
		discoveryProbes.Add(1)
		if hdhrParseRequest(buffer[:n], hdhrDeviceID(cfg)) {
			if _, err := conn.WriteToUDP(response, addr); err == nil {
				discoveryReplies.Add(1)
			}
		}
	}
}
func hdhrProbeRequest(cfg jellyfinConfig) []byte {
	var v [4]byte
	binary.BigEndian.PutUint32(v[:], hdhrTunerType)
	payload := hdhrTLV(1, v[:])
	binary.BigEndian.PutUint32(v[:], hdhrWildcard)
	payload = append(payload, hdhrTLV(2, v[:])...)
	return hdhrPacket(hdhrTypeRequest, payload)
}
