package logger

import "runtime/debug"

func PanicLogger() {
	if r := recover(); r != nil {
		log := SugaredLogger().With("op", "panic_logger")
		log.Fatalf("panic: %s stack: %s", r, string(debug.Stack()))
	}
}