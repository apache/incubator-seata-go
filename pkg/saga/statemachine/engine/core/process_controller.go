package core

import (
	"context"
)

type ProcessController interface {
	Process(ctx context.Context, context ProcessContext) error
}

type ProcessControllerImpl struct {
	businessProcessor BusinessProcessor
}

func (p *ProcessControllerImpl) Process(ctx context.Context, context ProcessContext) error {
	if err := p.businessProcessor.Process(ctx, context); err != nil {
		return err
	}
	if err := p.businessProcessor.Route(ctx, context); err != nil {
		return err
	}
	return nil
}

func (p *ProcessControllerImpl) BusinessProcessor() BusinessProcessor {
	return p.businessProcessor
}

func (p *ProcessControllerImpl) SetBusinessProcessor(businessProcessor BusinessProcessor) {
	p.businessProcessor = businessProcessor
}
