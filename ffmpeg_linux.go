package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"golang.org/x/sys/unix"
)

func offlineFFmpegCommand(ctx context.Context, bin, path string, args func(string) []string) (*exec.Cmd, error) {
	if sandboxOff.Load() {
		return exec.CommandContext(ctx, bin, args(path)...), nil
	}
	// Keep the extension for formats that ffmpeg recognizes by name. The
	// selected path itself never appears in the mount arguments or ffmpeg argv.
	input := "/input/selected" + filepath.Ext(path)
	cmd, err := bubblewrapCommand(ctx, "offline ffmpeg", []string{
		"--ro-bind", bin, "/app/ffmpeg", "--ro-bind-fd", "3", input, "--remount-ro", "/",
	}, append([]string{"/app/ffmpeg"}, args(input)...)...)
	if err != nil {
		return nil, err
	}
	// O_PATH does not open a device or wait for a FIFO. Check the exact inode
	// we hand bwrap, never stat a name and then let bwrap open it again.
	fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open selected media: %w", err)
	}
	f := os.NewFile(uintptr(fd), "selected media")
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !st.Mode().IsRegular() {
		f.Close()
		return nil, fmt.Errorf("selected media is not a regular file")
	}
	cmd.ExtraFiles = []*os.File{f} // bwrap consumes fd 3 during setup; caller closes its copy
	return cmd, nil
}

// publishFFmpegCommand: the publishing ffmpeg (stream.go's Player, and the cards
// and covers that become its inputs) in its box. As the offline box does, every
// file it may read — each movie named by -i, and the app's own temporary files,
// which the subtitle and drawtext filters name by their paths — is handed over
// pinned by descriptor, so nothing can be swapped under a name while it starts;
// unlike the offline box the movies keep their names, which those filters need.
// It writes nothing: its outputs are the stream it publishes and -progress
// pipe:1. The network is the host's own, to reach MediaMTX over RTSP — and with
// it whatever else is on it, which bubblewrap cannot narrow to one port as the
// macOS box does. Callers close cmd.ExtraFiles once the command has started.
func publishFFmpegCommand(ctx context.Context, bin string, own []string, args []string) (*exec.Cmd, error) {
	if ctx == nil {
		ctx = context.Background() // the Player starts ffmpeg without one, to outlive any call
	}
	if sandboxOff.Load() {
		return exec.CommandContext(ctx, bin, args...), nil
	}
	mounts := []string{"--ro-bind", bin, "/app/ffmpeg",
		// cards' words and burned-in subtitles are drawn with the system's fonts,
		// found through fontconfig (the macOS box reaches its font daemon instead)
		"--ro-bind-try", "/etc/fonts", "/etc/fonts",
		"--ro-bind-try", "/usr/share/fonts", "/usr/share/fonts",
		"--ro-bind-try", "/usr/local/share/fonts", "/usr/local/share/fonts",
		"--ro-bind-try", "/usr/share/fontconfig", "/usr/share/fontconfig",
		"--ro-bind-try", "/var/cache/fontconfig", "/var/cache/fontconfig"}
	var extra []*os.File
	bound := map[string]bool{}
	addFile := func(path string) {
		fd, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
		if err != nil {
			return
		}
		f := os.NewFile(uintptr(fd), path)
		st, err := f.Stat()
		if err != nil || !(st.Mode().IsRegular() || st.Mode().IsDir()) {
			f.Close()
			return
		}
		dest, err := filepath.Abs(path)
		if err != nil || bound[dest] {
			f.Close()
			return
		}
		bound[dest] = true
		mounts = append(mounts, "--ro-bind-fd", strconv.Itoa(3+len(extra)), dest)
		extra = append(extra, f)
	}
	for i, a := range args {
		if i > 0 && args[i-1] == "-i" {
			if _, err := os.Stat(a); err == nil || filepath.IsAbs(a) {
				addFile(a)
			}
		}
	}
	for _, p := range own {
		addFile(p)
	}
	run, argv, err := bubblewrapArgv("publishing ffmpeg", true, "", mounts, append([]string{"/app/ffmpeg"}, args...)...)
	if err != nil {
		for _, f := range extra {
			f.Close()
		}
		return nil, err
	}
	cmd := exec.CommandContext(ctx, run, argv...)
	cmd.Env = []string{}
	cmd.ExtraFiles = extra
	return cmd, nil
}
