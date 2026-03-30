package main

import (
	"github.com/y0anfa/rhino/cmd"
	"github.com/y0anfa/rhino/internal/logger"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	defer logger.Sync()
	cmd.SetVersionInfo(version, commit, date)
	cmd.Execute()
}
