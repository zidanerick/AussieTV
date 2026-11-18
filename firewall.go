package main

// Firewall access is intentionally independent of the authenticated M3U/XMLTV
// listener. Windows elevation occurs only as a direct result of clicking Save
// or Repair in the local Settings page, never on startup or from the LAN feed.
type firewallPorts struct{ TCP, DiscoveryUDP, DiscoveryHTTP bool }

func firewallFor(cfg desktopSettings) firewallPorts {
	lan := cfg.Enabled && cfg.LAN
	return firewallPorts{TCP: lan, DiscoveryUDP: lan && cfg.Discovery, DiscoveryHTTP: lan && cfg.Discovery}
}

var firewallUpdater = applyWindowsFirewall

func updateFirewall(previous, next desktopSettings, repair bool) error {
	before, after := firewallFor(previous), firewallFor(next)
	if !repair && before == after {
		return nil
	}
	return firewallUpdater(after)
}
