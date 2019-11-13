package mbs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/juju/errgo"
)

func (b *Builder) visitTarget(makefile string, target string) (*target, error) {
	if b.targets[targetName(makefile, target)] == nil {
		err := b.loadMakefile(makefile)
		if err != nil {
			return nil, err
		}
	}
	if b.targets[targetName(makefile, target)] == nil {
		panic("no such target in file" + targetName(makefile, target))
	}
	if target == "" {
		// if no target is specified we use the first one
		panic("not implemented yet")
	}

	tgt := b.targets[targetName(makefile, target)]
	if tgt.mark {
		panic("import cycle among targets")
	}
	tgt.mark = true
	for _, d := range tgt.t.Deps {
		// TOOD: Should we accumulate time here also?
		if d.Import != "" {
			impp := tgt.i[d.Import].Path
			path := resolveImport(makefile, impp)
			dt, err := b.visitTarget(path, d.Target)
			if err != nil {
				return nil, errgo.Mask(err)
			}
			tgt.children = append(tgt.children, dt)
			dt.parents = append(dt.parents, tgt)
		} else if d.Target != "" {
			dt, err := b.visitTarget(makefile, d.Target)
			if err != nil {
				return nil, errgo.Mask(err)
			}
			tgt.children = append(tgt.children, dt)
			dt.parents = append(dt.parents, tgt)
		} else if d.Filename != "" {
			tgt.globs = append(tgt.globs, d.Filename)
		}
	}

	tgt.mark = false
	return tgt, nil
}

func (b *Builder) statFiles(files string) (clean bool) {
	// was here, use the stat package
	return
}

// let the parallelism be that we launch one go-routine for each makefile, that
// makes sense to reduce block on file-reading, but assuming that the actual processing
// per target is small.

// buildDAG builds the DAG of targets in the makefile, from that extracting
// the files that needs statting is a seperate step.
func (b *Builder) buildDAG(ctx context.Context, makefile string, targets []string) (dag *target, err error) {

	// Create a phony wrapper that wraps all the targets:
	dag = &target{
		parents:  []*target{},
		children: []*target{},
		globs:    []string{},
	}

	for _, tgt := range targets {
		t, err := b.visitTarget(filepath.Clean(makefile), tgt)
		if err != nil {
			return nil, err
		}
		dag.children = append(dag.children, t)
		t.parents = append(t.parents, dag)
	}
	return dag, nil
}

func targetName(makefile, target string) string {
	return makefile + ":" + target
}

func resolveImport(makefile, path string) string {
	mp := filepath.Dir(makefile)
	path = filepath.Join(mp, path)
	if !isDir(path) {
		return path
	}
	return filepath.Join(path, "Makefile")
}

func isDir(path string) bool {
	id, err := os.Stat(path)
	if err != nil {
		return false
	}
	return id.IsDir()
}
