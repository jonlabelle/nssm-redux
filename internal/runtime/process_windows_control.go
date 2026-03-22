//go:build windows

package runtime

import (
	"fmt"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/jonlabelle/nssm-redux/internal/config"
	"golang.org/x/sys/windows"
)

const (
	ctrlCEvent           = 0
	wmClose              = 0x0010
	wmQuit               = 0x0012
	wmEndSession         = 0x0016
	endSessionCloseApp   = 0x00000001
	endSessionCritical   = 0x40000000
	endSessionLogoff     = 0x80000000
	stillActiveExitCode  = 259
	defaultTerminateCode = 1
)

var (
	kernel32Proc               = windows.NewLazySystemDLL("kernel32.dll")
	user32Proc                 = windows.NewLazySystemDLL("user32.dll")
	procAttachConsole          = kernel32Proc.NewProc("AttachConsole")
	procFreeConsole            = kernel32Proc.NewProc("FreeConsole")
	procSetConsoleCtrlHandler  = kernel32Proc.NewProc("SetConsoleCtrlHandler")
	procGetProcessAffinityMask = kernel32Proc.NewProc("GetProcessAffinityMask")
	procSetProcessAffinityMask = kernel32Proc.NewProc("SetProcessAffinityMask")
	procEnumWindows            = user32Proc.NewProc("EnumWindows")
	procPostMessageW           = user32Proc.NewProc("PostMessageW")
	procPostThreadMessageW     = user32Proc.NewProc("PostThreadMessageW")
	windowSignalStates         sync.Map
	nextWindowSignalStateID    atomic.Uintptr
)

// Stop attempts the legacy NSSM stop sequence and reports whether the process
// had to be detached because terminate-on-stop is disabled.
func (p *Process) Stop() (bool, error) {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return false, nil
	}

	handle, err := windows.OpenProcess(
		windows.SYNCHRONIZE|windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_TERMINATE,
		false,
		uint32(p.cmd.Process.Pid),
	)
	if err != nil {
		return false, fmt.Errorf("open managed process: %w", err)
	}
	defer windows.CloseHandle(handle)

	exited, err := processExited(handle)
	if err != nil {
		return false, fmt.Errorf("query managed process: %w", err)
	}
	if exited {
		return false, p.waitForCompletion()
	}

	var warnErr error
	methods := p.service.EnabledStopMethods()
	pid := uint32(p.cmd.Process.Pid)

	if methods.Has(config.StopMethodConsole) {
		signaled, err := sendConsoleCtrlC(pid)
		if err != nil {
			warnErr = joinRuntimeError(warnErr, fmt.Errorf("console stop method: %w", err))
		}
		if signaled {
			exited, err := waitForProcessExit(handle, p.service.StopConsoleDelay)
			if err != nil {
				return false, joinRuntimeError(warnErr, fmt.Errorf("wait for console stop: %w", err))
			}
			if exited {
				return false, joinRuntimeError(warnErr, p.waitForCompletion())
			}
		}
	}

	if methods.Has(config.StopMethodWindow) {
		signaled, err := postWindowClose(pid)
		if err != nil {
			warnErr = joinRuntimeError(warnErr, fmt.Errorf("window stop method: %w", err))
		}
		if signaled {
			exited, err := waitForProcessExit(handle, p.service.StopWindowDelay)
			if err != nil {
				return false, joinRuntimeError(warnErr, fmt.Errorf("wait for window stop: %w", err))
			}
			if exited {
				return false, joinRuntimeError(warnErr, p.waitForCompletion())
			}
		}
	}

	if methods.Has(config.StopMethodThreads) {
		signaled, err := postThreadQuit(pid)
		if err != nil {
			warnErr = joinRuntimeError(warnErr, fmt.Errorf("thread stop method: %w", err))
		}
		if signaled {
			exited, err := waitForProcessExit(handle, p.service.StopThreadsDelay)
			if err != nil {
				return false, joinRuntimeError(warnErr, fmt.Errorf("wait for thread stop: %w", err))
			}
			if exited {
				return false, joinRuntimeError(warnErr, p.waitForCompletion())
			}
		}
	}

	if !methods.Has(config.StopMethodTerminate) {
		return true, warnErr
	}

	if p.service.KillProcessTree && p.job != 0 {
		if err := p.terminateJob(defaultTerminateCode); err != nil {
			return false, joinRuntimeError(warnErr, fmt.Errorf("terminate process tree: %w", err))
		}
	} else if err := windows.TerminateProcess(handle, defaultTerminateCode); err != nil {
		exited, exitErr := processExited(handle)
		if exitErr == nil && exited {
			return false, joinRuntimeError(warnErr, p.waitForCompletion())
		}
		return false, joinRuntimeError(warnErr, fmt.Errorf("terminate process: %w", err))
	}

	return false, joinRuntimeError(warnErr, p.waitForCompletion())
}

// Rotate requests online log rotation when AppRotateOnline is enabled.
func (p *Process) Rotate() error {
	if p == nil || p.logs == nil {
		return fmt.Errorf("rotation is not configured")
	}
	return p.logs.Rotate()
}

func (p *Process) waitForCompletion() error {
	if p == nil || p.done == nil {
		return nil
	}
	result, ok := <-p.done
	if !ok {
		return nil
	}
	return result.Err
}

func processExited(handle windows.Handle) (bool, error) {
	var exitCode uint32
	if err := windows.GetExitCodeProcess(handle, &exitCode); err != nil {
		return false, err
	}
	return exitCode != stillActiveExitCode, nil
}

