package orchestration

import (
	errors2 "errors"

	"github.com/pkg/errors"
	gopsutil "github.com/shirou/gopsutil/v4/process"
)

type processNode struct {
	pid   int32
	nodes map[*processNode]struct{}
}

func (p *processNode) find(pid int32) *processNode {
	if p.pid == pid {
		return p
	}
	for n := range p.nodes {
		if found := n.find(pid); found != nil {
			return found
		}
	}
	return nil
}

// childrenOf returns a virtual root (pid -1) holding the descendant subtree of
// the process with the given pid, or an empty root when it isn't found. Killing,
// suspending or resuming it affects only that process's own children, never the
// process itself or anything outside its subtree. When the init runs as PID 1 in
// a pod this is the full step process tree (matching the previous whole-tree
// behavior); on a shared host such as CI it stays scoped, so concurrent test
// binaries can't reach into each other's trees.
func (p *processNode) childrenOf(pid int32) *processNode {
	virtual := &processNode{pid: -1, nodes: map[*processNode]struct{}{}}
	if target := p.find(pid); target != nil {
		for child := range target.nodes {
			virtual.nodes[child] = struct{}{}
		}
	}
	return virtual
}

// Suspend all the processes in group, starting from top
func (p *processNode) Suspend() error {
	errs := make([]error, 0)
	if p.pid != -1 {
		err := (&gopsutil.Process{Pid: p.pid}).Suspend()
		if err != nil {
			errs = append(errs, err)
		}
	}
	for node := range p.nodes {
		err := node.Suspend()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	if p.pid == -1 {
		return errors.Wrap(errors2.Join(errs...), "suspending processes")
	}
	return errors.Wrapf(errors2.Join(errs...), "suspending process %d", p.pid)
}

// Resume all the processes in group, starting from bottom
func (p *processNode) Resume() error {
	errs := make([]error, 0)
	for node := range p.nodes {
		err := node.Resume()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if p.pid != -1 {
		err := (&gopsutil.Process{Pid: p.pid}).Resume()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	if p.pid == -1 {
		return errors.Wrap(errors2.Join(errs...), "resuming processes")
	}
	return errors.Wrapf(errors2.Join(errs...), "resuming process %d", p.pid)
}

// Kill all the processes in group, starting from top
func (p *processNode) Kill() error {
	errs := make([]error, 0)
	if p.pid != -1 {
		err := errors.Wrap((&gopsutil.Process{Pid: p.pid}).Kill(), "killing processes")
		if err != nil {
			return err
		}
	}
	for node := range p.nodes {
		err := node.Kill()
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.Wrapf(errors2.Join(errs...), "killing process %d", p.pid)
}

func processes() (*processNode, bool, error) {
	// Get list of processes
	list, err := gopsutil.Processes()
	if err != nil {
		return nil, true, errors.Wrapf(err, "failed to list processes")
	}

	// Put all the processes in the map
	detached := map[int32]struct{}{}
	r := map[int32]*processNode{}
	var errs []error
	for _, p := range list {
		detached[p.Pid] = struct{}{}
		r[p.Pid] = &processNode{pid: p.Pid, nodes: map[*processNode]struct{}{}}
	}

	// Create tree of processes
	for _, p := range list {
		ppid, err := p.Ppid()
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if r[ppid] == nil || ppid == p.Pid {
			continue
		}
		r[ppid].nodes[r[p.Pid]] = struct{}{}
		delete(detached, p.Pid)
	}

	// Make virtual root of detached processes
	root := &processNode{pid: -1, nodes: make(map[*processNode]struct{}, len(detached))}
	for pid := range detached {
		root.nodes[r[pid]] = struct{}{}
	}

	// Return info
	if len(errs) > 0 {
		err = errors.Wrapf(errs[0], "failed to load %d/%d processes", len(errs), len(r))
	}
	return root, len(errs) == len(r), err
}
