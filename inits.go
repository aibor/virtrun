package virtrun

import (
	"embed"
	"fmt"
	"io/fs"
)

// Pre-compile init programs for all supported architectures. Statically linked
// so they can be used on any host platform.
//
//go:generate env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -buildvcs=false -ldflags "-s -w" -o inits/amd64 ./inits/main.go
//go:generate env CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -buildvcs=false -ldflags "-s -w" -o inits/arm64 ./inits/main.go

// Embed pre-compiled init programs explicitly to trigger build time errors.
//
//go:embed inits/amd64 inits/arm64
var _inits embed.FS

func InitFor(arch string) (fs.File, error) {
	switch arch {
	case "amd64":
		return _inits.Open("inits/amd64")
	case "arm64":
		return _inits.Open("inits/arm64")
	default:
		return nil, fmt.Errorf("arch not supported: %s", arch)
	}
}
