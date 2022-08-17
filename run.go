package goproject

import (
	"errors"

	"github.com/westarver/boa"
	msg "github.com/westarver/messenger"
)

const (
	DefaultCfgFile = "go-project.design"
)

func Run(writer *msg.Messenger) int {
	var (
		cfg      string
		exitCode int
	)

	cli := boa.FromHelp(getUsage())

	help, hlp := cli.Items["help"].(boa.CmdLineItem[string])
	if hlp {
		topic := help.Value()
		item, exist := cli.AllHelp[topic]
		if exist {
			ShowHelp(writer, item)
		} else {
			ShowHelp(writer)
		}
		return 0
	}

	file, f := cli.Items["--design"].(boa.CmdLineItem[string])
	if f {
		cfg = file.Value()
	} else {
		cfg = DefaultCfgFile
	}
	in, init := cli.Items["init"].(boa.CmdLineItem[string])
	if init {
		name := in.Value()
		parser, err := initProject(name, cfg)
		if err != nil {
			writer.LogMsg(writer.Logout(), 1, parser.Errors()+"\n")
			exitCode = 2
		}
		executeAst(parser)
		if parser.hasErrors() {
			writer.LogMsg(writer.Logout(), 1, parser.Errors()+"\n")
		}
	}
	_, ren := cli.Items["rename"].(boa.CmdLineItem[string])
	if ren {
		// TODO: implement rename later
		//doRename(cfg)
		writer.Catch(msg.LOG, errors.New("rename command not implemented"))
		exitCode = 0
	}

	if !hlp && !init && !ren { // default command is help
		ShowHelp(writer)
		return 0
	}

	writer.InfoMsg(writer.Logout(), msg.MESSAGE, "Exiting with exit code %d", exitCode)
	return exitCode
}
