// SPDX-FileCopyrightText: 2026 Tobias Böhm <code@aibor.de>
//
// SPDX-License-Identifier: GPL-3.0-or-later

//go:generate -command myenv env CGO_ENABLED=0 GOOS=linux GOFLAGS=-buildvcs=false
//go:generate myenv GOARCH=amd64 go build -o ../bin/amd64 arch.go
//go:generate myenv GOARCH=arm64 go build -o ../bin/arm64 arch.go
//go:generate myenv GOARCH=riscv64 go build -o ../bin/riscv64 arch.go

//go:build archbin

package main

func main() {}
