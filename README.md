# go-project

## Usage: go-project [command] [flag...]
	
## Commands:     
[help] <topic>           : Show help and exit, default command

[init] <name>            : Create a project root directory with optional sub directories.
		
## Flags:				
[--design | -d] <design> : File where project details are given. 

## Description:
## init:      
>The init command with name will create a minimal project with only a root directory and a
>readme file. More functionality can be had by using a design file. See the 'example.design'
>file in this repository. To use init without following with a name use -- for name.
>If no name is given, (eg. --) then a name must appear in the design file.

## --design:
>The project name is the minimum requirement, but can be followed by sub directories with optional 
>content specified for each one. Some content may include a license, a .gitignore file,
>or any file you want copied into the directory. 
>External commands can be executed and nearly any structure you want can be created. A workspace
>and/or module can be initiated as well.
>The --design flag is optional, if not given the app will look for a file named 'go-project.design'
>in the current directory. If that is not found then the program will exit with an exit code of 2.

