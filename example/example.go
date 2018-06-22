package main

import (
	"os"

	"github.com/lpar/blammo/log"
)

func example() {
	log.Error().Caller().Msg("Some terrible error occurred")
}

func main() {
	log.Info().Line().Msg("Example program starting")
	log.Debug().Int("x", 6).Int("y", 42).Msg("Debug trace")
	log.Warn().Msg("Things are not quite right")
	log.Logger.DebugWriter = os.Stderr
	log.Debug().Int("x", 6).Int("y", 42).Msg("Debug trace attempt 2")
	example()
}
