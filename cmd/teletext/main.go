// Standalone development server for the AussieTV Teletext generator.
// Run: go run ./cmd/teletext -config config/teletext.json -listen 127.0.0.1:45783
package main

import (
	"aussietv/teletext"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	cfg := flag.String("config", "config/teletext.json", "teletext page-map config JSON")
	listen := flag.String("listen", "127.0.0.1:45783", "loopback HTTP bind")
	export := flag.String("export", "", "output directory for .tti pages, instead of serving HTTP")
	flag.Parse()
	b, e := os.ReadFile(*cfg)
	if e != nil {
		log.Fatal(e)
	}
	c, e := teletext.LoadConfig(b)
	if e != nil {
		log.Fatal(e)
	}
	s := teletext.New(c, nil)
	if *export != "" {
		os.MkdirAll(*export, 0755)
		for _, n := range s.Listed() {
			if n >= 800 {
				continue
			}
			p, e := s.Page(n)
			if e != nil {
				log.Fatal(e)
			}
			if e := os.WriteFile(filepath.Join(*export, fmtPage(n)), teletext.TTI(p), 0644); e != nil {
				log.Fatal(e)
			}
		}
		return
	}
	log.Printf("AussieTV Teletext on http://%s/teletext/", *listen)
	log.Fatal(http.ListenAndServe(*listen, s.Handler()))
}
func fmtPage(n int) string { return fmt.Sprintf("%03d.tti", n) }
