package engine

import (
	"runtime"

	"github.com/JhnFrankz/upp/internal/adapters"
	"github.com/JhnFrankz/upp/internal/config"
)

// AdapterResolverFunc defines the signature for an adapter resolution function.
type AdapterResolverFunc func(cfg *config.Config, osName string) ([]adapters.Adapter, error)

// Engine coordinates headless tool discovery, checking, and planning.
type Engine struct {
	cfg             *config.Config
	osName          string
	numWorkers      int
	adapters        []adapters.Adapter
	adapterResolver AdapterResolverFunc
}

// Option configures an Engine instance.
type Option func(*Engine)

// CalculateWorkerCount calculates worker pool size clamped to [4, 8] based on CPU cores.
func CalculateWorkerCount(numCPU int) int {
	if numCPU < 4 {
		return 4
	}
	if numCPU > 8 {
		return 8
	}
	return numCPU
}

// WithConcurrency overrides the default worker pool size. Values < 1 are clamped to 1.
func WithConcurrency(workers int) Option {
	return func(e *Engine) {
		if workers < 1 {
			workers = 1
		}
		e.numWorkers = workers
	}
}

// WithWorkerCount is an alias for WithConcurrency.
func WithWorkerCount(workers int) Option {
	return WithConcurrency(workers)
}

// WithAdapters overrides adapter discovery with a pre-configured adapter slice.
func WithAdapters(adapters []adapters.Adapter) Option {
	return func(e *Engine) {
		e.adapters = adapters
	}
}

// WithAdapterResolver overrides adapter discovery with a custom resolver function.
func WithAdapterResolver(resolver AdapterResolverFunc) Option {
	return func(e *Engine) {
		e.adapterResolver = resolver
	}
}

// New creates and initializes an Engine instance.
func New(cfg *config.Config, osName string, opts ...Option) *Engine {
	e := &Engine{
		cfg:        cfg,
		osName:     osName,
		numWorkers: CalculateWorkerCount(runtime.NumCPU()),
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}
