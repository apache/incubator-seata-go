package process_ctrl

import "context"

type ProcessHandler interface {
	Process(ctx context.Context, processContext ProcessContext) error
}
