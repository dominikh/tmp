// SPDX-FileCopyrightText: 2025 Dominik Honnef and contributors
//
// SPDX-License-Identifier: MIT

package debug

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/sys/unix"
)

// BufferedDebug buffers all output to os.Stdout. The returned function must be
// called to restore the old value of os.Stdout. If the 'print' argument is
// true, or if the function was deferred and a panic occured, the buffered
// output will be written to os.Stdout. The returned function may be called
// multiple times, with all but the first call being no-ops.
func BufferedDebug(name string) func(print bool) {
	fd, err := unix.MemfdCreate("debug-"+name, unix.MFD_CLOEXEC)
	if err != nil {
		panic(fmt.Errorf("couldn't create memfd: %w", err))
	}

	f := os.NewFile(uintptr(fd), "debug-"+name)
	oldStdout := os.Stdout
	os.Stdout = f

	once := true
	return func(print bool) {
		if once {
			once = false
		} else {
			return
		}

		defer f.Close()
		os.Stdout = oldStdout

		doPrint := func() {
			if _, err := f.Seek(0, io.SeekStart); err != nil {
				panic(fmt.Errorf("couldn't seek memfd: %w", err))
			}
			l := fmt.Sprintf("Buffered debug output for %q", name)
			box := strings.Repeat("-", len(l))
			fmt.Println(box)
			fmt.Println(l)
			io.Copy(os.Stdout, f)
			fmt.Println(box)
		}

		if print {
			doPrint()
		} else if perr := recover(); perr != nil {
			doPrint()
			panic(perr)
		}
	}
}
