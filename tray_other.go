//go:build !windows

package main

import "errors"
func startTray(_, _, _ string, _ func())error{return errors.New("system tray supported on Windows only")}
func nativeAlert(_, _ string){}
