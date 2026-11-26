package main

import (
	"net"
	"net/netip"
	"strings"
)

// IsGlobalUnicast alone also accepts shared carrier NAT, documentation,
// benchmark and protocol-assignment networks. Block those explicitly.
var specialRemoteRanges = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),netip.MustParsePrefix("64:ff9b::/96"),
	netip.MustParsePrefix("64:ff9b:1::/48"),netip.MustParsePrefix("100::/64"),
	netip.MustParsePrefix("2001::/23"),netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),netip.MustParsePrefix("3fff::/20"),
}
func publicMediaIP(ip net.IP)bool{
	a,ok:=netip.AddrFromSlice(ip)
	if !ok{return false}
	a=a.Unmap()
	if !a.IsGlobalUnicast()||a.IsPrivate()||a.IsLoopback()||a.IsLinkLocalUnicast(){return false}
	for _,p:=range specialRemoteRanges{if p.Contains(a){return false}}
	return true
}
func localMediaHost(host string)bool {
	h:=strings.TrimSuffix(strings.ToLower(host),".")
	return h=="localhost"||strings.HasSuffix(h,".localhost")||strings.HasSuffix(h,".local")||strings.HasSuffix(h,".internal")||strings.HasSuffix(h,".home.arpa")||strings.Contains(h,"%")
}
