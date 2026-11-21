package main

import (
	"errors"
	"flag"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// probeArguments: this test binary runs again inside a box, its own arguments
// behind "--": the probe's name and what it is to check. The bwrap arguments
// around it come first as far as the flag parser is concerned, so the probe
// finds itself by name.
func probeArguments(name string) []string {
	args := flag.Args()
	if i := slices.Index(args, name); i >= 0 {
		return args[i:]
	}
	return nil
}

// Exercise the actual tool launch policy with a child that tests each grant.
// MediaMTX and cloudflared themselves are not needed: the test binary stands in
// for each, wrapped exactly as the tool would be.
func TestLinuxToolBoxes(t *testing.T) {
	if args := probeArguments("probe-tool"); len(args) == 7 {
		role, session, secret, home, listener, resolv := args[1], args[2], args[3], args[4], args[5], args[6]
		for _, path := range []string{secret, home, "/proc/self/status"} {
			if _, err := os.ReadFile(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("should not see %s: %v", path, err)
			}
		}
		if _, present := os.LookupEnv("MOUSEPARTY_SANDBOX_TEST_SECRET"); present {
			t.Fatal("inherited host environment")
		}
		if err := exec.Command("/bin/sh", "-c", "true").Run(); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("expected no other program to start: %v", err)
		}
		if c, err := net.DialTimeout("tcp4", listener, time.Second); err != nil {
			t.Fatalf("the host's network should reach %s: %v", listener, err)
		} else {
			c.Close()
		}
		switch role {
		case "mediamtx":
			if _, err := os.ReadFile(filepath.Join(session, "mediamtx.yml")); err != nil {
				t.Fatalf("the session's folder: %v", err)
			}
			if err := os.WriteFile(filepath.Join(session, "certificates"), []byte("ok"), 0600); err != nil {
				t.Fatalf("the session's folder should be writable: %v", err)
			}
			if wd, err := os.Getwd(); err != nil || wd != session {
				t.Fatalf("expected to work in the session's folder: %q, %v", wd, err)
			}
		case "cloudflared":
			if _, err := os.Stat(filepath.Join(session, "mediamtx.yml")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("cloudflared should not see the session's folder")
			}
			if resolv == "yes" {
				if _, err := os.ReadFile("/etc/resolv.conf"); err != nil {
					t.Fatalf("the resolver's configuration: %v", err)
				}
			}
		}
		return
	}
	requireLinuxBubblewrap(t)
	t.Setenv("MOUSEPARTY_SANDBOX_TEST_SECRET", "host only")
	dir := t.TempDir()
	session := filepath.Join(dir, "session")
	if err := os.Mkdir(session, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(session, "mediamtx.yml"), "config")
	secret, home := filepath.Join(dir, "secret"), filepath.Join(t.TempDir(), "secret")
	writeFile(t, secret, "private")
	writeFile(t, home, "private")
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	h := &Host{tmp: session}
	resolv := "no"
	if real, err := filepath.EvalSymlinks("/etc/resolv.conf"); err == nil {
		if st, err := os.Stat(real); err == nil && st.Mode().IsRegular() {
			resolv = "yes"
		}
	}
	for _, role := range []string{"mediamtx", "cloudflared"} {
		t.Run(role, func(t *testing.T) {
			run, argv, env, err := h.toolWrap(role, exe, []string{"probe"})
			if err != nil {
				t.Fatal(err)
			}
			if env == nil {
				t.Fatal("a boxed tool would inherit the app's environment")
			}
			cmd := exec.Command(run, argv...)
			cmd.Env = env
			cmd.Args[len(cmd.Args)-1] = "-test.run=^TestLinuxToolBoxes$" // in the tool's place
			cmd.Args = append(cmd.Args, "--", "probe-tool", role, session, secret, home, l.Addr().String(), resolv)
			if out, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("sandbox probe: %v\n%s", err, out)
			}
		})
	}
}

// A missing bubblewrap is a failure, never a tool run without its box.
func TestLinuxToolsRequireBubblewrap(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	h := &Host{tmp: t.TempDir()}
	for _, name := range []string{"mediamtx", "cloudflared"} {
		if _, _, _, err := h.toolWrap(name, "/bin/true", nil); !errors.Is(err, exec.ErrNotFound) {
			t.Errorf("%s: expected missing bubblewrap, got %v", name, err)
		}
	}
	if _, err := h.spawn("mediamtx", "/bin/true", "unused"); !errors.Is(err, exec.ErrNotFound) {
		t.Errorf("spawn: expected missing bubblewrap, got %v", err)
	}
}

