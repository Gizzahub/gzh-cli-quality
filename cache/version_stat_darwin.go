// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

//go:build darwin

package cache

import (
	"os"
	"syscall"
)

func statIdentity(info os.FileInfo) (fileIdentity, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return fileIdentity{}, false
	}
	return fileIdentity{
		device:       uint64(uint32(st.Dev)), //nolint:gosec // G115: dev_t bits, compared only for equality.
		inode:        st.Ino,
		links:        uint64(st.Nlink),
		modTimeNs:    st.Mtimespec.Nano(),
		changeTimeNs: st.Ctimespec.Nano(),
	}, true
}
