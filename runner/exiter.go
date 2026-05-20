package runner

import "os"

type exiter interface {
	Exit(code int)
}

type defaultExiter struct{}

func (d defaultExiter) Exit(code int) {
	os.Exit(code)
}
