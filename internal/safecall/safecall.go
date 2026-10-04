// Package safecall converts panics into errors at subsystem boundaries so that
// a bug in a parser or renderer cannot crash the process.
package safecall

import "fmt"

// Call runs function and converts any panic into an error.
func Call(function func()) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("safecall: recovered panic: %v", recovered)
		}
	}()
	function()
	return nil
}

// Value runs function and converts any panic into an error, returning the
// function's value on success.
func Value[T any](function func() T) (value T, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			var zero T
			value = zero
			err = fmt.Errorf("safecall: recovered panic: %v", recovered)
		}
	}()
	return function(), nil
}
