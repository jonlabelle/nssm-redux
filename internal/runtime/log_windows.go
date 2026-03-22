//go:build windows

package runtime

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jonlabelle/nssm-redux/internal/config"
)

type logManager struct {
	sinks map[string]*logSink
	order []*logSink
}

type logSink struct {
	path           string
	serviceName    string
	timestamp      bool
	rotation       config.LogRotationSettings
	reader         *os.File
	pipeWriter     *os.File
	done           chan error
	rotateRequests chan chan error

	mu        sync.Mutex
	file      *os.File
	lineStart bool
	size      uint64
}

func newLogManager() *logManager {
	return &logManager{sinks: make(map[string]*logSink)}
}

func (m *logManager) Sink(path string) *logSink {
	if m == nil {
		return nil
	}
	return m.sinks[strings.ToLower(path)]
}

func (m *logManager) AddSink(path string, sink *logSink) {
	key := strings.ToLower(path)
	m.sinks[key] = sink
	m.order = append(m.order, sink)
}

func (m *logManager) Start() {
	for _, sink := range m.order {
		go sink.run()
	}
}

func (m *logManager) Wait() error {
	if m == nil {
		return nil
	}
	var firstErr error
	for _, sink := range m.order {
		if err := <-sink.done; err != nil {
			firstErr = joinRuntimeError(firstErr, err)
		}
	}
	return firstErr
}

func (m *logManager) Rotate() error {
	if m == nil || len(m.order) == 0 {
		return fmt.Errorf("rotation is not configured")
	}
	var firstErr error
	rotated := false
	for _, sink := range m.order {
		if !sink.canRotateOnline() {
			continue
		}
		rotated = true
		reply := make(chan error, 1)
		sink.rotateRequests <- reply
		if err := <-reply; err != nil {
			firstErr = joinRuntimeError(firstErr, err)
		}
	}
	if !rotated {
		return fmt.Errorf("AppRotateOnline is not enabled")
	}
	return firstErr
}

func newLogSink(serviceName, path string, reader *os.File, rotation config.LogRotationSettings) *logSink {
	return &logSink{
		path:           path,
		serviceName:    serviceName,
		timestamp:      rotation.TimestampLog,
		rotation:       rotation,
		reader:         reader,
		done:           make(chan error, 1),
		rotateRequests: make(chan chan error, 1),
		lineStart:      true,
	}
}

func (s *logSink) canRotateOnline() bool {
	return s.rotation.Enabled && s.rotation.Online
}

func (s *logSink) run() {
	defer close(s.done)
	defer func() { _ = s.reader.Close() }()

	if err := rotateExistingFile(s.path, s.rotation); err != nil {
		s.done <- err
		return
	}

	file, size, err := openLogFile(s.path)
	if err != nil {
		s.done <- err
		return
	}
	s.file = file
	s.size = size
	defer s.closeFile()

	reader := bufio.NewReader(s.reader)
	buffer := make([]byte, 32*1024)
	var firstErr error

	for {
		select {
		case reply := <-s.rotateRequests:
			reply <- s.rotate()
		default:
		}

		n, err := reader.Read(buffer)
		if n > 0 {
			if writeErr := s.write(buffer[:n]); writeErr != nil {
				firstErr = joinRuntimeError(firstErr, writeErr)
			}
		}

		if err != nil {
			if err != io.EOF {
				firstErr = joinRuntimeError(firstErr, err)
			}
			s.done <- firstErr
			return
		}

		if s.canRotateOnline() && s.rotation.SizeBytes > 0 && s.size >= s.rotation.SizeBytes {
			if rotateErr := s.rotate(); rotateErr != nil {
				firstErr = joinRuntimeError(firstErr, rotateErr)
			}
		}
	}
}

func (s *logSink) write(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return fmt.Errorf("log file is not open")
	}
	if !s.timestamp {
		n, err := s.file.Write(data)
		s.size += uint64(n)
		return err
	}

	for len(data) > 0 {
		if s.lineStart {
			prefix := []byte(time.Now().Format("2006-01-02 15:04:05.000: "))
			n, err := s.file.Write(prefix)
			s.size += uint64(n)
			if err != nil {
				return err
			}
			s.lineStart = false
		}

		index := strings.IndexByte(string(data), '\n')
		if index < 0 {
			n, err := s.file.Write(data)
			s.size += uint64(n)
			return err
		}

		chunk := data[:index+1]
		n, err := s.file.Write(chunk)
		s.size += uint64(n)
		if err != nil {
			return err
		}
		s.lineStart = true
		data = data[index+1:]
	}

	return nil
}

func (s *logSink) rotate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file == nil {
		return fmt.Errorf("log file is not open")
	}
	if err := s.file.Close(); err != nil {
		return err
	}
	s.file = nil

	rotated := rotatedFilename(s.path, time.Now().UTC())
	if err := os.Rename(s.path, rotated); err != nil && !os.IsNotExist(err) {
		file, size, openErr := openLogFile(s.path)
		if openErr == nil {
			s.file = file
			s.size = size
		}
		return fmt.Errorf("rotate %s: %w", s.path, err)
	}
	if s.rotation.RotateDelay > 0 {
		time.Sleep(s.rotation.RotateDelay)
	}

	file, size, err := openLogFile(s.path)
	if err != nil {
		return err
	}
	s.file = file
	s.size = size
	s.lineStart = true
	return nil
}

func (s *logSink) closeFile() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		_ = s.file.Close()
		s.file = nil
	}
}

func rotateExistingFile(path string, rotation config.LogRotationSettings) error {
	if !rotation.Enabled {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if rotation.AgeThreshold > 0 && time.Since(info.ModTime()) < rotation.AgeThreshold {
		return nil
	}
	if rotation.SizeBytes > 0 && uint64(info.Size()) < rotation.SizeBytes {
		return nil
	}
	rotated := rotatedFilename(path, info.ModTime().UTC())
	if err := os.Rename(path, rotated); err != nil {
		return err
	}
	if rotation.RotateDelay > 0 {
		time.Sleep(rotation.RotateDelay)
	}
	return nil
}

func rotatedFilename(path string, now time.Time) string {
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	return fmt.Sprintf("%s-%s%s", base, now.Format("20060102T150405.000"), ext)
}

func openLogFile(path string) (*os.File, uint64, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, 0, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, 0, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, err
	}
	return file, uint64(info.Size()), nil
}
