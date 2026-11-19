package fork

import (
	"os"
	"time"

	"github.com/sevlyar/go-daemon"
)

func Fork() {
	cntxt := &daemon.Context{
		WorkDir: ".",
		Umask:   0o27,
		Args:    os.Args,
	}

	child, err := cntxt.Reborn()
	if err != nil {
		panic(err)
	}

	if child != nil {
		time.Sleep(time.Second)
		os.Exit(0)
	}
}
