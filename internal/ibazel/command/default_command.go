// Copyright 2017 The Bazel Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package command

import (
	"bytes"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/bazelbuild/bazel-watcher/internal/ibazel/log"
	"github.com/bazelbuild/bazel-watcher/internal/ibazel/process_group"
)

type defaultCommand struct {
	target            string
	startupArgs       []string
	bazelArgs         []string
	args              []string
	pg                process_group.ProcessGroup
	termSync          sync.Once
	doneChan          chan bool
	subprocessRunning atomic.Bool
}

// DefaultCommand is the normal mode of interacting with iBazel. If you start a
// server in this mode and notify of changes the server will be killed and
// restarted.
func DefaultCommand(startupArgs []string, bazelArgs []string, target string, args []string) Command {
	return &defaultCommand{
		target:      target,
		startupArgs: startupArgs,
		bazelArgs:   bazelArgs,
		args:        args,
	}
}

func (c *defaultCommand) Terminate() {
	if !c.IsSubprocessRunning() {
		c.pg = nil
		return
	}
	c.termSync.Do(func() {
		terminate(c.pg, c.doneChan)
	})
	c.pg = nil
	c.subprocessRunning.Store(false)
}

func (c *defaultCommand) Kill() {
	if !c.IsSubprocessRunning() {
		kill(c.pg)
	}
}

func (c *defaultCommand) Start() (*bytes.Buffer, error) {
	b := bazelNew()
	b.SetStartupArgs(c.startupArgs)
	b.SetArguments(c.bazelArgs)

	b.WriteToStderr(true)
	b.WriteToStdout(true)

	var outputBuffer *bytes.Buffer
	outputBuffer, pg := start(b, c.target, c.args)
	c.pg = pg
	doneChan := make(chan bool, 1)
	c.doneChan = doneChan
	c.subprocessRunning.Store(true)

	pg.RootProcess().Env = os.Environ()

	var err error
	if err = pg.Start(); err != nil {
		log.Errorf("Error starting process: %v", err)
		return outputBuffer, err
	}
	log.Log("Starting...")

	logf := log.Logf
	go func() {
		err := pg.Wait()

		code := 0
		if exiterr, ok := err.(*exec.ExitError); ok {
			code = exiterr.ExitCode()
		}
		logf("Exited (%d)", code)

		doneChan <- true
		c.subprocessRunning.Store(false)
	}()

	c.termSync = sync.Once{}
	return outputBuffer, nil
}

func (c *defaultCommand) NotifyOfChanges() *bytes.Buffer {
	c.Terminate()
	c.Start()
	return nil
}

func (c *defaultCommand) IsSubprocessRunning() bool {
	return c.subprocessRunning.Load()
}
