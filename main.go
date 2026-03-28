package main

import (
	"github.com/y0anfa/rhino/cmd"
	"github.com/y0anfa/rhino/internal/logger"
)

func main() {
	defer logger.Sync()
	cmd.Execute()
}
