package main

import (
	"errors"
	"testing"
)

func TestFirewallPlansAreScopedToLANAndOptionalDiscovery(t *testing.T) {
	cases := []struct {
		enabled, lan, discover bool
		expected               firewallPorts
	}{
		{false, false, false, firewallPorts{}},
		{true, false, false, firewallPorts{}},
		{true, true, false, firewallPorts{TCP: true}},
		{true, true, true, firewallPorts{TCP: true, DiscoveryUDP: true, DiscoveryHTTP: true}},
	}
	for _, tc := range cases {
		got := firewallFor(desktopSettings{Enabled: tc.enabled, LAN: tc.lan, Discovery: tc.discover})
		if got != tc.expected {
			t.Fatalf("%+v: got %+v, expected %+v", tc, got, tc.expected)
		}
	}
}

func TestFirewallUpdateOnlyOnChangeOrExplicitRepair(t *testing.T) {
	previous := firewallUpdater
	t.Cleanup(func() { firewallUpdater = previous })
	calls := []firewallPorts{}
	firewallUpdater = func(p firewallPorts) error { calls = append(calls, p); return nil }
	disabled := desktopSettings{}
	lan := desktopSettings{Enabled: true, LAN: true}
	discovered := desktopSettings{Enabled: true, LAN: true, Discovery: true}
	if err := updateFirewall(disabled, disabled, false); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("unexpected calls: %v", calls)
	}
	if err := updateFirewall(disabled, lan, false); err != nil {
		t.Fatal(err)
	}
	if err := updateFirewall(lan, discovered, false); err != nil {
		t.Fatal(err)
	}
	if err := updateFirewall(discovered, discovered, true); err != nil {
		t.Fatal(err)
	}
	if err := updateFirewall(discovered, disabled, false); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 4 || calls[0] != (firewallPorts{TCP: true}) || calls[1] != (firewallPorts{TCP: true, DiscoveryUDP: true, DiscoveryHTTP: true}) || calls[3] != (firewallPorts{}) {
		t.Fatalf("unexpected firewall transitions: %+v", calls)
	}
}
func TestFirewallUpdateReturnsElevationError(t *testing.T) {
	old := firewallUpdater
	t.Cleanup(func() { firewallUpdater = old })
	denied := errors.New("UAC denied")
	firewallUpdater = func(firewallPorts) error { return denied }
	if err := updateFirewall(desktopSettings{}, desktopSettings{Enabled: true, LAN: true}, false); !errors.Is(err, denied) {
		t.Fatalf("want denial error, got %v", err)
	}
}
