package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pschlump/dbgo"
)

func RunCmdImpl(cmdName string, args []string) (out string, err error) {
	dbgo.Fprintf(os.Stderr, "%(Yellow)Info: command to run [%s] args ----->%s<----- at:%(LF)\n", cmdName, dbgo.SVar(args))

	dbgo.DbFprintf("cmdrunner.1", os.Stderr, "AT: %s\n", dbgo.LF())
	cmd := exec.Command(cmdName, args...)
	buf, e0 := cmd.CombinedOutput()
	if e0 != nil {
		err = e0
		return
	}
	out = string(buf)
	fmt.Fprintf(os.Stderr, "out --->>>%s<<<---\n", out)

	return
}
