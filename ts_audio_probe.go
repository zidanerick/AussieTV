package main

// Lightweight MPEG-TS PAT/PMT and PES-presence inspector. This verifies that
// FFmpeg actually emits an audio PID before the experimental Jellyfin relay
// sends a successful response. It does not decode the audio or infer its
// language, and it only examines the first bounded output window.
type tsAudioProbe struct {
	pmtPID          uint16
	hasPMT          bool
	hasVideo        bool
	hasAudio        bool
	videoPacketSeen bool
	audioPacketSeen bool
	audioPIDs       map[uint16]struct{}
	videoPIDs       map[uint16]struct{}
}

func (t *tsAudioProbe) audioReady() bool {
	return t.hasVideo && t.hasAudio && t.videoPacketSeen && t.audioPacketSeen
}

func (t *tsAudioProbe) add(packet []byte) {
	if len(packet) != 188 || packet[0] != 0x47 || packet[1]&0x80 != 0 {
		return
	}
	pid := (uint16(packet[1]&0x1f) << 8) | uint16(packet[2])
	control := (packet[3] >> 4) & 3
	if control != 1 && control != 3 {
		return
	}
	pos := 4
	if control == 3 {
		pos += 1 + int(packet[4])
	}
	if pos >= len(packet) {
		return
	}
	payload := packet[pos:]
	if _, ok := t.audioPIDs[pid]; ok && len(payload) > 0 {
		t.audioPacketSeen = true
	}
	if _, ok := t.videoPIDs[pid]; ok && len(payload) > 0 {
		t.videoPacketSeen = true
	}
	// Section payloads begin with a pointer field on unit-start packets.
	if packet[1]&0x40 == 0 || len(payload) == 0 {
		return
	}
	pointer := int(payload[0])
	if 1+pointer >= len(payload) {
		return
	}
	section := payload[1+pointer:]
	if len(section) < 3 {
		return
	}
	length := int(section[1]&0x0f)<<8 | int(section[2])
	if length < 4 || 3+length > len(section) {
		return // No attempt to parse fragmented PSI as complete.
	}
	section = section[:3+length]
	if pid == 0 && section[0] == 0x00 && length >= 9 {
		// PAT: program entries start 8 bytes into the section; last 4
		// bytes are CRC. Program number 0 is the network PID, not PMT.
		for i := 8; i+4 <= len(section)-4; i += 4 {
			program := uint16(section[i])<<8 | uint16(section[i+1])
			if program != 0 {
				t.pmtPID = (uint16(section[i+2]&0x1f) << 8) | uint16(section[i+3])
				t.hasPMT = true
				break
			}
		}
		return
	}
	if !t.hasPMT || pid != t.pmtPID || section[0] != 0x02 || length < 13 {
		return
	}
	// PMT: after PCR PID and program descriptors, each elementary stream
	// entry specifies a stream type, PID, and stream-specific descriptors.
	programInfoLen := int(section[10]&0x0f)<<8 | int(section[11])
	for i := 12 + programInfoLen; i+5 <= len(section)-4; {
		streamType := section[i]
		streamPID := uint16(section[i+1]&0x1f)<<8 | uint16(section[i+2])
		infoLen := int(section[i+3]&0x0f)<<8 | int(section[i+4])
		if i+5+infoLen > len(section)-4 {
			return
		}
		desc := section[i+5 : i+5+infoLen]
		switch streamType {
		case 0x01, 0x02, 0x10, 0x1b, 0x24:
			t.hasVideo = true
			if t.videoPIDs == nil {
				t.videoPIDs = make(map[uint16]struct{})
			}
			t.videoPIDs[streamPID] = struct{}{}
		case 0x03, 0x04, 0x0f, 0x11, 0x81, 0x87:
			t.addAudioPID(streamPID)
		case 0x06:
			if hasPrivateAudioDescriptor(desc) {
				t.addAudioPID(streamPID)
			}
		}
		i += 5 + infoLen
	}
}

func (t *tsAudioProbe) addAudioPID(pid uint16) {
	t.hasAudio = true
	if t.audioPIDs == nil {
		t.audioPIDs = make(map[uint16]struct{})
	}
	t.audioPIDs[pid] = struct{}{}
}

func hasPrivateAudioDescriptor(data []byte) bool {
	for len(data) >= 2 {
		n := int(data[1])
		if n+2 > len(data) {
			return false
		}
		if data[0] == 0x6a || data[0] == 0x7a || data[0] == 0x7b {
			return true
		}
		if data[0] == 0x05 && n >= 4 {
			id := string(data[2:6])
			if id == "AC-3" || id == "EAC3" {
				return true
			}
		}
		data = data[2+n:]
	}
	return false
}
