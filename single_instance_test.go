package main

import (
	"fmt"
	"testing"
	"time"
)

func TestAcquireSingleInstanceAllowsOnlyOneHolder(t *testing.T) {
	name := fmt.Sprintf("jcp-test-single-instance-%d", time.Now().UnixNano())

	first, acquired, err := acquireSingleInstance(name)
	if err != nil {
		t.Fatalf("first acquire returned error: %v", err)
	}
	if !acquired {
		t.Fatalf("first acquire should succeed")
	}

	second, acquired, err := acquireSingleInstance(name)
	if err != nil {
		t.Fatalf("second acquire returned error: %v", err)
	}
	if acquired {
		_ = second.Close()
		t.Fatalf("second acquire should be rejected while first holder is alive")
	}

	if err := first.Close(); err != nil {
		t.Fatalf("close first lock: %v", err)
	}

	third, acquired, err := acquireSingleInstance(name)
	if err != nil {
		t.Fatalf("third acquire returned error: %v", err)
	}
	if !acquired {
		t.Fatalf("third acquire should succeed after first holder closes")
	}
	if err := third.Close(); err != nil {
		t.Fatalf("close third lock: %v", err)
	}
}
