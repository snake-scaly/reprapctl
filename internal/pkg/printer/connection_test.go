package printer

import (
	"bufio"
	"fmt"
	"testing"
)

func TestOpenFake(t *testing.T) {
	var err error

	c, err := OpenFake()

	if err != nil {
		t.Fatalf("OpenFake failed: %v", err)
	}
	if c == nil {
		t.Fatalf("Connection is nil")
	}

	n, err := fmt.Fprintln(c, "G1 X0 Y0")
	if err != nil {
		t.Errorf("Write failed: %v", err)
	}

	if n != 9 {
		t.Errorf("Write failed: want 9, got %v", n)
	}

	s := bufio.NewScanner(c)
	if !s.Scan() {
		t.Errorf("Failed to scan the input: %v", s.Err())
	}
	if tx := s.Text(); tx != "ok" {
		t.Errorf("Unexpected response: want `ok', got `%v'", tx)
	}

	c.Close()
}
