package SimpleCLI

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
)

type ConsoleCfg struct {
	Logger            *log.Logger
	Running           bool
	Commands          map[string]func([]string) error
	mutex             sync.Mutex
	OverwriteCommands bool
}

// RegisterCommand registers a new command with its associated function. The function signature must take a slice of strings as arguments and return an error.
// If the command already exists and OverwriteCommands is false, it will skip registration and log a warning.
// If OverwriteCommands is true, it will overwrite the existing command.
// This method is thread-safe and will not allow command registration while the console is running.
func (cfg *ConsoleCfg) RegisterCommand(command string, commandFunc func([]string) error) {
	// Check if the command list is locked
	if cfg.Running {
		cfg.Logger.Println("[WARN] For thread safety, commands cannot be registered while the console is running")
		cfg.Logger.Println("[WARN] If you're seeing this message you might be trying to register commands after starting the console")
		return
	}
	if !cfg.mutex.TryLock() {
		cfg.Logger.Println("[WARN] If you're seeing this message you might be trying to register commands from multiple threads")
		cfg.Logger.Println("[WARN] For thread safety please refrain from doing so")
		return
	}
	defer cfg.mutex.Unlock()

	if _, exists := cfg.Commands[command]; exists {
		if !cfg.OverwriteCommands {
			cfg.Logger.Printf("\t|--\tCommand %s already registered, skipping", command)
			return
		}
		cfg.Logger.Printf("\t|--\tCommand %s already registered, overwriting", command)
	}
	cfg.Logger.Printf("\t|--\tRegistering command %s", command)
	cfg.Commands[command] = commandFunc
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
		Commands:          make(map[string]func([]string) error),
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
		}
		return nil
	})

	cfg.RegisterCommand("stop", func(args []string) error {
		cfg.Logger.Print("Received stop command via console")
		fmt.Println("Stopping application...")
		cfg.Running = false
		return nil
	})

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
				err := cmdFunc(args[1:])
				if err != nil {
					fmt.Println("Error executing command:", err)
				}
			} else {
				fmt.Println("Unknown command:", args)
				err := cfg.Commands["help"](nil)
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
