package debug

import "fmt"

func ExampleBufferedDebug() {
	doWork := func(x int) (ret int) {
		dbg := BufferedDebug("doWork")
		defer dbg(false)

		defer func() {
			if ret == 42 {
				// Interesting value, let's print the debug log
				dbg(true)
			}
		}()

		fmt.Printf("x: %v\n", x)
		switch x {
		case 0:
			// Boring value
			return 1
		case 1:
			// Interesting value
			return 42
		default:
			// Broken value
			panic("we didn't expect this")
		}
	}

	doWork(0)
	doWork(1)

	defer func() {
		// BufferedDebug repanics any panic it caught; but we don't want 'go
		// test' to mark this example as failed because of it.
		recover()
	}()
	doWork(2)

	// Output:
	// ----------------------------------
	// Buffered debug output for "doWork"
	// x: 1
	// ----------------------------------
	// ----------------------------------
	// Buffered debug output for "doWork"
	// x: 2
	// ----------------------------------
}
