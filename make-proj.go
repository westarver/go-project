package goproject

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/bitfield/script"
	path "github.com/rhysd/abspath"
)

//─────────────┤ initProject ├─────────────

func initProject(name, desn string) (*designParser, error) {
	//var trace = trace.New(os.Stderr)                                       //<rmv/>
	//trace.Trace("----------------------------entering initProject\n")      //<rmv/>
	//defer trace.Trace("----------------------------leaving initProject\n") //<rmv/>
	//trace.Trace("project name as passed ", name)                           //<rmv/>

	p := script.Exec("xpanda " + desn)
	dsn, err := p.Slice()
	if err != nil {
		return nil, err
	}

	rs := mapFromPatSlice([]string{
		BeginPattern,
		ProjectPattern,
		ExecPattern,
		BlankPattern,
		CommentPattern,
		AloneClosePattern,
		EndPattern,
		DirPattern,
		CopyPattern,
		GetPattern,
		ModulePattern,
		WorkspacePattern,
		GitInitPattern,
	})

	if name == "--" {
		name = ""
	}

	dp := designParser{
		text:    dsn,
		line:    0,
		errs:    []error{},
		project: name,
		nest:    nestLevel{},
		regexs:  rs,
		ast:     astQueue{},
		nests:   nestStack{},
	}

	wd, err := path.Getwd()
	if err != nil {
		dp.setError(fmt.Errorf("unable to get working directory %v", err))
		return &dp, err
	}

	dp.nest.path = wd
	dp.nest.limit = len(dsn)
	dp.nest.nest = 0

	var atEOF bool
	for {
		l, eof := dp.nextline()
		if eof != nil {
			atEOF = true
			break
		}

		if dp.regexs[BeginPattern].MatchString(l) {
			break
		}
	}

	start := dp.line
	if !atEOF {
		for {
			l, eof := dp.nextline()
			if eof != nil {
				dp.nest.limit = len(dp.text)
				break
			}

			if dp.regexs[EndPattern].MatchString(l) {
				dp.nest.limit = dp.line
				dp.line = start
				break
			}
		}
	}

	if !atEOF {
		dp.nests.push(nestLevel{})
		scanBegin(&dp)
	}

	return &dp, nil
}

//─────────────┤ scanBegin ├─────────────

func scanBegin(d *designParser) {
	//var trace = trace.New(os.Stderr)                                     //<rmv/>
	//trace.Trace("----------------------------entering scanBegin\n")      //<rmv/>
	//defer trace.Trace("----------------------------leaving scanBegin\n") //<rmv/>

	_, eof := d.nextline()
	if eof != nil {
		return
	}

	scanf := scanCurrentLevel
	for {
		scanf = scanf(d)
		if scanf == nil {
			break
		}
	}
}

type scanfunc func(*designParser) scanfunc

//─────────────┤ scanCurrentLevel ├─────────────

func scanCurrentLevel(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)                                            //<rmv/>
	//trace.Trace("----------------------------entering scanCurrentLevel\n")      //<rmv/>
	//defer trace.Trace("----------------------------leaving scanCurrentLevel\n") //<rmv/>
	var reg *regexp.Regexp
	ln, eof := d.current()
	if eof != nil {
		return nil
	}

	for _, r := range d.regexs {
		if r.MatchString(ln) {
			//trace.Trace("ln ", ln, " matched ", r.String()) //<rmv/>
			reg = r
			break
		}
	}

	switch reg {
	case d.regexs[BlankPattern]:
		return scanBlank
	case d.regexs[CommentPattern]:
		return scanBlank
	case d.regexs[AloneClosePattern]:
		return scanBlank
	case d.regexs[ProjectPattern]:
		return scanProject
	case d.regexs[ExecPattern]:
		return scanExec
	case d.regexs[DirPattern]:
		return scanDir
	case d.regexs[EndPattern]:
		return nil
	case d.regexs[CopyPattern]:
		return scanCopy
	case d.regexs[GetPattern]:
		return scanGet
	case d.regexs[WorkspacePattern]:
		return scanWorkspace
	case d.regexs[ModulePattern]:
		return scanModule
	case d.regexs[GitInitPattern]:
		return scanGitInit
	}
	//trace.Trace("no match for ", ln) //<rmv/>
	return nil
}

//<rgn scanBlank>
//─────────────┤ scanBlank ├─────────────

