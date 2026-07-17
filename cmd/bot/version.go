package main

// Build-time metadata. Override via -ldflags, for example:
//
//	-X main.Version=1.0.0 -X main.GitCommit=$(git rev-parse --short HEAD) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)
