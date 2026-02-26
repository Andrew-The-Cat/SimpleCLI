package SimpleCLI

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type Flag struct {
	Name         string
	ExpectsValue bool
}

type Command struct {
	Name  string
	Exec  func([]string, map[string]string) error
	Desc  string
	Flags []Flag
}

type ConsoleCfg struct {
	Logger            *log.Logger
	Running           bool
	Commands          map[string]Command
	mutex             sync.Mutex
	OverwriteCommands bool
}

// NewCommandRegister returns a new Command that can be put into a cfg.Register
func NewCommandRegister(name string, exec func([]string, map[string]string) error) *Command {
	return &Command{Name: name, Exec: exec}
}

// WithDescription adds a description to the command
func (c *Command) WithDescription(description string) *Command {
	c.Desc = description
	return c
}

// WithFlag adds a flag to the command for parsing during execution.
// expectsValue determines if the argument right after the flag will be parsed together with it
// Example -p 8080 wil resolve in the function flags parameter as flags["p"] = 8080
func (c *Command) WithFlag(controlString string, expectsValue bool) *Command {
	c.Flags = append(c.Flags, Flag{controlString, expectsValue})
	return c
}

func (cfg *ConsoleCfg) Register(c Command) error {
	// Check if the command list is locked
	if cfg.Running {
		cfg.Logger.Println("[WARN] For thread safety, commands cannot be registered while the console is running")
		cfg.Logger.Println("[WARN] If you're seeing this message you might be trying to register commands after starting the console")
		return nil
	}
	if !cfg.mutex.TryLock() {
		cfg.Logger.Println("[WARN] If you're seeing this message you might be trying to register commands from multiple threads")
		cfg.Logger.Println("[WARN] For thread safety please refrain from doing so")
		return nil
	}
	defer cfg.mutex.Unlock()

	if _, exists := cfg.Commands[c.Name]; exists {
		if !cfg.OverwriteCommands {
			cfg.Logger.Printf("\t|--\tCommand %s already registered, skipping", c.Name)
			return nil
		}
		cfg.Logger.Printf("\t|--\tCommand %s already registered, overwriting", c.Name)
	}
	cfg.Logger.Printf("\t|--\tRegistering command %s", c.Name)
	if len(c.Name) == 0 {
		return fmt.Errorf("command name is required")
	}
	cfg.Commands[c.Name] = c
	return nil
}

// RegisterCommandWithDescription registers a new command with its associated function. The function signature must take a slice of strings as arguments and return an error.
// If the command already exists and OverwriteCommands is false, it will skip registration and log a warning.
// If OverwriteCommands is true, it will overwrite the existing command.
// This method is thread-safe and will not allow command registration while the console is running.
// @Deprecated - command is deprecated in favor of cfg.Register(NewCommandRegister(name, exec))
func (cfg *ConsoleCfg) RegisterCommandWithDescription(command string, commandFunc func(args []string) error, description string) {
	c := NewCommandRegister(command, func(exec func([]string) error) func([]string, map[string]string) error {
		return func(args []string, flags map[string]string) error {
			return exec(args)
		}
	}(commandFunc)).WithDescription(description)

	err := cfg.Register(*c)
	if err != nil {
		fmt.Printf("Error registering command '%s': %s\n", command, err)
	}
}

// RegisterCommand is a convenience method for registering commands without a description. It calls RegisterCommandWithDescription with an empty description.
// @Deprecated - command is deprecated in favor of cfg.Register(NewCommandRegister(name, exec))
func (cfg *ConsoleCfg) RegisterCommand(command string, commandFunc func(args []string) error) {
	cfg.RegisterCommandWithDescription(command, commandFunc, "")
}

