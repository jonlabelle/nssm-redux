//go:build windows

package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jonlabelle/nssm-redux/internal/config"
	"github.com/jonlabelle/nssm-redux/internal/support"
	"golang.org/x/sys/windows"
)

// Logger is the logging interface used by the runtime package.
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// Result describes a completed process run.
type Result struct {
	ExitCode int
	Runtime  time.Duration
	Err      error
}

// Process manages a single child process instance.
type Process struct {
	cmd     *exec.Cmd
	done    chan Result
	job     windows.Handle
	jobMu   sync.Mutex
	logs    *logManager
	service config.Service
	started time.Time
}

// Start launches the configured child process.
func Start(service config.Service, logger Logger) (*Process, error) {
	service.Normalize()
	if err := service.Validate(); err != nil {
		return nil, err
	}

	exe, err := expandWindowsString(service.Executable)
	if err != nil {
		return nil, fmt.Errorf("expand application path: %w", err)
	}

	dir, err := expandWindowsString(service.Directory)
	if err != nil {
		return nil, fmt.Errorf("expand startup directory: %w", err)
	}
	if dir == "" {
		dir = filepath.Dir(exe)
	}

	cmd := exec.Command(exe)
	cmd.Dir = dir
	cmd.Env = support.MergeEnvironment(os.Environ(), service.Environment, service.EnvironmentExtra)

	flags := uint32(windows.CREATE_UNICODE_ENVIRONMENT | windows.CREATE_NEW_PROCESS_GROUP)
	if service.NoConsole {
		flags |= windows.CREATE_NO_WINDOW
	} else {
		flags |= windows.CREATE_NEW_CONSOLE
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: flags,
		CmdLine:       buildCommandLine(exe, service.Arguments),
	}

	files, logs, err := attachIO(cmd, service, dir)
	if err != nil {
		return nil, err
	}
	defer closeFiles(files)

	process := &Process{
		cmd:     cmd,
		done:    make(chan Result, 1),
		logs:    logs,
		service: service.Clone(),
		started: time.Now(),
	}

	if err := cmd.Start(); err != nil {
		if logs != nil {
			_ = logs.Wait()
		}
		return nil, fmt.Errorf("start managed process: %w", err)
	}
	if logger != nil {
		logger.Infof("started %s for service %s", exe, service.Name)
	}

	job, err := configureManagedProcess(uint32(cmd.Process.Pid), service)
	process.job = job
	if err != nil {
		if logger != nil {
			logger.Warnf("managed process runtime settings are only partially applied: %v", err)
		}
	}

	go process.wait()
	return process, nil
}

// Wait returns a channel that receives a single process result.
func (p *Process) Wait() <-chan Result {
	return p.done
}

