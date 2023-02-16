package util

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// AsyncTaskAndShowLoadingBar go a task and show loading bar
func AsyncTaskAndShowLoadingBar[T any](label string, task func() (T, error)) (T, error) {
	fmt.Print(label)
	defer fmt.Print("\r")
	var (
		i    = 0
		over = make(chan error, 0)
		t    T
	)
	go func() {
		ret, err := task()
		t = ret
		over <- err
	}()
	const maxDot = 10
	for {
		i++
		idx := i % maxDot
		fmt.Printf("\r%s(%.1fs)%s%s", label, 0.1*float32(i), strings.Repeat(".", idx), strings.Repeat(" ", maxDot-idx))
		select {
		case e := <-over:
			if e != nil {
				return t, e
			}
			return t, nil
		default:
		}
		time.Sleep(time.Millisecond * 100)
	}
}

// OpenWebBrowser open web browse
func OpenWebBrowser(url string) (err error) {
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	return err
}
