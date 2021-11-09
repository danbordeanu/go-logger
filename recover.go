package logger

import "runtime/debug"

// PanicLogger will pass the error which caused the go routine to panic and the
// stack trace onto the current SugaredLogger as a Fatal message and add the
// field "op" with value of "panic_logger". Use it to log panics as structured
// logs. Remember that you must defer this call at the beginning of each
// goroutine!
//
// Example
//
// 	defer logger.PanicLogger()
func PanicLogger() {
	if r := recover(); r != nil {
		log := SugaredLogger().With("op", "panic_logger")
		log.Fatalf("panic: %s stack: %s", r, string(debug.Stack()))
	}
}