func scanBlank(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)       //<rmv/>
	//trace.Trace("entering scanBlank")      //<rmv/>
	//defer trace.Trace("leaving scanBlank") //<rmv/>
	_, eof := d.nextline()
	if eof == nil {
		return scanCurrentLevel
	}

	d.setError(fmt.Errorf("unexpected EOF at line %d", d.line))
	return nil
} //</rgn scanBlank>

//<rgn scanProject>
//─────────────┤ scanProject ├─────────────

func scanProject(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)         //<rmv/>
	//trace.Trace("entering scanProject")      //<rmv/>
	//defer trace.Trace("leaving scanProject") //<rmv/>
	cur, eof := d.nextline()
	if eof == nil {
		r := regexStatFromPat(ProjectPattern, cur)
		//trace.Trace("project match ", r.after) //<rmv/>
		if r.after != "" {
			// a name passed on the command line overrides the one in the design file
			prj := strings.Trim(r.after, "\t ")
			if d.project == "" {
				d.project = prj
			}
		}
		//trace.Trace("project name ", d.project) //<rmv/>
		return scanCurrentLevel
	}

	return nil
} //</rgn scanProject>

//<rgn scanExec>
//─────────────┤ scanExec ├─────────────

func scanExec(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)      //<rmv/>
	//trace.Trace("entering scanExec")      //<rmv/>
	//defer trace.Trace("leaving scanExec") //<rmv/>
	var bal bool
	var n = -1

	cur, eof := d.current()

	if eof == nil {
		//trace.Trace("cur from exec ", cur) //<rmv/>
		r := regexStatFromPat(OpenPattern, cur)
		if r.length > 0 {
			//trace.Trace("parens found") //<rmv/>
			n, bal = scanToClose(d)
			//trace.Trace("multiline count ", n) //<rmv/>
		}

		if n == -1 { // no parentheses found
			r := regexStatFromPat(ExecPattern, cur)
			if r.length > 0 {
				cur = strings.Trim(r.after, "\t ")
			}
			// cmd, _ := shell.Split(cur)
			//trace.Trace("exec command ", cmd) //<rmv/>
			// arg := strings.Join(cmd[1:], " '")
			d.ast.push(astNode{nest: d.nest, cmd: CmdExec, cmdParams: cur})
			d.line++
			return scanCurrentLevel
		}

		if n > 0 { // multi-line string
			if !bal {
				d.setError(fmt.Errorf("unbalanced parentheses near line %d", d.line+n))
				return nil
			}
			startLn := d.line
			//arg := getMultilineQuotedStr(d.text[startLn : startLn+n])
			cur = strings.Join(d.text[startLn:startLn+n], "\n")
			cur = stripParens(cur)
			//trace.Trace("after strip ", cur) //<rmv/>
			r := regexStatFromPat(ExecPattern, cur)
			if r.length > 0 {
				cur = strings.Trim(r.after, "\t ")
			}
			// args, ok := shell.Split(cur)
			// if !ok {
			// 	d.setError(fmt.Errorf("unbalanced quotes or backslashes in exec command near line %d", startLn+n))
			// 	return nil
			// }
			//trace.Trace("exec command ", args) //<rmv/>
			// arg := strings.Join(args[1:], " ")
			d.ast.push(astNode{nest: d.nest, cmd: CmdExec, cmdParams: cur})
			d.line += n
			return scanCurrentLevel
		}

		if n == 0 { //parentheses found and closed on one line
			if bal {
				cur = stripParens(cur)
				//trace.Trace("cur after strip ", cur) //<rmv/>
				r := regexStatFromPat(ExecPattern, cur)
				if r.length > 0 {
					cur = strings.Trim(r.after, "\t ")
				}

				d.ast.push(astNode{nest: d.nest, cmd: CmdExec, cmdParams: cur})
				//trace.Trace("exec command ", cur) //<rmv/>
				d.line++
				return scanCurrentLevel
			} else {
				d.setError(fmt.Errorf("unbalanced parentheses at line %d", d.line))
			}
		}
	}

	return nil
} //</rgn scanExec>

