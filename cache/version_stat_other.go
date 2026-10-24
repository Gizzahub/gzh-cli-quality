// Copyright (c) 2025 Gizzahub
// SPDX-License-Identifier: MIT

//go:build !darwin && !linux

package cache

import "os"

// statIdentity reports false where inode and ctime are not read, so the
// version is probed on every run instead of trusting a weaker key.
func statIdentity(os.FileInfo) (fileIdentity, bool) {
	return fileIdentity{}, false
}
