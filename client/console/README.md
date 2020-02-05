## Console

The `console` package contains the code for the console 'per se'.

* `console.go`          - Console instantiation and setup (using chzyer/readline package)
* `prompt.go`           - Code for dynamic display/refresh of the prompt, with variable substitution.
* `exec.go`             - Dispatch and execution of commands.
* `console-connect.go`  - Function for connecting to server and binding commands/events.
* `console-context.go`  - Initialize shell context/variables and makes them available to commands.
