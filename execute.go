package goproject

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bitfield/script"
	path "github.com/rhysd/abspath"
	"github.com/westarver/helper"
)

//─────────────┤ executeAst ├─────────────

func executeAst(p *designParser) error {
	//var trace = trace.New(os.Stderr) //<rmv/>
	//trace.Trace("----------------------------entering executeAst")      //<rmv/>
	//defer trace.Trace("----------------------------leaving executeAst") //<rmv/>
	for _, n := range p.ast.q {
		//trace.Trace("executing node ", i, " ", n) //<rmv/>
		runCommand(p, n)
	}
	return nil
}

//─────────────┤ runCommand ├─────────────

func runCommand(p *designParser, an astNode) error {
	//var trace = trace.New(os.Stderr) //<rmv/>

	switch an.cmd {
	case CmdExec:
		execCmd(an)
	case CmdDir:
		dir := an.cmdParams.(path.AbsPath).String()
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			p.setError(fmt.Errorf("error creating directory %s", dir))
			return err
		}
	case CmdCopy:
		// dest could be a dir, but if src is a single file then a single file is ok
		dest := an.nest.path.String()
		// src could be a single file, a list of files or a directory
		src := an.cmdParams.(string)

		if err, ok := copyDir(dest, src); ok {
			//trace.Trace("is src dir? ", err) //<rmv/>
			return err
		}
		// split the source string into individual names respecting quoted strings
		slice := splitWithQuotes(src)
		for _, s := range slice {
			s = strings.Trim(s, "\t ")
			if len(s) == 0 {
				continue
			}
			//trace.Trace("source ", s) //<rmv/>
			dst := filepath.Join(dest, s)
			//trace.Trace("dest ", dst) //<rmv/>
			err := CopyFileStr(dst, s)
			if err != nil {
				p.setError(fmt.Errorf("error copying %s to %s", s, dest))
				//trace.Trace(err) //<rmv/>
				return err
			}
		}
	case CmdGet:
		wd, err := path.Getwd()
		if err != nil {
			return err
		}
		//trace.Trace("original wd ", wd.String()) //<rmv/>
		err = os.Chdir(an.nest.path.String())
		if err != nil {
			p.setError(fmt.Errorf("error downloading URL %s", an.cmdParams.(string)))
			return err
		}

		//cd, _ := path.Getwd()  //<rmv/>
		//trace.Trace("new wd ", cd.String())                //<rmv/>
		//trace.Trace("downloading ", an.cmdParams.(string)) //<rmv/>
		pipe := script.Exec("wget " + an.cmdParams.(string))
		if pipe.Error() != nil {
			p.setError(pipe.Error())
		}
		pipe.Stdout()

		err = os.Chdir(wd.String())
		if err != nil {
			p.setError(fmt.Errorf("error downloading %s", an.cmdParams.(string)))
			return err
		}

	case CmdModule:
		wd, err := path.Getwd()
		if err != nil {
			return err
		}
		//trace.Trace("original wd ", wd.String()) //<rmv/>
		err = os.Chdir(an.nest.path.String())
		if err != nil {
			p.setError(fmt.Errorf("error initializing module %s", an.cmdParams.(string)))
			return err
		}

		//cd, _ := path.Getwd()  //<rmv/>
		//trace.Trace("new wd ", cd.String()) //<rmv/>

		pipe := script.Exec("go mod init " + an.cmdParams.(string))
		if pipe.Error() != nil {
			p.setError(pipe.Error())
		}
		pipe.Stdout()

		err = os.Chdir(wd.String())
		if err != nil {
			p.setError(fmt.Errorf("error initializing module %s", an.cmdParams.(string)))
			return err
		}
	case CmdWorkspace:
		wd, err := path.Getwd()
		if err != nil {
			return err
		}
		//trace.Trace("initial wd ", wd.String()) //<rmv/>
		err = os.Chdir(an.nest.path.String())
		if err != nil {
			p.setError(fmt.Errorf("error initializing workspace %s", an.cmdParams.(string)))
			return err
		}
		//cd, _ := path.Getwd()   //<rmv/>
		//trace.Trace("changed wd ", cd.String()) //<rmv/>
		mods := strings.Split(an.cmdParams.(string), " ")
		if len(mods) > 0 {
			pipe := script.Exec("go work init " + mods[0])
			if pipe.Error() != nil {
				p.setError(fmt.Errorf("error initializing workspace %s", an.cmdParams.(string)))
				return pipe.Error()
			}
			pipe.Stdout()
			for _, m := range mods[1:] {
				pipe := script.Exec("go work use " + m)
				pipe.Stdout()
			}
		}

		err = os.Chdir(wd.String())
		if err != nil {
			p.setError(fmt.Errorf("error initializing workspace %s", an.cmdParams.(string)))
			return err
		}
	case CmdGitInit:
		wd, err := path.Getwd()
		if err != nil {
			return err
		}

		err = os.Chdir(an.nest.path.String())
		if err != nil {
			p.setError(fmt.Errorf("error initializing git repo"))
			return err
		}

		_, _ = script.Exec("git init").Stdout()
		err = os.Chdir(wd.String())
		if err != nil {
			p.setError(fmt.Errorf("error initializing git repo"))
			return err
		}

	}

	return nil
}

