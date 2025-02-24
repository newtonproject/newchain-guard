package cli

import "testing"

func TestServer(t *testing.T) {
	cli := NewCLI()

	cli.TestCommand("server")
}
