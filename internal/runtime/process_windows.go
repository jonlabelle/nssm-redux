//go:build windows

package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

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
	cmd      *exec.Cmd
	done     chan Result
	job      windows.Handle
	killTree bool
	started  time.Time
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

	files, err := attachIO(cmd, service, dir)
	if err != nil {
		return nil, err
	}
	defer closeFiles(files)

	process := &Process{
		cmd:      cmd,
		done:     make(chan Result, 1),
		killTree: service.KillProcessTree,
		started:  time.Now(),
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start managed process: %w", err)
	}
	if logger != nil {
		logger.Infof("started %s for service %s", exe, service.Name)
	}

	if service.KillProcessTree {
		job, err := createKillOnCloseJob(uint32(cmd.Process.Pid))
		if err != nil && logger != nil {
			logger.Warnf("process tree tracking is disabled: %v", err)
		}
		process.job = job
	}

	go process.wait()
	return process, nil
}

// Wait returns a channel that receives a single process result.
func (p *Process) Wait() <-chan Result {
	return p.done
}

// Stop forcibly terminates the managed process.
func (p *Process) Stop() error {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	if p.job != 0 {
		if err := windows.TerminateJobObject(p.job, 1); err != nil {
			return fmt.Errorf("terminate job object: %w", err)
		}
		return nil
	}

	if err := p.cmd.Process.Kill(); err != nil && err != os.ErrProcessDone {
		return fmt.Errorf("kill process: %w", err)
	}
	return nil
}

func (p *Process) wait() {
	defer close(p.done)

	err := p.cmd.Wait()
	exitCode := 1
	if p.cmd.ProcessState != nil {
		exitCode = p.cmd.ProcessState.ExitCode()
		err = nil
	}

	if p.job != 0 {
		windows.CloseHandle(p.job)
		p.job = 0
	}

	p.done <- Result{
		ExitCode: exitCode,
		Runtime:  time.Since(p.started),
		Err:      err,
	}
}

func attachIO(cmd *exec.Cmd, service config.Service, dir string) ([]*os.File, error) {
	files := make([]*os.File, 0, 3)

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
		return nil, fmt.Errorf("resolve stdin path: %w", err)
	}
	if stdinPath != "" {
		file, err := os.Open(stdinPath)
		if err != nil {
			return nil, fmt.Errorf("open stdin path: %w", err)
		}
		cmd.Stdin = file
		files = append(files, file)
	}

	stdoutPath, err := openPath(service.StdoutPath)
	if err != nil {
		return nil, fmt.Errorf("resolve stdout path: %w", err)
	}
	stderrPath, err := openPath(service.StderrPath)
	if err != nil {
		return nil, fmt.Errorf("resolve stderr path: %w", err)
	}

	var stdoutFile *os.File
	if stdoutPath != "" {
		if err := os.MkdirAll(filepath.Dir(stdoutPath), 0o755); err != nil {
			return nil, fmt.Errorf("create stdout directory: %w", err)
		}
		file, err := os.OpenFile(stdoutPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("open stdout path: %w", err)
		}
		cmd.Stdout = file
		stdoutFile = file
		files = append(files, file)
	}

	if stderrPath != "" {
		if stdoutFile != nil && strings.EqualFold(stderrPath, stdoutPath) {
			cmd.Stderr = stdoutFile
		} else {
			if err := os.MkdirAll(filepath.Dir(stderrPath), 0o755); err != nil {
				return nil, fmt.Errorf("create stderr directory: %w", err)
			}
			file, err := os.OpenFile(stderrPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
			if err != nil {
				return nil, fmt.Errorf("open stderr path: %w", err)
			}
			cmd.Stderr = file
			files = append(files, file)
		}
	}

	return files, nil
}

func closeFiles(files []*os.File) {
	for _, file := range files {
		if file != nil {
			file.Close()
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

func createKillOnCloseJob(pid uint32) (windows.Handle, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return 0, err
	}

	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err := windows.SetInformationJobObject(
		job,
		windows.JobObjectExtendedLimitInformation,
		uintptr(unsafe.Pointer(&info)),
		uint32(unsafe.Sizeof(info)),
	); err != nil {
		windows.CloseHandle(job)
		return 0, err
	}

	processHandle, err := windows.OpenProcess(
		windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_SET_QUOTA|windows.PROCESS_TERMINATE|windows.SYNCHRONIZE,
		false,
		pid,
	)
	if err != nil {
		windows.CloseHandle(job)
		return 0, err
	}
	defer windows.CloseHandle(processHandle)

	if err := windows.AssignProcessToJobObject(job, processHandle); err != nil {
		windows.CloseHandle(job)
		return 0, err
	}

	return job, nil
}