// The publishing ffmpeg's box: the inputs and the app's own files by their
// pinned inodes, nothing writable, the host's network for the stream out.
func TestLinuxPublishFFmpegIsolation(t *testing.T) {
	if args := probeArguments("probe-publish"); len(args) == 5 {
		input, sibling, own, listener := args[1], args[2], args[3], args[4]
		if got, err := os.ReadFile(input); err != nil || string(got) != "selected" {
			t.Fatalf("expected the pinned input inode: %q, %v", got, err)
		}
		if got, err := os.ReadFile(own); err != nil || string(got) != "ours" {
			t.Fatalf("expected our own temporary file: %q, %v", got, err)
		}
		for _, path := range []string{sibling, "/proc/self/status"} {
			if _, err := os.ReadFile(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("should not see %s: %v", path, err)
			}
		}
		for _, path := range []string{input, own} {
			if err := os.WriteFile(path, []byte("cannot write"), 0600); !errors.Is(err, unix.EROFS) {
				t.Fatalf("expected a read-only mount at %s: %v", path, err)
			}
		}
		if _, present := os.LookupEnv("MOUSEPARTY_SANDBOX_TEST_SECRET"); present {
			t.Fatal("inherited host environment")
		}
		if c, err := net.DialTimeout("tcp4", listener, time.Second); err != nil {
			t.Fatalf("the stream's network should reach the host: %v", err)
		} else {
			c.Close()
		}
		return
	}
	requireLinuxBubblewrap(t)
	t.Setenv("MOUSEPARTY_SANDBOX_TEST_SECRET", "host only")
	dir := t.TempDir()
	input, sibling, own, second := filepath.Join(dir, "movie.mkv"), filepath.Join(dir, "sibling.mkv"),
		filepath.Join(dir, "words.txt"), filepath.Join(dir, "card.png")
	for _, p := range []string{input, sibling, own, second} {
		writeFile(t, p, map[string]string{input: "selected", own: "ours", second: "ours"}[p])
	}
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := contextTimeout(10 * time.Second)
	defer cancel()
	cmd, err := publishFFmpegCommand(ctx, exe, []string{own}, []string{"probe", "-i", input, "-i", second})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range cmd.ExtraFiles {
		defer f.Close()
	}
	cmd.Args[slices.Index(cmd.Args, "probe")] = "-test.run=^TestLinuxPublishFFmpegIsolation$" // in ffmpeg's place
	i := slices.Index(cmd.Args, "-i") // ffmpeg's own arguments stop the flag parser: put them behind "--"
	cmd.Args = append(cmd.Args[:i], append([]string{"--"}, cmd.Args[i:]...)...)
	cmd.Args = append(cmd.Args, "probe-publish", input, sibling, own, l.Addr().String())
	// Replacing the input's name with a symlink to its folder cannot grant the
	// folder, even before bwrap has started: the descriptor holds the inode.
	if err := os.Rename(input, input+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir, input); err != nil {
		t.Fatal(err)
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sandbox probe: %v\n%s", err, out)
	}
}

// The publishing box without bubblewrap fails; no_sandbox runs it as it is.
func TestLinuxPublishFFmpegNoFallback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	movie := filepath.Join(dir, "movie.mkv")
	writeFile(t, movie, "selected")
	bare := func() error {
		cmd, err := ffmpegCmd(t.Context(), nil, "-i", movie, "-f", "rtsp", "rtsp://127.0.0.1:1/live")
		if cmd != nil {
			closeExtra(cmd)
		}
		return err
	}
	if err := bare(); !errors.Is(err, exec.ErrNotFound) {
		t.Fatalf("expected missing bubblewrap: %v", err)
	}
	unboxed(t)
	cmd, err := ffmpegCmd(t.Context(), nil, "-i", movie)
	if err != nil || len(cmd.Args) != 3 || cmd.Args[0] != "ffmpeg" {
		t.Fatalf("expected the bare ffmpeg: %v, %v", cmd.Args, err)
	}
}
