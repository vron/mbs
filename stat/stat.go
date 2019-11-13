package stat

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"sort"

	"github.com/bmatcuk/doublestar"
)

type Stater struct {
	checkContent bool
}

func New(checkContent bool) *Stater {
	s := &Stater{
		checkContent: checkContent,
	}
	return s
}

// Stat creates a hash of all the files given by expr (using root), either
// by using contents of files or only the mod-time.
// TODO: include a ref to cache so we can write for each individual file
func (s *Stater) Stat(root string, expr string) (hash []byte, err error) {
	if !filepath.IsAbs(expr) {
		expr = filepath.Join(root, expr)
	}

	files, err := doublestar.Glob(expr)
	if err != nil {
		return nil, err
	}
	h := sha256.New224()

	// We must be carefull with the ordering when hashing
	sort.Strings(files)
	for _, f := range files {
		// TODO: Stat or LStat - should also be configurable?
		fi, err := os.Stat(f) // Todo - really should be merged with the recursive directory handling
		if err != nil {
			return nil, err
		}
		binary.Write(h, binary.BigEndian, fi.Name())
		binary.Write(h, binary.BigEndian, fi.Size())
		binary.Write(h, binary.BigEndian, fi.ModTime().UnixNano())
	}

	return h.Sum(nil), nil
}
