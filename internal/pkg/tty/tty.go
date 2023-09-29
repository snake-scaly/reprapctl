package tty

import (
	"fmt"
	"golang.org/x/sys/unix"
)

// ConfigureTty sets up a TTY device for communication.
func ConfigureTty(path string) error {
	var err error

	fd, err := unix.Open(path, unix.O_RDWR|unix.O_NONBLOCK, 0)
	if err != nil {
		return fmt.Errorf("%v: %v", path, err)
	}

	defer func(fd int) {
		_ = unix.Close(fd)
	}(fd)

	termios := unix.Termios{
		Iflag: unix.INPCK,
		Oflag: 0,
		Cflag: unix.PARENB | unix.CRTSCTS,
		Lflag: 0,
		Line:  0,
		Cc: [19]uint8{
			unix.VDISCARD,
			0,
			unix.VEOF,
			unix.VEOL,
			unix.VEOL2,
			unix.VERASE,
			unix.VINTR,
			unix.VKILL,
			unix.VLNEXT,
			unix.VMIN,
			unix.VQUIT,
			unix.VREPRINT,
			unix.VSTART,
			0,
			unix.VSTOP,
			unix.VSUSP,
			unix.VSWTC,
			unix.VTIME,
			unix.VWERASE,
		},
	}

	err = unix.IoctlSetTermios(fd, unix.TCSETS, &termios)
	if err != nil {
		return fmt.Errorf("%v: %v", path, err)
	}

	return nil
}