//─────────────┤ execCmd ├─────────────

func execCmd(an astNode) error {
	redir := false
	mode := os.O_RDWR | os.O_CREATE
	var file string

	arg := an.cmdParams.(string)

	// hack to honor redirection in command string
	re := regexp.MustCompile(`\s+>{1,2}\s+`)
	loc := re.FindAllStringIndex(arg, -1)
	if loc != nil {
		last := loc[len(loc)-1]
		mat := strings.Trim(arg[last[0]:last[1]], "\t ")
		if len(mat) == 2 {
			mode |= os.O_APPEND
		}
		redir = true
		file = arg[last[1]:]
		arg = arg[:last[0]]
	}
	// end of dirty hack

	p := script.Exec(arg)
	if redir {
		j := filepath.Join(an.nest.path.String(), file)
		if mode&os.O_APPEND == 0 {
			p.WriteFile(j)
		} else {
			p.AppendFile(j)
		}
	} else {
		p.Stdout()
	}
	if p.Error() != nil {
		return p.Error()
	}
	return nil
}

//─────────────┤ copyDir ├─────────────

func copyDir(dst, src string) (error, bool) {
	//var trace = trace.New(os.Stderr)                                   //<rmv/>
	//trace.Trace("----------------------------entering copyDir\n")      //<rmv/>
	//defer trace.Trace("----------------------------leaving copyDir\n") //<rmv/>
	//trace.Trace("dest ", dst)                                          //<rmv/>
	//trace.Trace("src ", src)                                           //<rmv/>
	exist := helper.DirExists(dst)
	if !exist {
		err := os.MkdirAll(dst, 0777)
		if err != nil {
			return err, false
		}
	}
	exist = helper.DirExists(src)
	if exist {
		slice, err := script.FindFiles(src).Slice()
		if err != nil {
			return err, false
		}
		for _, sl := range slice {
			dest := filepath.Join(dst, filepath.Base(sl))
			//trace.Trace("dest ", dest) //<rmv/>
			//trace.Trace("line ", sl)   //<rmv/>
			err := CopyFileStr(dest, sl)
			if err != nil {
				return err, false
			}
		}
		return nil, true
	}

	return nil, false
}

//─────────────┤ splitWithQuotes ├─────────────

func splitWithQuotes(s string) []string {
	var slice = []string{}

	for {
		r := regexStatFromPat(`\"(.)*\"`, s)
		if len(r.before) == 0 && len(r.after) == 0 && r.length == 0 {
			break
		}
		slice = append(slice, strings.Split(r.before, " ")...)
		slice = append(slice, r.match)
		s = r.after
	}

	if len(slice) == 0 {
		slice = append(slice, strings.Split(s, " ")...)
	}

	return slice
}

//─────────────┤ CopyFileStr ├─────────────

func CopyFileStr(dst, src string) error {
	src, err := helper.ValidatePath(src)
	if err != nil {
		return err
	}
	s, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	dst, err = helper.ValidatePath(dst)
	if err != nil {
		return err
	}

	err = os.WriteFile(dst, s, 0666)
	if err != nil {
		return err
	}

	return nil
}
