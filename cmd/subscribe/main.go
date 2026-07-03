// Command subscribe is SubScribe, a subtitle translation workbench.
//
// Run with no arguments to open the tool locally in your browser (CLI mode).
// Run `subscribe serve` to run it as a long-lived web server.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ylallemant/subscribe/internal/project"
	"github.com/ylallemant/subscribe/internal/reading"
	"github.com/ylallemant/subscribe/internal/server"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	// Subcommand: default is CLI mode (opens browser); "serve" is headless.
	cmd := "open"
	if len(args) > 0 && args[0] == "serve" {
		cmd, args = "serve", args[1:]
	} else if len(args) > 0 && args[0] == "open" {
		args = args[1:]
	}

	fs := flag.NewFlagSet("subscribe", flag.ContinueOnError)
	addr := fs.String("addr", envOr("SUBSCRIBE_ADDR", ":8080"), "address the web server listens on")
	fps := fs.Float64("fps", envFloat("SUBSCRIBE_FPS", 25), "frame rate used to convert :FF frames to a duration")
	metric := fs.String("reading-metric", envOr("SUBSCRIBE_READING_METRIC", "cps"), "reading-speed metric: cps or wps")
	cpsMax := fs.Float64("cps-max", envFloat("SUBSCRIBE_CPS_MAX", 17), "max comfortable characters per second")
	wpsMax := fs.Float64("wps-max", envFloat("SUBSCRIBE_WPS_MAX", 3), "max comfortable words per second")
	noBrowser := fs.Bool("no-browser", envBool("SUBSCRIBE_NO_BROWSER", false), "do not open a browser (CLI mode)")
	dataDir := fs.String("data-dir", envOr("SUBSCRIBE_DATA_DIR", defaultDataDir()), "directory where projects are stored")
	disableDelete := fs.Bool("disable-delete", envBool("SUBSCRIBE_DISABLE_DELETE", false), "disable project deletion")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg := reading.Default()
	cfg.Metric = reading.Metric(*metric)
	cfg.CPSMax = *cpsMax
	cfg.WPSMax = *wpsMax

	store, err := project.NewStore(*dataDir)
	if err != nil {
		return fmt.Errorf("open data dir %s: %w", *dataDir, err)
	}

	handler := server.New(server.Options{FPS: *fps, Reading: cfg, Store: store, DisableDelete: *disableDelete})

	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", *addr, err)
	}
	url := "http://" + browserAddr(ln.Addr().String())

	httpSrv := &http.Server{Handler: handler}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	errc := make(chan error, 1)
	go func() { errc <- httpSrv.Serve(ln) }()

	fmt.Printf("subscribe listening on %s\n", url)
	if cmd == "open" && !*noBrowser {
		if err := openBrowser(url); err != nil {
			fmt.Fprintf(os.Stderr, "could not open browser (%v); visit %s manually\n", err, url)
		}
	}

	select {
	case <-ctx.Done():
		fmt.Println("\nshutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return httpSrv.Shutdown(shutdownCtx)
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

// browserAddr turns a listener address like "[::]:8080" into something a
// browser can open ("localhost:8080").
func browserAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || host == "::" || host == "0.0.0.0" {
		host = "localhost"
	}
	return net.JoinHostPort(host, port)
}

func openBrowser(url string) error {
	var bin string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		bin = "open"
	case "windows":
		bin, args = "rundll32", []string{"url.dll,FileProtocolHandler"}
	default:
		bin = "xdg-open"
	}
	return exec.Command(bin, append(args, url)...).Start()
}

// defaultDataDir is ~/.subscribe, falling back to ./subscribe-data.
func defaultDataDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".subscribe")
	}
	return "subscribe-data"
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		var f float64
		if _, err := fmt.Sscanf(v, "%g", &f); err == nil {
			return f
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	switch os.Getenv(key) {
	case "1", "true", "TRUE", "yes":
		return true
	case "0", "false", "FALSE", "no":
		return false
	default:
		return def
	}
}