//<rgn scanDir>
//─────────────┤ scanDir ├─────────────
// scanDir is the only scanner that has to deal with multiple nesting levels
func scanDir(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)                                     //<rmv/>
	//trace.Trace("------------------------------entering scanDir\n")      //<rmv/>
	//defer trace.Trace("\n----------------------------leaving scanDir\n") //<rmv/>
	//trace.Trace("current ", d.text[d.line]) //<rmv/>
	var bal bool
	var n = -1

	startLn := d.line
	cur, eof := d.current()

	if eof == nil {

		r := regexStatFromPat(DirPattern, cur)
		if r.length > 0 {
			cur = strings.Trim(r.after, "\t ")
		}
		//trace.Trace("cur from dir ", cur) //<rmv/>

		r = regexStatFromPat(OpenPattern, cur)
		if r.length > 0 {
			//trace.Trace("parens found") //<rmv/>
			n, bal = scanToClose(d)
			//trace.Trace("multiline count ", n) //<rmv/>
		}

		if n == -1 { // no parentheses found
			path, err := path.ExpandFrom(cur)
			if err != nil {
				d.setError(fmt.Errorf("invalid path given at line %d", d.line))
				return nil
			}
			d.ast.push(astNode{nest: d.nest, cmd: CmdDir, cmdParams: path})
			//trace.Trace("new dir ", path) //<rmv/>
			d.line++
			return scanCurrentLevel
		}

		if n > 0 { // multi-line nesting level
			if !bal {
				d.setError(fmt.Errorf("unbalanced parentheses near line %d", startLn+n))
				return nil
			}

			cur = strings.Trim(stripParens(cur), "\t ")
			path, err := path.ExpandFrom(cur)
			if err != nil {
				d.setError(fmt.Errorf("invalid path given at line %d", d.line))
				return nil
			}
			d.nesting(1, startLn+n, path)
			//trace.Trace("new nesting level ", d.nest.nest, " limit ", startLn+n, " path ", path) //<rmv/>
			d.ast.push(astNode{nest: d.nest, cmd: CmdDir, cmdParams: path})
			//trace.Trace("new dir ", path.String()) //<rmv/>
			d.line++
			return scanCurrentLevel
		}

		if n == 0 { //parentheses found and closed on one line
			if !bal {
				d.setError(fmt.Errorf("unbalanced parentheses near line %d", d.line+n))
				return nil
			}

			cur = strings.Trim(stripParens(cur), "\t ")
			path, err := path.ExpandFrom(cur)
			if err != nil {
				d.setError(fmt.Errorf("invalid path given at line %d", d.line))
				return nil
			}
			d.ast.push(astNode{nest: d.nest, cmd: CmdDir, cmdParams: path})
			//trace.Trace("new dir ", path.String()) //<rmv/>
			d.line++
			return scanCurrentLevel
		}
	}
	return nil
} //</rgn scanDir>

//<rgn scanCopy>
//─────────────┤ scanCopy ├─────────────

func scanCopy(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)      //<rmv/>
	//trace.Trace("entering scanCopy")      //<rmv/>
	//defer trace.Trace("leaving scanCopy") //<rmv/>
	cur, eof := d.nextline()
	//trace.Trace("current line in scanCopy ", cur) //<rmv/>
	if eof == nil {
		r := regexStatFromPat(CopyPattern, cur)
		if r.length == 0 {
			d.setError(fmt.Errorf("landed in scanCopy but did not match CopyPattern"))
			return nil
		}
		//trace.Trace("copy source ", strings.Trim(r.after, "\t ")) //<rmv/>
		d.ast.push(astNode{nest: d.nest, cmd: CmdCopy, cmdParams: strings.Trim(r.after, "\t ")})
		return scanCurrentLevel
	} else {
		d.setError(fmt.Errorf("unexpected EOF at line %d", d.line))
	}

	return nil
} //</rgn scanCopy>

//<rgn scanGet>
//─────────────┤ scanGet ├─────────────

func scanGet(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)     //<rmv/>
	//trace.Trace("entering scanGet")      //<rmv/>
	//defer trace.Trace("leaving scanGet") //<rmv/>
	cur, eof := d.nextline()
	//trace.Trace("current line in scanGet ", cur) //<rmv/>
	if eof == nil {
		r := regexStatFromPat(GetPattern, cur)
		if r.length == 0 {
			d.setError(fmt.Errorf("landed in scanGet but did not match GetPattern"))
			return nil
		}
		d.ast.push(astNode{nest: d.nest, cmd: CmdGet, cmdParams: strings.Trim(r.after, "\t ")})
		return scanCurrentLevel
	} else {
		d.setError(fmt.Errorf("unexpected EOF at line %d", d.line))
	}

	return nil
} //</rgn scanGet>

