// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

//go:build linux

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
		device:       uint64(st.Dev), //nolint:unconvert // uint32 on mips*.
		inode:        st.Ino,
		links:        uint64(st.Nlink), //nolint:unconvert // uint32 on 386, arm, arm64 and others.
		modTimeNs:    st.Mtim.Nano(),
		changeTimeNs: st.Ctim.Nano(),
	}, true
}
