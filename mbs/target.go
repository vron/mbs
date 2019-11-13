package mbs

import (
	"bufio"
	"os"
	"path/filepath"

	"github.com/vron/mbs/conf"
)

type target struct {
	mark  bool // mark used to look for import cycles
	clean bool

	t *conf.Target
	i map[string]*conf.Import

	self_time float32
	priority  float32
	parents   []*target
	children  []*target

	path  string
	globs []string
}

func (t *target) String() string {
	if t.t == nil {
		return "empty_target"
	}
	return t.t.Name + "@" + t.path
}

func (b *Builder) loadMakefile(path string) error {
	// load the makefile and turn it into target structures that we need
	if !filepath.IsAbs(path) {
		panic("invariant broken")
	}
	folder := filepath.Dir(path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	reader := bufio.NewReader(f)
	m, err := conf.Parse(reader)
	if err != nil {
		return err
	}

	for nm, tg := range m.Targets {
		b.targets[targetName(path, nm)] = &target{
			t:        tg,
			i:        m.Imports,
			parents:  []*target{},
			children: []*target{},
			globs:    []string{},
			path:     folder,
		}
	}
	return nil
}
