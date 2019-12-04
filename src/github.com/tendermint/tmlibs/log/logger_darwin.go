package log

import (
	"syscall"
)

func dup2(fd1, fd2 int) {
	syscall.Dup2(fd1, fd2)
}
