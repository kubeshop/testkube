package localinstall

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
	gopsutilhost "github.com/shirou/gopsutil/v4/host"
)

type Status string

const (
	StatusPass Status = "pass"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

type Result struct {
	Name    string
	Status  Status
	Version string
	Detail  string
	// Secondary info, shown dimmed after Detail.
	Hint string
	Fix  string
}

const (
	gigabyte      = 1 << 30
	minCPUs       = 4
	minMemory     = 6 * gigabyte
	minFreeDisk   = 10 * gigabyte
	resourcesHint = "Testkube may run slowly or fail to start"

	// Docker reports ~3% under its setting: 6 GB shows ~5.8.
	minMemoryReported = 55 * gigabyte / 10

	// kind's known issue: lower limits crash pods at random.
	minInotifyInstances = 512
	minInotifyWatches   = 524288

	systemDockerSocket  = "/var/run/docker.sock"
	dockerPermissionFix = "Your user can't use Docker yet. Run:\n" +
		"  sudo usermod -aG docker $USER\n" +
		"  newgrp docker\n" +
		"Then run the installer again. Details: https://docs.docker.com/engine/install/linux-postinstall/"
)

// We install these ourselves, so missing is only a warning.
var installableTools = []string{"kubectl"}

// Always our pinned copies: other versions break the install.
var OwnTools = []string{"helm", "kind"}

type host interface {
	LookPath(name string) (string, error)
	ToolVersion(ctx context.Context, path string, args ...string) string
	Reachable(ctx context.Context, url string) bool
	FreeDiskAt(path string) (free uint64, visible bool, err error)
	OSVersion() string
	ReadSysctl(name string) string
}

type realHost struct{}

func (realHost) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// Each tool prints its version differently; all contain vX.Y.Z.
var semver = regexp.MustCompile(`v\d+\.\d+\.\d+`)

var versionArgs = map[string][]string{
	"kubectl": {"version", "--client"},
	"helm":    {"version", "--short"},
	"kind":    {"version"},
}

func (realHost) ToolVersion(ctx context.Context, path string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, _ := exec.CommandContext(ctx, path, args...).Output()
	return semver.FindString(string(out))
}

func (realHost) FreeDiskAt(path string) (uint64, bool, error) {
	return freeDiskAt(path)
}

// Empty off Linux: there is no /proc there.
func (realHost) ReadSysctl(name string) string {
	data, _ := os.ReadFile(filepath.Join("/proc/sys", strings.ReplaceAll(name, ".", "/")))
	return strings.TrimSpace(string(data))
}

func (realHost) OSVersion() string {
	platform, _, version, err := gopsutilhost.PlatformInformation()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(platform + " " + version)
}

type Checker struct {
	host        host
	docker      dockerClient
	info        *dockerInfo
	diskFree    *uint64
	unreachable []string
}

// Facts are the measured values behind the results, for tracking.
func (c *Checker) Facts() map[string]any {
	facts := map[string]any{"os_version": c.host.OSVersion()}
	if c.info != nil {
		if c.info.ServerVersion != "" {
			facts["docker_version"] = c.info.ServerVersion
		}
		if c.info.OperatingSystem != "" {
			facts["docker_engine"] = c.info.OperatingSystem
		}
		if len(c.info.PodmanHost) > 0 {
			facts["docker_engine"] = "Podman"
		}
		if c.info.NCPU > 0 && c.info.MemTotal > 0 {
			facts["cpus"] = c.info.NCPU
			facts["memory_gb"] = roundGB(uint64(c.info.MemTotal))
		}
	}
	if c.diskFree != nil {
		facts["disk_free_gb"] = roundGB(*c.diskFree)
	}
	if len(c.unreachable) > 0 {
		facts["network_unreachable"] = c.unreachable
	}
	return facts
}

func NewChecker() *Checker {
	return &Checker{host: realHost{}, docker: dockerCLI{}}
}

func (c *Checker) CheckTools(ctx context.Context) []Result {
	results := []Result{c.checkDocker(ctx)}
	for _, name := range installableTools {
		results = append(results, c.checkInstallable(ctx, name))
	}
	return results
}

// The user's copy never counts, only ours at the pinned version.
func (c *Checker) NeedsOwn(ctx context.Context, name string) bool {
	dir, err := ToolsDir()
	if err != nil {
		return true
	}
	return c.host.ToolVersion(ctx, filepath.Join(dir, name), versionArgs[name]...) != ToolVersion(name)
}

func (c *Checker) CheckOS() Result {
	name := strings.Replace(c.host.OSVersion(), "darwin", "macOS", 1)
	if name == "" {
		name = runtime.GOOS
	}
	return Result{Name: "os", Status: StatusPass, Detail: name + " " + runtime.GOARCH}
}

// Machine size only warns: users may continue on smaller machines.
func (c *Checker) CheckMachine(ctx context.Context) []Result {
	info, err := c.readDockerInfo(ctx)
	// Docker-compatible engines like Podman report other fields, so zeros.
	if err != nil || info.NCPU == 0 || info.MemTotal == 0 {
		return []Result{
			{Name: "cpu", Status: StatusWarn, Detail: "could not read from Docker"},
			{Name: "memory", Status: StatusWarn, Detail: "could not read from Docker"},
			{Name: "disk", Status: StatusWarn, Detail: "could not read free space"},
		}
	}
	// Docker Desktop only gets a share of the host.
	results := []Result{
		minimumResult("cpu", info.NCPU >= minCPUs, fmt.Sprintf("%d cores", info.NCPU), fmt.Sprintf("needs %d", minCPUs)),
		minimumResult("memory", info.MemTotal >= minMemoryReported, formatGB(uint64(info.MemTotal)), "needs "+formatGB(minMemory)),
	}

	if info.DockerRootDir == "" {
		return append(results, Result{Name: "disk", Status: StatusWarn, Detail: "could not read free space"})
	}
	free, visible, err := c.host.FreeDiskAt(info.DockerRootDir)
	switch {
	case err != nil:
		return append(results, Result{Name: "disk", Status: StatusWarn, Detail: "could not read free space"})
	case !visible:
		return results
	}
	c.diskFree = &free
	return append(results, minimumResult("disk", free >= minFreeDisk, formatGB(free)+" free", "needs "+formatGB(minFreeDisk)))
}

func HasFailure(results []Result) bool {
	for _, r := range results {
		if r.Status == StatusFail {
			return true
		}
	}
	return false
}

// A VM engine has its own kernel, so its own limits.
func (c *Checker) CheckInotify(ctx context.Context) []Result {
	info, err := c.readDockerInfo(ctx)
	if err != nil || info.KernelVersion == "" || info.KernelVersion != c.host.ReadSysctl("kernel.osrelease") {
		return nil
	}
	instances, err1 := strconv.Atoi(c.host.ReadSysctl("fs.inotify.max_user_instances"))
	watches, err2 := strconv.Atoi(c.host.ReadSysctl("fs.inotify.max_user_watches"))
	if err1 != nil || err2 != nil || (instances >= minInotifyInstances && watches >= minInotifyWatches) {
		return nil
	}
	return []Result{{Name: "inotify", Status: StatusWarn, Detail: "too low",
		Hint: fmt.Sprintf("needs %d instances, %d watches", minInotifyInstances, minInotifyWatches),
		Fix: fmt.Sprintf("Testkube may crash at random. Raise the limits:\nsudo sysctl fs.inotify.max_user_instances=%d fs.inotify.max_user_watches=%d",
			minInotifyInstances, minInotifyWatches)}}
}

func isWSL(kernel string) bool {
	return strings.Contains(strings.ToLower(kernel), "microsoft")
}

// Both check groups read one docker info call.
func (c *Checker) readDockerInfo(ctx context.Context) (dockerInfo, error) {
	if c.info != nil {
		return *c.info, nil
	}
	info, err := c.docker.Info(ctx)
	if err != nil {
		return dockerInfo{}, err
	}
	c.info = &info
	return info, nil
}

func (c *Checker) checkDocker(ctx context.Context) Result {
	if _, err := c.host.LookPath("docker"); err != nil {
		// Usually Docker Desktop is installed, just not shared here.
		if isWSL(c.host.ReadSysctl("kernel.osrelease")) {
			return Result{Name: "docker", Status: StatusFail, Detail: "not found",
				Fix: "In Docker Desktop, open Settings > Resources > WSL integration,\nturn on this distro, then run again."}
		}
		return Result{Name: "docker", Status: StatusFail, Detail: "not found", Fix: "Install Docker: https://docs.docker.com/get-docker/"}
	}
	if _, err := c.readDockerInfo(ctx); err != nil {
		if errors.Is(err, errDockerTimeout) {
			return Result{Name: "docker", Status: StatusFail, Detail: "not answering", Fix: "Docker did not answer within 10 seconds. It may still be starting; wait a moment, then run again"}
		}
		// The docker group only grants access to the system socket.
		if strings.Contains(err.Error(), "permission denied") && strings.Contains(err.Error(), systemDockerSocket) {
			return Result{Name: "docker", Status: StatusFail, Detail: "permission denied", Fix: dockerPermissionFix}
		}
		return Result{Name: "docker", Status: StatusFail, Detail: "not reachable", Fix: err.Error() + "\nStart Docker, then run the installer again"}
	}
	if len(c.info.PodmanHost) > 0 {
		return Result{Name: "docker", Status: StatusFail, Version: "Podman", Detail: "not supported yet",
			Fix: "Testkube needs Docker for now. Install Docker:\nhttps://docs.docker.com/get-docker/"}
	}
	version := ""
	if c.info.ServerVersion != "" {
		version = "v" + c.info.ServerVersion
	}
	return Result{Name: "docker", Status: StatusPass, Version: version, Detail: "running"}
}

func (c *Checker) checkInstallable(ctx context.Context, name string) Result {
	path, err := c.host.LookPath(name)
	if err != nil {
		return Result{Name: name, Status: StatusWarn, Detail: "not found"}
	}
	version := c.host.ToolVersion(ctx, path, versionArgs[name]...)
	// Keeps the row aligned with the others.
	if version == "" {
		version = "unknown"
	}
	return Result{Name: name, Status: StatusPass, Version: version, Detail: path}
}

// Docker Desktop's data dir lives inside its VM, invisible here.
func freeDiskAt(root string) (free uint64, visible bool, err error) {
	if _, err := os.Stat(root); err != nil {
		// Only absence means VM-hidden; other errors must still warn.
		if errors.Is(err, fs.ErrNotExist) {
			return 0, false, nil
		}
		return 0, true, err
	}
	usage, err := disk.Usage(root)
	if err != nil {
		return 0, true, err
	}
	return usage.Free, true, nil
}

func minimumResult(name string, ok bool, have, need string) Result {
	// Shown on pass too, so users see their headroom.
	if ok {
		return Result{Name: name, Status: StatusPass, Detail: have, Hint: need}
	}
	return Result{Name: name, Status: StatusWarn, Detail: have, Hint: need, Fix: resourcesHint}
}

func formatGB(bytes uint64) string {
	return fmt.Sprintf("%.1f GB", float64(bytes)/gigabyte)
}

func roundGB(bytes uint64) float64 {
	return math.Round(float64(bytes)/gigabyte*10) / 10
}
