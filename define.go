package goproject

import (
	"errors"
	"fmt"
	"regexp"

	path "github.com/rhysd/abspath"
)

const (
	BlankPattern      = `^\s*$`
	CommentPattern    = `^\s*#`
	BeginPattern      = `^\s*begin-design:`
	EndPattern        = `^\s*end-design:`
	OpenPattern       = `\s*\(\s*`
	ClosePattern      = `\s*\)\s*`
	AloneClosePattern = `^\s*\)`
	ProjectPattern    = `^\s*project:`
	ExecPattern       = `^\s*exec:`
	DirPattern        = `^\s*dir:`
	NamePattern       = `^\s*name:`
	CopyPattern       = `^\s*copy:`
	GetPattern        = `^\s*get:`
	ModulePattern     = `^\s*module:`
	WorkspacePattern  = `^\s*workspace:`
	GitInitPattern    = `^\s*git-init:`
)

type CommandToken int

const (
	CmdBegin CommandToken = iota
	CmdExec
	CmdDir
	CmdCopy
	CmdGet
	CmdModule
	CmdWorkspace
	CmdGitInit
)

type astQueue struct {
	q []astNode
}

func (q *astQueue) push(an astNode) {
	q.q = append(q.q, an)
}

type nestLevel struct {
	path        path.AbsPath
	nest, limit int
}

type nestStack struct {
	nest []nestLevel
	pt   int
}

func (s *nestStack) push(n nestLevel) {
	s.nest = append(s.nest, n)
	s.pt++
}

func (s *nestStack) pop() nestLevel {
	s.pt--
	if s.pt < 0 {
		return nestLevel{}
	}
	return s.nest[s.pt]
}

type astNode struct {
	nest nestLevel
	cmd  CommandToken
	//tag       string
	cmdParams any
}

type designParser struct {
	text    []string
	line    int
	errs    []error
	project string
	nest    nestLevel
	regexs  map[string]*regexp.Regexp
	ast     astQueue
	nests   nestStack
}

func (d *designParser) current() (string, error) {
	if d.line < d.nest.limit {
		ret := d.text[d.line]
		return ret, nil
	}

	d.nesting(-1, 0)
	if d.nest.limit == 0 { //zero value empty struct
		return "", errors.New("EOF")
	}
	ret := d.text[d.line]
	return ret, nil
}

func (d *designParser) nextline() (string, error) {
	l, eof := d.current()
	d.line++
	return l, eof
}

func (d *designParser) setError(e error) {
	d.errs = append(d.errs, e)
}

func (d *designParser) Errors() string {
	var ret string
	for _, e := range d.errs {
		ret += e.Error() + "\n"
	}
	return ret
}

func (d *designParser) nesting(dir, lim int, path ...path.AbsPath) {
	//var trace = trace.New(os.Stderr)                             //<rmv/>
	//trace.Trace("----------------------entering nesting\n")      //<rmv/>
	//defer trace.Trace("----------------------leaving nesting\n") //<rmv/>

	if dir > 0 {
		if len(path) == 0 {
			d.setError(fmt.Errorf("no path given to increase nesting level at line %d", d.line))
			return
		}
		d.nests.push(d.nest)
		d.nest = nestLevel{limit: lim, nest: d.nest.nest + 1, path: path[0]}
		//trace.Trace("new nesting level ", d.nest.nest, " limit ", d.nest.limit, " path ", path) //<rmv/>
	}
	if dir < 0 {
		d.nest = d.nests.pop()
	}
}
func (d *designParser) hasErrors() bool {
	return !(len(d.errs) == 0)
}
