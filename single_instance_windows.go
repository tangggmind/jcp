//go:build windows

package main

import (
	"errors"
	"fmt"

	"golang.org/x/sys/windows"
)

type singleInstanceLock struct {
	handle windows.Handle
}

func acquireSingleInstance(name string) (*singleInstanceLock, bool, error) {
	mutexName, err := windows.UTF16PtrFromString(`Local\` + name)
	if err != nil {
		return nil, false, err
	}

	handle, err := windows.CreateMutex(nil, false, mutexName)
	if handle == 0 {
		return nil, false, fmt.Errorf("create single instance mutex: %w", err)
	}
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		_ = windows.CloseHandle(handle)
		return nil, false, nil
	}
	if err != nil {
		_ = windows.CloseHandle(handle)
		return nil, false, fmt.Errorf("create single instance mutex: %w", err)
	}

	return &singleInstanceLock{handle: handle}, true, nil
}

func (l *singleInstanceLock) Close() error {
	if l == nil || l.handle == 0 {
		return nil
	}
	err := windows.CloseHandle(l.handle)
	l.handle = 0
	return err
}
