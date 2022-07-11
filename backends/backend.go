package backends

import (
	"github.com/urfave/cli/v2"
)

type Backend interface {
	GenerateCommand() *cli.Command
}

var StandardFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Usage:   "File to write output to. If not set, file will be written to stdout.",
	},
}

var Backends = []Backend{}

func RegisterBackend(b Backend) {
	Backends = append(Backends, b)
}
