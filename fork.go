/*Package forkprocess is a small process execution that forked from the process
that execute it.

It is safe for systemd execution (tested it as the only init system) if used
properly.

  package main

  import (
    "fmt"
    "os"

    forkprocess "github.com/ik5/fork_process"
  )

  func main() {
    rpipe, wpipe, err := os.Pipe()
    if err != nil {
      fmt.Printf("Unable to get read and write files: %s\n", err)
      os.Exit(-2)
    }
    defer rpipe.Close()
    defer wpipe.Close()

    fork := forkprocess.NewForkProcess(
      rpipe, wpipe, wpipe,
      uint32(os.Getuid()), uint32(os.Getgid()), "/")

    processName := "/usr/bin/sleep"
    args := []string{processName, "60"}
    err = fork.Exec(true, processName, args)
    if err != nil {
      fmt.Printf("Unable to fork: %s\n", err)
      os.Exit(-1)
    }
    fmt.Printf("Run `ps axuf | grep '%s'` to see the process running\n", processName)
    os.Exit(0) // bye ...
  }

*/
package forkprocess

import (
	"os"
	"syscall"
)

// ForkProcess holds information for doing some forking
type ForkProcess struct {
	stdin    *os.File
	stdout   *os.File
	stderr   *os.File
	uid      uint32
	gid      uint32
	execPath string

	process *os.Process
}

// NewForkProcess initialize a new ForkProcess
//
// Parameters:
// stdin is the standard input to use, cannot be nil, or an error will be triggered on Exec by the OS.
// stdout is the standard output to use.
// stderr is the standard output to use.
// uid is the user id to execute the process with.
// gid is the group id to execute the process with.
// execPath is the path that the process will see as it was executed from.
func NewForkProcess(stdin, stdout, stderr *os.File, uid, gid uint32, execPath string) *ForkProcess {
	return &ForkProcess{
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
		uid:      uid,
		gid:      gid,
		execPath: execPath,
	}
}

// Exec execute a process and detach it from current process
//
// Parameters:
// release will auto release the process.
// processName is what to execute.
// args is a list of arguments to add for execution.
func (f *ForkProcess) Exec(release bool, processName string, args []string) error {
	var cred = &syscall.Credential{
		Uid:         f.uid,
		Gid:         f.gid,
		Groups:      nil,
		NoSetGroups: true,
	}

	var sysproc = &syscall.SysProcAttr{
		Credential: cred,
		Setsid:     true,
	}
	var attr = os.ProcAttr{
		Dir: f.execPath,
		Env: os.Environ(),
		Files: []*os.File{
			f.stdin,
			f.stdout,
			f.stderr,
		},
		Sys: sysproc,
	}
	var err error
	f.process, err = os.StartProcess(processName, args, &attr)
	if err != nil {
		return err
	}

	if release {
		err = f.process.Release()
		f.process = nil
		if err != nil {
			return err
		}
	}
	return nil
}

// Release releases any resources associated with the Process, rendering it
// unusable in the future. Release only needs to be called if Wait is not.
func (f ForkProcess) Release() error {
	if f.process != nil {
		return f.process.Release()
	}
	return nil
}
