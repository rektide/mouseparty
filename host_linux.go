package main

import (
	"os"
	"path/filepath"
)

// toolWrap (Linux): MediaMTX and cloudflared in a bubblewrap box, like the
// viewer site (bubblewrap_linux.go) — and like it, mandatory: no bubblewrap, no
// tool, never a tool without its box. MediaMTX keeps the session's folder (its
// config, the stand-by stream, the certificates it writes beside itself) and
// works there; cloudflared gets no file of ours, only the system's certificates
// and resolver for its TLS and DNS. Both keep the host's network — MediaMTX to
// listen on it (viewers reach the ICE port, the app and ffmpeg its loopback
// ones), cloudflared to reach Cloudflare and the site's port — and neither sees
// the home folder, another program to start, or any of the app's environment.
func (h *Host) toolWrap(name, bin string, args []string) (string, []string, []string, error) {
	if sandboxOff.Load() {
		return bin, args, nil, nil
	}
	mounts := []string{"--ro-bind", bin, "/app/" + name}
	chdir := ""
	switch name {
	case "mediamtx":
		if h.tmp != "" {
			mounts = append(mounts, "--bind", h.tmp, h.tmp)
			chdir = h.tmp
		}
	case "cloudflared":
		mounts = append(mounts, cloudflaredMounts()...)
	}
	run, argv, err := bubblewrapArgv(name, true, chdir, mounts, append([]string{"/app/" + name}, args...)...)
	if err != nil {
		return "", nil, nil, err
	}
	return run, argv, []string{}, nil
}

// cloudflaredMounts: the system's certificate store, however this distribution
// lays it out, and the resolver's configuration. resolv.conf is often a link
// into /run: bind what it points at, a stub on the host's network this box shares.
func cloudflaredMounts() []string {
	mounts := []string{}
	for _, path := range []string{"/etc/ssl/certs", "/etc/ssl/cert.pem", "/etc/pki", "/etc/ca-certificates"} {
		mounts = append(mounts, "--ro-bind-try", path, path)
	}
	if real, err := filepath.EvalSymlinks("/etc/resolv.conf"); err == nil {
		if st, err := os.Stat(real); err == nil && st.Mode().IsRegular() {
			mounts = append(mounts, "--ro-bind", real, "/etc/resolv.conf")
		}
	}
	return mounts
}
