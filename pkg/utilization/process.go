package utilization

import (
	"os"

	"github.com/pkg/errors"
	gopsutil "github.com/shirou/gopsutil/v4/process"
)

// getAllChildProcesses returns all processes which have the current process as an ancestor.
func getAllChildProcesses() ([]*gopsutil.Process, error) {
	root, err := gopsutil.NewProcess(int32(os.Getpid()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current process")
	}
	children, err := getChildProcesses(root)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get all children for process with pid %d", root.Pid)
	}
	// If the process is not found, return an error.
	if len(children) == 0 {
		return nil, errors.New("failed to find child processes")
	}

	return children, nil
}

// getChildProcesses returns all processes which originated from the provided process.
func getChildProcesses(p *gopsutil.Process) ([]*gopsutil.Process, error) {
	var all []*gopsutil.Process
	children, err := p.Children()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get children for process with pid %d", p.Pid)
	}
	all = append(all, children...)
	for _, c := range children {
		grandChildren, err := getChildProcesses(c)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get children for process with pid %d", c.Pid)
		}
		all = append(all, grandChildren...)
	}
	return all, nil
}
