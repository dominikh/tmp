package main

import (
	"fmt"
	"time"

	"honnef.co/go/stuff/container/persistent"
)

func main() {
	const maxBranch = 4
	for n := 1; n <= maxBranch*10; n++ {
		v := persistent.NewVector(make([]int, n))
		for range 1000 {
			t := time.Now()
			for range 1000 {
				v.Pop()
			}
			d := time.Since(t)
			fmt.Printf("%d,%d,%d\n", n, 1000, int64(d))
		}
	}
}
