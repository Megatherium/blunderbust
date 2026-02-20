// Copyright (C) 2026 megatherium
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package tmux

import (
	"context"
	"fmt"
	"strings"
)

// FakeRunner implements CommandRunner for testing without real subprocess execution.
type FakeRunner struct {
	Commands     []string
	Output       map[string][]byte
	Errors       map[string]error
	AlwaysReturn []byte
	AlwaysError  error
}

// NewFakeRunner creates a new FakeRunner.
func NewFakeRunner() *FakeRunner {
	return &FakeRunner{
		Commands: make([]string, 0),
		Output:   make(map[string][]byte),
		Errors:   make(map[string]error),
	}
}

// Run records the command invocation and returns configured output/error.
func (f *FakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	key := f.commandKey(name, args...)
	f.Commands = append(f.Commands, key)

	if f.AlwaysError != nil {
		return nil, f.AlwaysError
	}
	if f.AlwaysReturn != nil {
		return f.AlwaysReturn, nil
	}
	if err, ok := f.Errors[key]; ok {
		return nil, err
	}
	if output, ok := f.Output[key]; ok {
		return output, nil
	}

	return nil, fmt.Errorf("no output configured for command: %s", key)
}

// SetOutput configures the output for a specific command.
func (f *FakeRunner) SetOutput(name string, args []string, output []byte) {
	key := f.commandKey(name, args...)
	f.Output[key] = output
}

// SetError configures the error for a specific command.
func (f *FakeRunner) SetError(name string, args []string, err error) {
	key := f.commandKey(name, args...)
	f.Errors[key] = err
}

// commandKey creates a unique key for caching command results.
func (f *FakeRunner) commandKey(name string, args ...string) string {
	return fmt.Sprintf("%s %s", name, strings.Join(args, " "))
}

// Verify interface compliance at compile time.
var _ CommandRunner = (*RealRunner)(nil)
var _ CommandRunner = (*FakeRunner)(nil)
