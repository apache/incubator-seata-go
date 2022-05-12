package tcc

import (
	"context"
)

var (
	tccRunnerSingleton = &TccRunner{}
)

type TccRunner struct {
}

func (c *TccRunner) run(ctx context.Context, tccService TCCService, params []interface{}) {
	// step1: register rm service

	// step2: open global transaction

	// step3: do prepare Method

	// step4: commit prepare transaction
}
