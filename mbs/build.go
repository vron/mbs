package mbs

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"time"

	"github.com/vron/mbs/cache"
)

// Options configure how the builder should operate
type Options struct {
	LogCommands bool
	LogOutput   bool

	// If nil os.Stdout will be used
	Stdout io.Writer
	// If nil os.Stderr will be used
	Stderr io.Writer
}

type Measures struct {
	TimeGraph time.Duration
}

type Builder struct {
	Options

	cache *cache.Cache

	targets map[string]*target
}

func NewBuilder(c *cache.Cache, o Options) *Builder {
	b := &Builder{
		Options: o,
		targets: make(map[string]*target, 100),
		cache:   c,
	}
	return b
}

func (b *Builder) Build(ctx context.Context, makefile string, targets []string) error {
	makefile, err := filepath.Abs(makefile)
	if err != nil {
		return errors.New("error reading makefile: " + err.Error())
	}

	dag, err := b.buildDAG(ctx, makefile, targets)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	err = b.checkFiles(ctx, dag)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	err = b.doRun(ctx, dag)
	if err != nil {
		return err
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}

	return nil
}