// PID returns the managed process ID, or 0 when unavailable.
func (p *Process) PID() int {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

// StartedAt reports when the managed process was launched.
func (p *Process) StartedAt() time.Time {
	if p == nil {
		return time.Time{}
	}
	return p.started
}

func (p *Process) wait() {
	defer close(p.done)

	err := p.cmd.Wait()
	exitCode := 1
	if p.cmd.ProcessState != nil {
		exitCode = p.cmd.ProcessState.ExitCode()
		err = nil
	}

	if p.service.KillProcessTree && p.job != 0 {
		_ = p.terminateJob(uint32(exitCode))
	}
	p.closeJob()
	if p.logs != nil {
		err = joinRuntimeError(err, p.logs.Wait())
	}

	p.done <- Result{
		ExitCode: exitCode,
		Runtime:  time.Since(p.started),
		Err:      err,
	}
}

func attachIO(cmd *exec.Cmd, service config.Service, dir string) ([]*os.File, *logManager, error) {
	files := make([]*os.File, 0, 3)
	logs := newLogManager()

	openPath := func(path string) (string, error) {
		if path == "" {
			return "", nil
		}
		expanded, err := expandWindowsString(path)
		if err != nil {
			return "", err
		}
		if filepath.IsAbs(expanded) {
			return expanded, nil
		}
		return filepath.Join(dir, expanded), nil
	}

	stdinPath, err := openPath(service.StdinPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve stdin path: %w", err)
	}
	if stdinPath != "" {
		file, err := os.Open(stdinPath)
		if err != nil {
			return nil, nil, fmt.Errorf("open stdin path: %w", err)
		}
		cmd.Stdin = file
		files = append(files, file)
	}

	stdoutPath, err := openPath(service.StdoutPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve stdout path: %w", err)
	}
	stderrPath, err := openPath(service.StderrPath)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve stderr path: %w", err)
	}

	prepareOutput := func(path string) (*os.File, error) {
		if path == "" {
			return nil, nil
		}
		if sink := logs.Sink(path); sink != nil {
			return sink.pipeWriter, nil
		}

		usePipeline := service.Logging.TimestampLog || (service.Logging.Enabled && service.Logging.Online)
		if service.Logging.Enabled && !usePipeline {
			if err := rotateExistingFile(path, service.Logging); err != nil {
				return nil, err
			}
		}

		if usePipeline {
			reader, writer, err := os.Pipe()
			if err != nil {
				return nil, err
			}
			sink := newLogSink(service.Name, path, reader, service.Logging)
			sink.pipeWriter = writer
			logs.AddSink(path, sink)
			files = append(files, writer)
			return writer, nil
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
		return file, nil
	}

	var stdoutFile *os.File
	if stdoutPath != "" {
		file, err := prepareOutput(stdoutPath)
		if err != nil {
			return nil, nil, fmt.Errorf("open stdout path: %w", err)
		}
		cmd.Stdout = file
		stdoutFile = file
	}

	if stderrPath != "" {
		if stdoutFile != nil && strings.EqualFold(stderrPath, stdoutPath) {
			cmd.Stderr = stdoutFile
		} else {
			file, err := prepareOutput(stderrPath)
			if err != nil {
				return nil, nil, fmt.Errorf("open stderr path: %w", err)
			}
			cmd.Stderr = file
		}
	}

	if len(logs.order) == 0 {
		return files, nil, nil
	}
	logs.Start()
	return files, logs, nil
}

func closeFiles(files []*os.File) {
	for _, file := range files {
		if file != nil {
			_ = file.Close()
		}
	}
}

func buildCommandLine(executable, arguments string) string {
	commandLine := support.JoinCommandLine([]string{executable})
	if strings.TrimSpace(arguments) == "" {
		return commandLine
	}
	return commandLine + " " + arguments
}

func expandWindowsString(value string) (string, error) {
	if value == "" {
		return "", nil
	}

	ptr, err := windows.UTF16PtrFromString(value)
	if err != nil {
		return "", err
	}

	buffer := make([]uint16, 32768)
	n, err := windows.ExpandEnvironmentStrings(ptr, &buffer[0], uint32(len(buffer)))
	if err != nil {
		return "", err
	}
	if n == 0 || n > uint32(len(buffer)) {
		return "", fmt.Errorf("expanded string is too long")
	}

	return windows.UTF16ToString(buffer[:n]), nil
}

func configureManagedProcess(pid uint32, service config.Service) (windows.Handle, error) {
	handle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_SET_INFORMATION|windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.SYNCHRONIZE,
		false,
		pid,
	)
	if err != nil {
		return 0, fmt.Errorf("open managed process: %w", err)
	}
	defer func() { _ = windows.CloseHandle(handle) }()

	var firstErr error
	if service.Priority != "" && service.Priority != config.PriorityNormal {
		if err := windows.SetPriorityClass(handle, service.Priority.WindowsValue()); err != nil {
			firstErr = joinRuntimeError(firstErr, fmt.Errorf("set priority class: %w", err))
		}
	}
	if service.Affinity != 0 {
		if err := applyAffinity(handle, service.Affinity); err != nil {
			firstErr = joinRuntimeError(firstErr, err)
		}
	}
	if !service.KillProcessTree {
		return 0, firstErr
	}

	job, err := createProcessJob(handle)
	if err != nil {
		firstErr = joinRuntimeError(firstErr, fmt.Errorf("enable process tree tracking: %w", err))
		return 0, firstErr
	}

	return job, firstErr
}

func createProcessJob(processHandle windows.Handle) (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}

	if err := windows.AssignProcessToJobObject(job, processHandle); err != nil {
		_ = windows.CloseHandle(job)
		return 0, err
	}

	return job, nil
}

func applyAffinity(processHandle windows.Handle, mask config.AffinityMask) error {
	target := uintptr(mask)
	_, systemAffinity, err := getProcessAffinityMask(processHandle)
	if err == nil && systemAffinity != 0 {
		target &= systemAffinity
	}
	if target == 0 {
		return fmt.Errorf("set affinity mask: configured CPUs are unavailable")
	}
	if err := setProcessAffinityMask(processHandle, target); err != nil {
		return fmt.Errorf("set affinity mask: %w", err)
	}
	return nil
}

func (p *Process) terminateJob(exitCode uint32) error {
	p.jobMu.Lock()
	defer p.jobMu.Unlock()

	if p.job == 0 {
		return nil
	}
	if err := windows.TerminateJobObject(p.job, exitCode); err != nil {
		return err
	}
	return nil
}

func (p *Process) closeJob() {
	p.jobMu.Lock()
	defer p.jobMu.Unlock()

	if p.job == 0 {
		return
	}
	_ = windows.CloseHandle(p.job)
	p.job = 0
}

func joinRuntimeError(current, next error) error {
	if current == nil {
		return next
	}
	if next == nil {
		return current
	}
	return fmt.Errorf("%v; %w", current, next)
}
