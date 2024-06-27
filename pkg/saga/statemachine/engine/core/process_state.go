package core

import (
	"context"
	"errors"
	"sync"
)

type StateHandler interface {
	State() string
	ProcessHandler
}

type InterceptAbleStateHandler interface {
	StateHandler
	StateHandlerInterceptorList() []StateHandlerInterceptor
	RegistryStateHandlerInterceptor(stateHandlerInterceptor StateHandlerInterceptor)
}

type ProcessHandler interface {
	Process(ctx context.Context, processContext ProcessContext) error
}

type StateHandlerInterceptor interface {
	PreProcess(ctx context.Context, processContext ProcessContext) error
	PostProcess(ctx context.Context, processContext ProcessContext) error
	Match(stateType string) bool
}

type StateMachineProcessHandler struct {
	mp map[string]StateHandler
	mu sync.RWMutex
}

func NewStateMachineProcessHandler() *StateMachineProcessHandler {
	return &StateMachineProcessHandler{
		mp: make(map[string]StateHandler),
	}
}

func (s *StateMachineProcessHandler) Process(ctx context.Context, processContext ProcessContext) error {
	stateInstruction, _ := processContext.GetInstruction().(StateInstruction)

	state, err := stateInstruction.GetState(processContext)
	if err != nil {
		return err
	}

	stateType := state.Type()
	stateHandler := s.GetStateHandler(stateType)
	if stateHandler == nil {
		return errors.New("Not support [" + stateType + "] state handler")
	}

	interceptAbleStateHandler, ok := stateHandler.(InterceptAbleStateHandler)

	var stateHandlerInterceptorList []StateHandlerInterceptor
	if ok {
		stateHandlerInterceptorList = interceptAbleStateHandler.StateHandlerInterceptorList()
	}

	if stateHandlerInterceptorList != nil && len(stateHandlerInterceptorList) > 0 {
		for _, stateHandlerInterceptor := range stateHandlerInterceptorList {
			err = stateHandlerInterceptor.PreProcess(ctx, processContext)
			if err != nil {
				return err
			}
		}
	}

	err = stateHandler.Process(ctx, processContext)
	if err != nil {
		return err
	}

	if stateHandlerInterceptorList != nil && len(stateHandlerInterceptorList) > 0 {
		for _, stateHandlerInterceptor := range stateHandlerInterceptorList {
			err = stateHandlerInterceptor.PostProcess(ctx, processContext)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *StateMachineProcessHandler) GetStateHandler(stateType string) StateHandler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mp[stateType]
}

func (s *StateMachineProcessHandler) RegistryStateHandler(stateType string, stateHandler StateHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mp == nil {
		s.mp = make(map[string]StateHandler)
	}
	s.mp[stateType] = stateHandler
}
