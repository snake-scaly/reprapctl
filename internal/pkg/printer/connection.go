package printer

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

func OpenFake() (io.ReadWriteCloser, error) {
	inReader, inWriter := io.Pipe()
	outReader, outWriter := io.Pipe()
	c := connection{
		inReader:  inReader,
		inWriter:  inWriter,
		outReader: outReader,
		outWriter: outWriter,
	}
	go c.run()
	return &c, nil
}

type connection struct {
	inReader  *io.PipeReader
	inWriter  *io.PipeWriter
	outReader *io.PipeReader
	outWriter *io.PipeWriter
}

func (c *connection) run() {
	s := bufio.NewScanner(c.inReader)
	m20Re := regexp.MustCompile(`^[mM]20\b`)
	for s.Scan() {
		t := s.Text()
		switch {
		case m20Re.MatchString(t):
			fmt.Fprintln(c.outWriter, "Begin file list:")
			fmt.Fprintln(c.outWriter, "Foo bar baz.gcode")
			fmt.Fprintln(c.outWriter, "End file list")
			fmt.Fprintln(c.outWriter, "ok")
		default:
			fmt.Fprintln(c.outWriter, "ok")
		}
	}
	c.outWriter.Close()
}

func (c *connection) Read(p []byte) (int, error) {
	return c.outReader.Read(p)
}

func (c *connection) Write(p []byte) (int, error) {
	return c.inWriter.Write(p)
}

func (c *connection) Close() error {
	return c.inWriter.Close()
}
