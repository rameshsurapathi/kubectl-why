// In cobra applications, main.go is kept intentionally tiny.
// Its only job is to call cmd.Execute(), which starts the CLI.

package main

import "github.com/rameshsurapathi/kubectl-why/cmd"

func main() {
	cmd.Execute()
}
