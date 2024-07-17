package orchestration

import (
	errors2 "errors"

	"github.com/pkg/errors"
	gopsutil "github.com/shirou/gopsutil/v3/process"
)

type processNode struct {
	pid   int32
	nodes map[*processNode]struct{}
}

func (p *processNode) Find(pid int32) []*processNode {
	if p.pid == pid {
		return []*processNode{p}
	}
	// Try to find directly
	for n := range p.nodes {
		if n.pid == pid {
			return append([]*processNode{p}, n)
		}
	}

	// Try to find in the children
	for n := range p.nodes {
		found := n.Find(pid)
		if found != nil {
			return append([]*processNode{p}, found...)
		}
	}

	return nil
}

func (p *processNode) VirtualizePath(pid int32) {
	path := p.Find(pid)
	if path == nil {
		return
	}

	// Cannot virtualize itself
	if len(path) == 1 {
		return
	}

	// Virtualize recursively
	for i := 1; i < len(path); i++ {
		delete(path[0].nodes, path[i])
		for node := range path[i].nodes {
			path[0].nodes[node] = struct{}{}
		}
	}
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
		return errors.Wrap(errors2.Join(errs...), "suspending processes")
	}
	return errors.Wrapf(errors2.Join(errs...), "suspending process %d", p.pid)
}

// Kill all the processes in group, starting from top
func (p *processNode) Kill() error {
	errs := make([]error, 0)
	if p.pid != -1 {
		return errors.Wrap((&gopsutil.Process{Pid: p.pid}).Kill(), "killing processes")
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
		r[p.Pid] = &processNode{pid: p.Pid}
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
