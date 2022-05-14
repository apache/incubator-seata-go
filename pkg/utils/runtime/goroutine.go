package runtime

import (
	"fmt"
	"os"
	"runtime/debug"
	"time"
)

var debugIgnoreStdout = false

// GoWithRecover wraps a `go func()` with recover()
func GoWithRecover(handler func(), recoverHandler func(r interface{})) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// TODO: log
				if !debugIgnoreStdout {
					fmt.Fprintf(os.Stderr, "%s goroutine panic: %v\n%s\n",
						time.Now(), r, string(debug.Stack()))
				}
				if recoverHandler != nil {
					go func() {
						defer func() {
							if p := recover(); p != nil {
								if !debugIgnoreStdout {
									fmt.Fprintf(os.Stderr, "recover goroutine panic:%v\n%s\n", p, string(debug.Stack()))
								}
							}
						}()
						recoverHandler(r)
					}()
				}
			}
		}()
		handler()
	}()
}
