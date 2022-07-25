package backends

import (
	"github.com/urfave/cli/v2"
)

type Backend interface {
	GenerateCommand() *cli.Command
}

var StandardFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     "outdir",
		Usage:    "The directory to output generated documentation to",
		Required: true,
		Aliases:  []string{"o"},
	},
	&cli.StringFlag{
		Name:        "workspace",
		Usage:       "The directory to load a workspace from",
		DefaultText: ".",
		Aliases:     []string{"w"},
	},
}

var Backends = []Backend{}

func RegisterBackend(b Backend) {
	Backends = append(Backends, b)
}
