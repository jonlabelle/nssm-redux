//go:build windows

package scm

import (
	"fmt"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ProcessNode describes one process in a managed service's process tree.
type ProcessNode struct {
	PID       uint32
	ParentPID uint32
	Depth     int
	ImagePath string
}

// ProcessTree captures the currently running process hierarchy for a service.
type ProcessTree struct {
	Service   string
	ProcessID uint32
	Nodes     []ProcessNode
}

func ProcessTreeForService(name string) (ProcessTree, error) {
	manager, serviceHandle, _, err := openService(name)
	if err != nil {
		return ProcessTree{}, err
	}
	defer manager.Disconnect()
	defer serviceHandle.Close()

	status, err := serviceHandle.Query()
	if err != nil {
		return ProcessTree{}, fmt.Errorf("query service: %w", err)
	}
	if status.ProcessId == 0 {
		return ProcessTree{Service: name}, nil
	}

	rootCreation, err := processCreationTime(status.ProcessId)
	if err != nil {
		return ProcessTree{}, err
	}

	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return ProcessTree{}, fmt.Errorf("create process snapshot: %w", err)
	}
	defer windows.CloseHandle(snapshot)

	entries, err := readProcessEntries(snapshot)
	if err != nil {
		return ProcessTree{}, err
	}

	tree := ProcessTree{
		Service:   name,
		ProcessID: status.ProcessId,
	}
	appendProcessTree(&tree, entries, status.ProcessId, 0, rootCreation)
	return tree, nil
}

func appendProcessTree(tree *ProcessTree, entries []windows.ProcessEntry32, pid uint32, depth int, rootCreation windows.Filetime) {
	node := ProcessNode{
		PID:       pid,
		Depth:     depth,
		ImagePath: queryProcessImage(pid),
	}
	for _, entry := range entries {
		if entry.ProcessID == pid {
			node.ParentPID = entry.ParentProcessID
			break
		}
	}
	tree.Nodes = append(tree.Nodes, node)

	for _, entry := range entries {
		if entry.ParentProcessID != pid {
			continue
		}
		childCreation, err := processCreationTime(entry.ProcessID)
		if err != nil {
			continue
		}
		if compareFiletime(childCreation, rootCreation) < 0 {
			continue
		}
		appendProcessTree(tree, entries, entry.ProcessID, depth+1, rootCreation)
	}
}

func readProcessEntries(snapshot windows.Handle) ([]windows.ProcessEntry32, error) {
	entry := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
	if err := windows.Process32First(snapshot, &entry); err != nil {
		if err == windows.ERROR_NO_MORE_FILES {
			return nil, nil
		}
		return nil, fmt.Errorf("enumerate processes: %w", err)
	}

	entries := []windows.ProcessEntry32{entry}
	for {
		next := windows.ProcessEntry32{Size: uint32(unsafe.Sizeof(windows.ProcessEntry32{}))}
		if err := windows.Process32Next(snapshot, &next); err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				return entries, nil
			}
			return nil, fmt.Errorf("enumerate processes: %w", err)
		}
		entries = append(entries, next)
	}
}

func processCreationTime(pid uint32) (windows.Filetime, error) {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return windows.Filetime{}, fmt.Errorf("open process %d: %w", pid, err)
	}
	defer windows.CloseHandle(handle)

	var creation, exit, kernel, user windows.Filetime
	if err := windows.GetProcessTimes(handle, &creation, &exit, &kernel, &user); err != nil {
		return windows.Filetime{}, fmt.Errorf("query process %d times: %w", pid, err)
	}
	return creation, nil
}

func queryProcessImage(pid uint32) string {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return fmt.Sprintf("PID %d", pid)
	}
	defer windows.CloseHandle(handle)

	buffer := make([]uint16, windows.MAX_PATH)
	size := uint32(len(buffer))
	if err := windows.QueryFullProcessImageName(handle, 0, &buffer[0], &size); err == nil {
		return windows.UTF16ToString(buffer[:size])
	}
	return fmt.Sprintf("PID %d", pid)
}

func compareFiletime(a, b windows.Filetime) int {
	av := (uint64(a.HighDateTime) << 32) | uint64(a.LowDateTime)
	bv := (uint64(b.HighDateTime) << 32) | uint64(b.LowDateTime)
	switch {
	case av < bv:
		return -1
	case av > bv:
		return 1
	default:
		return 0
	}
}

func FormatProcessTree(tree ProcessTree) []string {
	lines := make([]string, 0, len(tree.Nodes))
	for _, node := range tree.Nodes {
		lines = append(lines, fmt.Sprintf("%s%d %s", strings.Repeat("  ", node.Depth), node.PID, node.ImagePath))
	}
	return lines
}