// NewConsoleCfg creates a new ConsoleCfg instance that will manage console commands.
// If overwriteCommands is true, registering a command that already exists will overwrite it.
// If false, it will skip registering the command and log a warning.
// The logger parameter is used for logging console activities. If nil, a default logger will be created on stdout.
func NewConsoleCfg(logger *log.Logger, overwriteCommands bool) *ConsoleCfg {
	return &ConsoleCfg{
		Logger: func() *log.Logger {
			if logger != nil {
				return logger
			}
			return log.New(os.Stdout, "CONSOLE: ", log.Ldate|log.Ltime|log.Lshortfile)
		}(),
		Running:           false,
		Commands:          make(map[string]Command),
		mutex:             sync.Mutex{},
		OverwriteCommands: overwriteCommands,
	}
}

// StartConsole starts the console interface, allowing users to input commands.
// It registers default commands like "help" and "stop".
// The console runs in a separate goroutine and listens for user input until the Running flag is set to false.
// When the console stops it sends a signal to the provided chanStop channel.
func (cfg *ConsoleCfg) StartConsole(chanStop chan struct{}) {
	// Register default commands
	cfg.RegisterCommand("help", func(args []string) error {
		fmt.Println("Available Commands:")
		for cmd := range cfg.Commands {
			fmt.Println(" -", cmd)
			if desc := cfg.Commands[cmd].Desc; desc != "" {
				fmt.Printf("\t|--\t%s\n", desc)
			}
		}
		return nil
	})

	// Stop command; when executed will return a signal in the chanStop channel
	cfg.RegisterCommandWithDescription("stop", func(args []string) error {
		cfg.Logger.Print("Received stop command via console")
		fmt.Println("Stopping application...")
		cfg.Running = false
		return nil
	}, "Stops the console and signals the application to stop")

	// Console mode for imputing Commands
	cfg.Logger.Print("Starting console...")
	fmt.Println("Starting console...")

	go func() {
		cfg.mutex.Lock()
		cfg.Running = true
		reader := bufio.NewReader(os.Stdin)
		for cfg.Running {
			fmt.Print(">> ")
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println("Error reading command:", err)
				continue
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			args := strings.Split(line, " ")

			if cmdFunc, exists := cfg.Commands[args[0]]; exists {
				flags, newArgs, err := extractFlags(args[1:], cmdFunc)
				if err != nil {
					fmt.Println("Error parsing flags:", err)
					continue
				}
				err = cmdFunc.Exec(newArgs, flags)
				if err != nil {
					fmt.Println("Error executing command:", err)
				}
			} else {
				fmt.Println("Unknown command:", args)
				err := cfg.Commands["help"].Exec(nil, nil)
				if err != nil {
					continue
				}
			}
		}
		cfg.mutex.Unlock()
		cfg.Logger.Print("Console stopped.")
		fmt.Println("Console stopped.")
		chanStop <- struct{}{}
	}()
}

/*
=======================

     Helpers

=======================
*/

func parseFlag(argument string, possibleFlags []Flag) (Flag, error) {
	if len(argument) == 0 {
		return Flag{}, fmt.Errorf("no argument specified")
	}

	if !strings.HasPrefix(argument, "-") {
		return Flag{}, fmt.Errorf("given argument does not have flag format")
	}

	argument = argument[1:]

	for _, flag := range possibleFlags {
		if flag.Name == argument {
			return flag, nil
		}
	}

	return Flag{}, fmt.Errorf("unknown argument: %s", argument)
}

func extractFlags(args []string, cmdFunc Command) (map[string]string, []string, error) {
	flags := make(map[string]string)
	newArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		argument := args[i]
		flag, err := parseFlag(argument, cmdFunc.Flags)
		if err != nil {
			newArgs = append(newArgs, argument)
			continue
		}
		if flag.ExpectsValue {
			if i == len(args)-1 {
				return nil, nil, fmt.Errorf("no value specified for flag that expects value: %v", flag.Name)
			}

			flags[flag.Name] = args[i+1]
			i++
		} else {
			flags[flag.Name] = ""

		}
	}
	return flags, newArgs, nil
}
