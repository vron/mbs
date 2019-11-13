// Command mbs replaces make with the functionality I need
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/shibukawa/configdir"
	"github.com/vron/mbs/cache"
	"github.com/vron/mbs/mbs"
)

var (
	fVerbose     bool
	fVeryVerbose bool
	fHelp        bool
	fClearCache  bool
	fMakefile    string
)

func init() {
	flag.BoolVar(&fVerbose, "v", false, "verbose logging")
	flag.BoolVar(&fVeryVerbose, "vv", false, "very verbose logging")
	flag.BoolVar(&fHelp, "h", false, "show help information")
	flag.BoolVar(&fClearCache, "clear-cache", false, "completely wipe the cache")
	flag.StringVar(&fMakefile, "i", "Makefile.mbs", "conf file from which to read configuration")
}

func main() {
	targets, options := handleArgs()
	cache := loadCache()

	b := mbs.NewBuilder(cache, options)

	doBuild(b, fMakefile, targets)
}

func doBuild(b *mbs.Builder, makefile string, targets []string) {
	ctx, cf := context.WithCancel(context.Background())
	stopped := make(chan struct{}, 0)

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cf()
		stopped <- struct{}{}
	}()

	if err := b.Build(ctx, makefile, targets); err != nil {
		select {
		case <-stopped:
			// we were interupted so simply quit
			fmt.Fprintln(os.Stderr, "interrupted")
			os.Exit(0)
		default:
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func handleArgs() (targets []string, options mbs.Options) {
	flag.Parse()
	if fHelp {
		showHelp()
		os.Exit(1)
	}

	if fClearCache {
		clearCache()
		os.Exit(0)
	}

	targets = flag.Args()
	options = createOptions()
	return
}

func createOptions() (o mbs.Options) {
	if fVerbose || fVeryVerbose {
		o.LogCommands = true
	}
	if fVeryVerbose {
		o.LogOutput = true
	}

	return
}

func getCachePath() string {
	configDirs := configdir.New("vron", "mbs")
	cf := configDirs.QueryCacheFolder()
	return filepath.Join(cf.Path, "cache.db")
}

func loadCache() *cache.Cache {
	c, err := cache.Open(getCachePath())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return c
}

func clearCache() {
	err := os.Remove(getCachePath())
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func showHelp() {
	flag.PrintDefaults()
}
