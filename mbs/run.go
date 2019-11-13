package mbs

import (
	"bytes"
	"context"
	"os/exec"
)

// TODO: Set the priority so stuff with many dependicies are started first

type runner struct {
	b                   *Builder
	ctx                 context.Context
	cancel              context.CancelFunc
	maxWorkers, workers int
	queue               *queue
	result              chan runResult
}

func (r *runner) startMax() {
	for r.workers < r.maxWorkers {
		t := r.queue.Pop()
		if t == nil {
			return
		}
		r.workers++

		go r.runTarget(r.ctx, t, r.result)
	}
}

func (b *Builder) doRun(ctx context.Context, dag *target) error {
	// walk down - get the ones that are not clean and start building.
	r := &runner{
		b:          b,
		maxWorkers: 4,
		queue:      newQueue(),
		result:     make(chan runResult, 100),
	}
	r.ctx, r.cancel = context.WithCancel(ctx)
	r.findStart(dag, r.queue)

	var firstErr error

	r.startMax()
	if r.workers == 0 {
		return nil // there was no work to be done
	}

loop:
	for {
		select {
		case <-r.ctx.Done():
			panic("unimplemented cancelation handling")
		case res := <-r.result:
			if res.err != nil {
				if firstErr == nil {
					firstErr = res.err
				}
				// TODO: Dump the output of the command that failed.
				r.cancel()
			} else if res.done {
				// so one target was completely done, that means that we should
				// check if this enables any new stuff to be added to the priority
				// queue and subsequently run.
				r.workers--

				res.t.clean = true
				for _, p := range res.t.parents {
					if p.t == nil {
						continue // this is the wrapper node that needs no building
					}
					dirty := false
					for _, c := range p.children {
						if !c.clean {
							dirty = true
						}
					}
					if !dirty {
						r.queue.Insert(p)
					}
				}
				r.startMax()

				if r.workers == 0 {
					break loop
				}
			}
		}
	}

	// sanity check so all is build
	if firstErr == nil {
		for _, c := range dag.children {
			if !c.clean {
				panic("invariant broken since not all children clean...")
			}
		}
	}

	return firstErr
}

func (r *runner) findStart(dag *target, q *queue) {
	// There is no need to walk down into those that are clean, we know that
	// a target i sonly clean if all children are clean..
	if dag.clean {
		return
	}

	doRun := true
	for _, c := range dag.children {
		if !c.clean {
			doRun = false
			r.findStart(c, q)
		}
	}
	if doRun {
		q.Insert(dag)
	}
}

type runResult struct {
	t      *target
	stdout []byte
	stderr []byte

	done bool
	code int
	err  error
}

func (rr *runner) runTarget(ctx context.Context, t *target, ch chan runResult) {
	// All commands in a target are run sequentially
	// TODO: Introduce flag if we should keep the stdout/err or not, depending on
	// command they could becode expensive? (or only log those with bad exit signals?)
	for i, c := range t.t.Cmds {
		cmd := exec.CommandContext(ctx, "bash", "-c", c.Cmd)
		stdout := bytes.NewBuffer(nil)
		stderr := bytes.NewBuffer(nil)
		cmd.Stdout = stdout
		cmd.Stderr = stderr
		err := cmd.Run()
		r := runResult{
			t:      t,
			stdout: stdout.Bytes(),
			stderr: stderr.Bytes(),
			err:    err,
		}
		if i >= len(t.t.Cmds)-1 {
			r.done = true // the last one, so this command is done, signal that.
		}
		if e, ok := err.(*exec.ExitError); ok {
			r.code = e.ExitCode()
			if r.code != 0 {
				// we abort as soon as we get a non - zero code...
				r.done = true
				ch <- r
				return
			}
		}

		rr.b.logCommandOutput(stdout.Bytes())

		ch <- r
	}
}
