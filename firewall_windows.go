//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

const firewallArgPrefix = "--aussietv-windows-firewall="

var (
	shellExecuteEx = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteExW")
	shellExecute   = syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW")
	winWait        = syscall.NewLazyDLL("kernel32.dll").NewProc("WaitForSingleObject")
	winExitCode    = syscall.NewLazyDLL("kernel32.dll").NewProc("GetExitCodeProcess")
	winCloseHandle = syscall.NewLazyDLL("kernel32.dll").NewProc("CloseHandle")
)

type shellExecuteInfo struct {
	Size       uint32
	Mask       uint32
	Window     uintptr
	Verb       *uint16
	File       *uint16
	Parameters *uint16
	Directory  *uint16
	Show       int32
	Instance   uintptr
	IDList     uintptr
	Class      *uint16
	ClassKey   uintptr
	HotKey     uint32
	Icon       uintptr
	Process    uintptr
}

func openWindowsDefaultBrowser(addr string) error {
	file, err := syscall.UTF16PtrFromString(addr)
	if err != nil {
		return err
	}
	verb, _ := syscall.UTF16PtrFromString("open")
	r, _, callErr := shellExecute.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, 1)
	if r <= 32 {
		return fmt.Errorf("Windows URL association failed (code %d): %v", r, callErr)
	}
	return nil
}

func firewallArg(p firewallPorts) string {
	switch {
	case p.TCP && p.DiscoveryUDP:
		return "both"
	case p.TCP:
		return "feed"
	default:
		return "off"
	}
}

func applyWindowsFirewall(p firewallPorts) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	f, err := syscall.UTF16PtrFromString(exe)
	if err != nil {
		return err
	}
	v, _ := syscall.UTF16PtrFromString("runas")
	params, _ := syscall.UTF16PtrFromString(firewallArgPrefix + firewallArg(p))
	info := shellExecuteInfo{
		Size: uint32(unsafe.Sizeof(shellExecuteInfo{})),
		Mask: 0x00000040, // SEE_MASK_NOCLOSEPROCESS
		Verb: v, File: f, Parameters: params, Show: 0,
	}
	r, _, callErr := shellExecuteEx.Call(uintptr(unsafe.Pointer(&info)))
	if r == 0 {
		return fmt.Errorf("administrator approval was cancelled or unavailable: %v", callErr)
	}
	if info.Process == 0 {
		return fmt.Errorf("Windows did not provide a firewall helper process")
	}
	defer winCloseHandle.Call(info.Process)
	waitResult, _, waitErr := winWait.Call(info.Process, 120000) // 2 minutes to approve UAC and configure rules
	if waitResult != 0 {
		return fmt.Errorf("firewall configuration did not finish (wait=%d): %v", waitResult, waitErr)
	}
	var exitCode uint32
	ok, _, errCode := winExitCode.Call(info.Process, uintptr(unsafe.Pointer(&exitCode)))
	if ok == 0 {
		return fmt.Errorf("cannot read firewall configuration result: %v", errCode)
	}
	if exitCode != 0 {
		return fmt.Errorf("Windows Firewall configuration failed (helper exit code %d); check administrator rights", exitCode)
	}
	return nil
}

func maybeRunFirewallHelper(args []string) (int, bool) {
	if len(args) != 2 || !strings.HasPrefix(args[1], firewallArgPrefix) {
		return 0, false
	}
	value := strings.TrimPrefix(args[1], firewallArgPrefix)
	var p firewallPorts
	switch value {
	case "off":
	case "feed":
		p.TCP = true
	case "both":
		p.TCP = true
		p.DiscoveryUDP = true
		p.DiscoveryHTTP = true
	default:
		return 2, true
	}
	if err := configureFirewallRules(p); err != nil {
		log.Printf("AussieTV firewall helper failed: %v", err)
		return 1, true
	}
	return 0, true
}

func configureFirewallRules(p firewallPorts) error {
	// Names are scoped to AussieTV. Never alter other applications' rules.
	const feedName = "AussieTV Jellyfin Feed Private LAN"
	const discoveryName = "AussieTV HDHomeRun Discovery Private LAN"
	const hdhrHTTPName = "AussieTV HDHomeRun HTTP Discovery Private LAN"
	rules := []struct {
		name, protocol, port string
		enabled              bool
	}{
		{feedName, "TCP", "45782", p.TCP},
		{discoveryName, "UDP", "65001", p.DiscoveryUDP},
		{hdhrHTTPName, "TCP", "80", p.DiscoveryHTTP},
	}
	for _, rule := range rules {
		// Missing rules are normal. netsh returns a nonzero exit code for these.
		_ = exec.Command("netsh.exe", "advfirewall", "firewall", "delete", "rule", "name="+rule.name).Run()
		if rule.enabled {
			output, err := exec.Command("netsh.exe", "advfirewall", "firewall", "add", "rule",
				"name="+rule.name, "dir=in", "action=allow", "enable=yes", "profile=private",
				"remoteip=localsubnet", "protocol="+rule.protocol, "localport="+rule.port).CombinedOutput()
			if err != nil {
				return fmt.Errorf("add %s: %w (%s)", rule.name, err, strings.TrimSpace(string(output)))
			}
		}
	}
	return nil
}
