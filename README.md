# go-pidonetest

[![PkgGoDev](https://pkg.go.dev/badge/github.com/aibor/go-pideonetest)](https://pkg.go.dev/github.com/aibor/go-pidonetest)

go-pidonetest is a library and binary QEMU wrapper for running go tests in an 
isolated system. See [doc.go](doc.go) for a package description.

The package uses itself for testing, so see 
[pidonetest_test.go](pidonetest_test.go) for a real life example.

For now, the only architecture supported is x86_64. Also the test binaries must
compile statically (set `CGO_ENABLED=0`).
