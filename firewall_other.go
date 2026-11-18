//go:build !windows

package main

func applyWindowsFirewall(ports firewallPorts) error   { return nil }
func maybeRunFirewallHelper(args []string) (int, bool) { return 0, false }
func openWindowsDefaultBrowser(addr string) error      { return nil }
