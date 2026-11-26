package main

import (
	"errors"
	"net/url"
	"strings"
)

var errInvalidHLSKey = errors.New("invalid HLS key declaration")
var errHLSKeyFormat = errors.New("non-identity HLS key format is unsupported")
var errHLSKeyMethod = errors.New("unsupported HLS encryption method")
var errHLSKeyVersion = errors.New("unsupported HLS key format version")

// RFC 8216 defines identity as the default KEYFORMAT. An explicit
// identity declaration is not a DRM license scheme. Preserve ordinary
// AES-128 HLS, already supported when KEYFORMAT is omitted, without
// adding a decryption implementation, acquiring licenses, or removing
// any encryption tags. Non-identity formats and unsupported methods
// remain blocked. Every ordinary key URI uses the same guarded relay.
func validateHLSKeyTag(line string, base *url.URL) error {
	session := strings.HasPrefix(line, "#EXT-X-SESSION-KEY:")
	var raw string
	switch {
	case session:
		raw = strings.TrimPrefix(line, "#EXT-X-SESSION-KEY:")
	case strings.HasPrefix(line, "#EXT-X-KEY:"):
		raw = strings.TrimPrefix(line, "#EXT-X-KEY:")
	default:
		return errInvalidHLSKey
	}
	attrs, ok := fastTuneAttributes(raw)
	if !ok {
		return errInvalidHLSKey
	}
	if format, present := attrs["KEYFORMAT"]; present && format != "identity" {
		return errHLSKeyFormat
	}
	switch attrs["METHOD"] {
	case "NONE":
		if session || len(attrs) != 1 {
			return errInvalidHLSKey
		}
		return nil
	case "AES-128":
		// Only the existing ordinary AES-128 playback mechanism is supported.
	default:
		if attrs["METHOD"] == "" {
			return errInvalidHLSKey
		}
		return errHLSKeyMethod
	}
	if version, present := attrs["KEYFORMATVERSIONS"]; present && version != "1" {
		return errHLSKeyVersion
	}
	for name := range attrs {
		switch name {
		case "METHOD", "URI", "IV", "KEYFORMAT", "KEYFORMATVERSIONS":
		default:
			return errInvalidHLSKey
		}
	}
	// Only a normal quoted key URI can be rewritten by our existing relay.
	// Never turn a DRM URI into a different key source or accept an external
	// protocol, private address, unquoted URI, or malformed reference.
	match := uriRegexp.FindAllStringSubmatch(raw, -1)
	if attrs["URI"] == "" || len(match) != 1 || match[0][1] != attrs["URI"] {
		return errInvalidHLSKey
	}
	if _, err := validatedPlaylistRef(base, attrs["URI"]); err != nil {
		return err
	}
	return nil
}
