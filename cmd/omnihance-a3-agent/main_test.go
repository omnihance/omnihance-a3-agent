package main

import (
	"bytes"
	"context"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestAgentProcess(t *testing.T) {
	if os.Getenv("OMNIHANCE_TEST_PROCESS") == "1" {
		main()
		return
	}

	for _, sig := range []os.Signal{nil, syscall.SIGTERM, os.Interrupt} {
		name := "listen failure"
		if sig != nil {
			name = sig.String()
		}

		t.Run(name, func(t *testing.T) {
			if sig != nil && runtime.GOOS == "windows" {
				t.Skip("Windows does not support sending Unix signals to a child process")
			}

			listener, err := net.Listen("tcp", ":0")
			if err != nil {
				t.Fatal(err)
			}

			t.Cleanup(func() { _ = listener.Close() })
			_, port, err := net.SplitHostPort(listener.Addr().String())
			if err != nil {
				t.Fatal(err)
			}

			if sig != nil {
				_ = listener.Close()
			}

			t.Setenv("OMNIHANCE_TEST_PROCESS", "1")
			t.Setenv("PORT", port)
			t.Setenv("METRICS_ENABLED", "false")
			t.Setenv("DATABASE_URL", "file:agent.db?mode=rwc")
			t.Setenv("LOG_DIR", "logs")
			t.Setenv("REVISIONS_DIRECTORY", ".revisions")
			t.Setenv("BACKUPS_DIRECTORY", ".backups")
			t.Setenv("DIRECTORY_DOWNLOADS_DIRECTORY", ".directory-download")
			executable, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestAgentProcess$")
			cmd.Dir = t.TempDir()
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}

			defer func() { _ = cmd.Process.Kill() }()
			if sig != nil {
				client := &http.Client{Timeout: 200 * time.Millisecond}
				defer client.CloseIdleConnections()
				for ctx.Err() == nil {
					response, err := client.Get("http://127.0.0.1:" + port + "/health")
					if err == nil {
						_ = response.Body.Close()
						if response.StatusCode == http.StatusOK {
							break
						}
					}

					time.Sleep(20 * time.Millisecond)
				}

				if err := cmd.Process.Signal(sig); err != nil {
					_ = cmd.Wait()
					t.Fatalf("signal: %v\n%s", err, output.String())
				}
			}

			err = cmd.Wait()
			if ctx.Err() != nil || (sig == nil && cmd.ProcessState.ExitCode() != 1) || (sig != nil && err != nil) {
				t.Fatalf("unexpected exit: %v (context: %v)\n%s", err, ctx.Err(), output.String())
			}

			if !strings.Contains(output.String(), "backup service stopped") {
				t.Fatalf("deferred service cleanup did not run:\n%s", output.String())
			}

			if sig != nil && strings.Contains(output.String(), "Could not start Omnihance") {
				t.Fatalf("normal shutdown reported as startup failure:\n%s", output.String())
			}
		})
	}
}
