package scope

import (
	"context"
	"log"
)

type options struct {
	ctx          context.Context
	errorHandler func(error)
}

func defaultOptions() options {
	return options{
		ctx:          context.Background(),
		errorHandler: func(err error) { log.Fatal(err) },
	}
}

// Option represents an option which can be used to configure
// a scope.
type Option func(*options)

// WithContext defines the base context, which will be used by the
// scope to derive its context.
func WithContext(ctx context.Context) Option {
	return func(o *options) {
		if ctx == nil {
			panic("scope options: no context specified")
		}
		o.ctx = ctx
	}
}

// WithErrorHandler defines an error handler, which will be called
// in case of an error while running functions. The default behaviour
// calls log.Fatal.
func WithErrorHandler(f func(error)) Option {
	return func(o *options) {
		if f == nil {
			panic("scope options: no error handler specified")
		}
		o.errorHandler = f
	}
}
