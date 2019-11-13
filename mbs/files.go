package mbs

import (
	"context"
	"path/filepath"

	"github.com/vron/mbs/stat"
)

func (b *Builder) checkFiles(ctx context.Context, dag *target) error {
	// walk the DAG to check all files to the cache to
	// set them as clean and dirty etc.

	// note that a non-existing file is not an error.

	// TOOD: Many go-routines to not block on each stat..
	clean := true
	for _, c := range dag.children {
		err := b.checkFiles(ctx, c)
		if err != nil {
			return err
		}
		if !c.clean {
			clean = false
		}
	}

	for _, g := range dag.globs {
		hash, err := stat.New(false).Stat(dag.path, g)
		if err != nil {
			return err
		}
		changed := b.cache.Set(filepath.Join(dag.path, g), hash)
		if changed {
			clean = false
		}
	}
	dag.clean = clean

	return nil
}
