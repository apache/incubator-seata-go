package process_ctrl

import (
	"context"
	"github.com/pkg/errors"
	"github.com/seata/seata-go/pkg/saga/statemachine/engine"
	"sync"
)

type BusinessProcessor interface {
	Process(ctx context.Context, processContext ProcessContext) error

	Route(ctx context.Context, processContext ProcessContext) error
}

type DefaultBusinessProcessor struct {
	processHandlers map[string]ProcessHandler
	routerHandlers  map[string]RouterHandler
	mu              sync.RWMutex
}

func (d *DefaultBusinessProcessor) RegistryProcessHandler(processType ProcessType, processHandler ProcessHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.processHandlers[string(processType)] = processHandler
}

func (d *DefaultBusinessProcessor) RegistryRouterHandler(processType ProcessType, routerHandler RouterHandler) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.routerHandlers[string(processType)] = routerHandler
}

func (d *DefaultBusinessProcessor) Process(ctx context.Context, processContext ProcessContext) error {
	processType := d.matchProcessType(processContext)

	processHandler, err := d.getProcessHandler(processType)
	if err != nil {
		return err
	}

	return processHandler.Process(ctx, processContext)
}

func (d *DefaultBusinessProcessor) Route(ctx context.Context, processContext ProcessContext) error {
	processType := d.matchProcessType(processContext)

	routerHandler, err := d.getRouterHandler(processType)
	if err != nil {
		return err
	}

	return routerHandler.Route(ctx, processContext)
}

func (d *DefaultBusinessProcessor) getProcessHandler(processType ProcessType) (ProcessHandler, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	processHandler, ok := d.processHandlers[string(processType)]
	if !ok {
		return nil, errors.New("Cannot find process handler by type " + string(processType))
	}
	return processHandler, nil
}

func (d *DefaultBusinessProcessor) getRouterHandler(processType ProcessType) (RouterHandler, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	routerHandler, ok := d.routerHandlers[string(processType)]
	if !ok {
		return nil, errors.New("Cannot find router handler by type " + string(processType))
	}
	return routerHandler, nil
}

func (d *DefaultBusinessProcessor) matchProcessType(processContext ProcessContext) ProcessType {
	ok := processContext.HasVariable(engine.VarNameProcessType)
	if ok {
		return processContext.GetVariable(engine.VarNameProcessType).(ProcessType)
	}
	return StateLang
}

type ProcessHandler interface {
	Process(ctx context.Context, processContext ProcessContext) error
}

type RouterHandler interface {
	Route(ctx context.Context, processContext ProcessContext) error
}
