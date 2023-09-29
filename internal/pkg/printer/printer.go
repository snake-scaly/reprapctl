package printer

type Printer interface {
	Send(cmd Command) error
}

type Command struct {
	command         string
	responseHandler func(response string)
}