func waitForProcessExit(handle windows.Handle, timeout time.Duration) (bool, error) {
	waitResult, err := windows.WaitForSingleObject(handle, waitMilliseconds(timeout))
	if err != nil {
		return false, err
	}

	switch waitResult {
	case windows.WAIT_OBJECT_0:
		return true, nil
	case uint32(windows.WAIT_TIMEOUT):
		return false, nil
	default:
		return false, fmt.Errorf("unexpected wait result %d", waitResult)
	}
}

func waitMilliseconds(timeout time.Duration) uint32 {
	if timeout <= 0 {
		return 0
	}
	max := time.Duration(^uint32(0)) * time.Millisecond
	if timeout >= max {
		return ^uint32(0)
	}
	return uint32(timeout.Milliseconds())
}

func sendConsoleCtrlC(pid uint32) (bool, error) {
	_, _, _ = procFreeConsole.Call()

	if err := attachConsole(pid); err != nil {
		switch err {
		case windows.ERROR_INVALID_HANDLE, windows.ERROR_GEN_FAILURE:
			return false, nil
		default:
			return false, err
		}
	}
	defer procFreeConsole.Call()

	if err := setConsoleCtrlHandler(true); err != nil {
		return false, err
	}
	defer procSetConsoleCtrlHandler.Call(0, 0)

	if err := windows.GenerateConsoleCtrlEvent(ctrlCEvent, 0); err != nil {
		return false, err
	}
	return true, nil
}

func postWindowClose(pid uint32) (bool, error) {
	state := &windowSignalState{pid: pid}
	stateID := registerWindowSignalState(state)
	defer unregisterWindowSignalState(stateID)

	callback := syscall.NewCallback(func(hwnd, param uintptr) uintptr {
		state, ok := lookupWindowSignalState(param)
		if !ok {
			return 1
		}
		var windowPID uint32
		if _, err := windows.GetWindowThreadProcessId(windows.HWND(hwnd), &windowPID); err == nil && windowPID == state.pid {
			if err := postMessage(windows.HWND(hwnd), wmClose, 0, 0); err == nil {
				state.signaled = true
			}
			if err := postMessage(windows.HWND(hwnd), wmEndSession, 1, endSessionCloseApp|endSessionCritical|endSessionLogoff); err == nil {
				state.signaled = true
			}
		}
		return 1
	})

	if err := enumWindows(callback, stateID); err != nil {
		return state.signaled, err
	}
	return state.signaled, nil
}

func postThreadQuit(pid uint32) (bool, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return false, err
	}
	defer windows.CloseHandle(snapshot)

	entry := windows.ThreadEntry32{Size: uint32(unsafe.Sizeof(windows.ThreadEntry32{}))}
	if err := windows.Thread32First(snapshot, &entry); err != nil {
		if err == windows.ERROR_NO_MORE_FILES {
			return false, nil
		}
		return false, err
	}

	signaled := false
	for {
		if entry.OwnerProcessID == pid {
			if err := postThreadMessage(entry.ThreadID, wmQuit, 0, 0); err == nil {
				signaled = true
			}
		}

		if err := windows.Thread32Next(snapshot, &entry); err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				return signaled, nil
			}
			return signaled, err
		}
	}
}

type windowSignalState struct {
	pid      uint32
	signaled bool
}

func attachConsole(pid uint32) error {
	r1, _, err := procAttachConsole.Call(uintptr(pid))
	if r1 == 0 {
		return err
	}
	return nil
}

func setConsoleCtrlHandler(ignore bool) error {
	var add uintptr
	if ignore {
		add = 1
	}
	r1, _, err := procSetConsoleCtrlHandler.Call(0, add)
	if r1 == 0 {
		return err
	}
	return nil
}

func getProcessAffinityMask(handle windows.Handle) (uintptr, uintptr, error) {
	var processMask uintptr
	var systemMask uintptr
	r1, _, err := procGetProcessAffinityMask.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&processMask)),
		uintptr(unsafe.Pointer(&systemMask)),
	)
	if r1 == 0 {
		return 0, 0, err
	}
	return processMask, systemMask, nil
}

func setProcessAffinityMask(handle windows.Handle, mask uintptr) error {
	r1, _, err := procSetProcessAffinityMask.Call(uintptr(handle), mask)
	if r1 == 0 {
		return err
	}
	return nil
}

func registerWindowSignalState(state *windowSignalState) uintptr {
	id := nextWindowSignalStateID.Add(1)
	windowSignalStates.Store(id, state)
	return id
}

func unregisterWindowSignalState(id uintptr) {
	windowSignalStates.Delete(id)
}

func lookupWindowSignalState(id uintptr) (*windowSignalState, bool) {
	value, ok := windowSignalStates.Load(id)
	if !ok {
		return nil, false
	}
	state, ok := value.(*windowSignalState)
	if !ok {
		return nil, false
	}
	return state, true
}

func enumWindows(callback, param uintptr) error {
	r1, _, err := procEnumWindows.Call(callback, param)
	if r1 == 0 && err != windows.ERROR_SUCCESS {
		return err
	}
	return nil
}

func postMessage(hwnd windows.HWND, message uint32, wParam, lParam uintptr) error {
	r1, _, err := procPostMessageW.Call(uintptr(hwnd), uintptr(message), wParam, lParam)
	if r1 == 0 {
		return err
	}
	return nil
}

func postThreadMessage(threadID uint32, message uint32, wParam, lParam uintptr) error {
	r1, _, err := procPostThreadMessageW.Call(uintptr(threadID), uintptr(message), wParam, lParam)
	if r1 == 0 {
		return err
	}
	return nil
}
