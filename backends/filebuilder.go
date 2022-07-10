package backends

import (
	"fmt"
	"strings"
)

type Filebuilder struct {
	Einzug int

	strings.Builder
}

func (f *Filebuilder) Add(format string, a ...interface{}) {
	f.WriteString(strings.Repeat("\t", f.Einzug))
	f.WriteString(fmt.Sprintf(format, a...))
	f.WriteRune('\n')
}

func (f *Filebuilder) AddE(format string, a ...interface{}) {
	f.WriteString(strings.Repeat("\t", f.Einzug))
	f.WriteString(fmt.Sprintf(format, a...))
}

func (f *Filebuilder) AddK(format string, a ...interface{}) {
	f.WriteString(fmt.Sprintf(format, a...))
}

func (f *Filebuilder) AddNL() {
	f.WriteRune('\n')
}

func (f *Filebuilder) AddI(format string, a ...interface{}) {
	f.Add(format, a...)
	f.Einzug++
}

func (f *Filebuilder) AddD(format string, a ...interface{}) {
	f.Einzug--
	f.Add(format, a...)
}
