package main

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

// health checks use the configured local listeners, never upstream EPG or
// video requests. Credentials stay inside the process, not CLI arguments.
func localHealthURL(bind, path string) (*url.URL, error) {
	host, port, err := net.SplitHostPort(bind)
	if err != nil {
		return nil, errors.New("invalid health listener address")
	}
	switch host {
	case "", "0.0.0.0":
		host = "127.0.0.1"
	case "::":
		host = "::1"
	}
	return &url.URL{Scheme: "http", Host: net.JoinHostPort(host, port), Path: path}, nil
}

func checkLocalHealth(client *http.Client, bind, path, host, user, password, key string) error {
	u, err := localHealthURL(bind, path)
	if err != nil {
		return err
	}
	if key != "" {
		u.RawQuery = url.Values{"key": {key}}.Encode()
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return errors.New("cannot create local health request")
	}
	if host != "" {
		req.Host = host
		req.SetBasicAuth(user, password)
	}
	resp, err := client.Do(req)
	if err != nil {
		// net/http errors may contain the feed key in the URL.
		return errors.New("configured listener is not responding")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("configured listener returned unhealthy status")
	}
	return nil
}

func serverHealthcheck() error {
	web, err := dockerWebConfigFromEnv(os.Getenv("AUSSIETV_MODE") == "server")
	if err != nil {
		return err
	}
	feed, enabled, err := jellyfinConfigFromEnv()
	if err != nil {
		return err
	}
	if web == nil && !enabled {
		return errors.New("no server health target configured")
	}
	client := &http.Client{
		Timeout: 3*time.Second,
		Transport: &http.Transport{Proxy: nil, DisableKeepAlives: true},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	if web != nil {
		if err := checkLocalHealth(client, web.bind, "/api/health", web.origin.Host, os.Getenv("AUSSIETV_WEB_USER"), os.Getenv("AUSSIETV_WEB_PASSWORD"), ""); err != nil {
			return err
		}
	}
	if enabled {
		return checkLocalHealth(client, feed.Bind, "/jellyfin/health", "", "", "", feed.Key)
	}
	return nil
}
