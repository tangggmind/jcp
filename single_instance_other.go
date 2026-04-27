//go:build !windows

package main

import (
	"fmt"
	"hash/fnv"
	"net"
)

type singleInstanceLock struct {
	listener net.Listener
}

func acquireSingleInstance(name string) (*singleInstanceLock, bool, error) {
	port := 31000 + int(hashName(name)%20000)
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, false, nil
	}
	return &singleInstanceLock{listener: listener}, true, nil
}

func (l *singleInstanceLock) Close() error {
	if l == nil || l.listener == nil {
		return nil
	}
	return l.listener.Close()
}

func hashName(name string) uint32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(name))
	return h.Sum32()
}