//<rgn scanWorkspace>
//─────────────┤ scanWorkspace ├─────────────

func scanWorkspace(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)           //<rmv/>
	//trace.Trace("entering scanWorkspace")      //<rmv/>
	//defer trace.Trace("leaving scanWorkspace") //<rmv/>
	cur, eof := d.nextline()
	if eof == nil {
		r := regexStatFromPat(WorkspacePattern, cur)
		if r.length == 0 {
			d.setError(fmt.Errorf("landed in scanWorkspace but did not match Workspace Pattern"))
			return nil
		}
		cur = strings.Trim(r.after, "\t ")
		//trace.Trace("workspace params ", cur) //<rmv/>
		d.ast.push(astNode{nest: d.nest, cmd: CmdWorkspace, cmdParams: cur})
		return scanCurrentLevel
	} else {
		d.setError(fmt.Errorf("unexpected EOF at line %d", d.line))
	}

	return nil
} //</rgn scanWorkspace>

//<rgn scanModule>
//─────────────┤ scanModule ├─────────────

func scanModule(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)        //<rmv/>
	//trace.Trace("entering scanModule")      //<rmv/>
	//defer trace.Trace("leaving scanModule") //<rmv/>
	cur, eof := d.nextline()
	if eof == nil {
		r := regexStatFromPat(ModulePattern, cur)
		if r.length == 0 {
			d.setError(fmt.Errorf("landed in scanModule but did not match ModulePattern"))
			return nil
		}

		cur = strings.Trim(r.after, "\t ")
		//trace.Trace("module name ", cur) //<rmv/>
		d.ast.push(astNode{nest: d.nest, cmd: CmdModule, cmdParams: cur})
		return scanCurrentLevel
	} else {
		d.setError(fmt.Errorf("unexpected EOF at line %d", d.line))
	}

	return nil
} //</rgn scanModule>

//<rgn scanGitInit>
//─────────────┤ scanGitInit ├─────────────

func scanGitInit(d *designParser) scanfunc {
	//var trace = trace.New(os.Stderr)         //<rmv/>
	//trace.Trace("entering scanGitInit")      //<rmv/>
	//defer trace.Trace("leaving scanGitInit") //<rmv/>
	_, eof := d.nextline()
	if eof == nil {
		//trace.Trace("git init ", cur) //<rmv/>
		d.ast.push(astNode{nest: d.nest, cmd: CmdGitInit, cmdParams: true})
		return scanCurrentLevel
	} else {
		d.setError(fmt.Errorf("unexpected EOF at line %d", d.line))
	}

	return nil
} //</rgn scanGitInit>

//------Utility functions ------

//─────────────┤ stripParens ├─────────────

func stripParens(line string) string {
	//var trace = trace.New(os.Stderr)                                         //<rmv/>
	//trace.Trace("----------------------------entering stripParens\n")        //<rmv/>
	//defer trace.Trace("\n----------------------------leaving stripParens\n") //<rmv/>
	m := strings.Index(line, "(")
	if m >= 0 {
		line = line[:m] + line[m+1:]
		//trace.Trace("strip first ( leaves", line) //<rmv/>
	}
	n := strings.LastIndex(line, ")")
	if n > 1 {
		line = line[:n] + line[n+1:]
	}
	//trace.Trace("strip all ( ) leaves", line) //<rmv/>
	return line
}

//<rgn scanToClose>───────────────────────────────────
// scanToClose expects to receive the remaining text starting at the char
// after the keyword to eof
func scanToClose(d *designParser) (int, bool) {
	//var trace = trace.New(os.Stderr)         //<rmv/>
	//trace.Trace("entering scanToClose")      //<rmv/>
	//defer trace.Trace("leaving scanToClose") //<rmv/>
	parens := 0
	var r regexStat
	found := false
	n := 1

	for _, l := range d.text[d.line:] {
		r = regexStatFromPat(OpenPattern, l)
		if r.length > 0 {
			found = true
			parens++
		}

		r = regexStatFromPat(ClosePattern, l)
		if r.length > 0 {
			parens--
			if parens == 0 {
				//trace.Trace("lines between ()", d.text[d.line:d.line+n]) //<rmv/>
				return n, true
			}
		}
		n++
		if d.regexs[EndPattern].MatchString(l) {
			break
		}
	}

	if parens != 0 {
		return n, false
	}

	if !found {
		n = -1
	}
	return n, true
} //</rgn scanToClose>


