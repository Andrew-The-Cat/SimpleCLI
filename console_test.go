package SimpleCLI

import (
	"fmt"
	"log"
	"os"
	"testing"
)

func TestHook(t *testing.T) {
	t.Run("TestRegisterCommand", TestRegisterCommand)
	t.Run("TestNewConsoleCfg", TestNewConsoleCfg)
	t.Run("TestRegisterCommandWhileRunning", TestRegisterCommandWhileRunning)
	t.Run("TestRegisterCommandConcurrency", TestRegisterCommandConcurrency)
	t.Run("TestExtractFlags", TestExtractFlags)
}

func TestRegisterCommand(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.Ldate|log.Ltime|log.Lshortfile)
	console := NewConsoleCfg(logger, false)

	// Register a command
	console.RegisterCommand("test", func(args []string) error {
		fmt.Println("Test command executed")
		return nil
	})

	// Attempt to register the same command without overwrite
	console.RegisterCommand("test", func(args []string) error {
		fmt.Println("This should not be registered")
		return nil
	})

	if len(console.Commands) != 1 {
		t.Errorf("Expected 1 command, got %d", len(console.Commands))
	}

	// Now test with overwrite enabled
	console.OverwriteCommands = true
	console.RegisterCommand("test", func(args []string) error {
		fmt.Println("Test command overwritten and executed")
		return nil
	})
	if len(console.Commands) != 1 {
		t.Errorf("Expected 1 command after overwrite, got %d", len(console.Commands))
	}
}

func TestNewConsoleCfg(t *testing.T) {
	// Test with nil logger
	console := NewConsoleCfg(nil, false)
	if console.Logger == nil {
		t.Error("Expected default logger to be created, got nil")
	}

	// Test with custom logger
	customLogger := log.New(os.Stdout, "CUSTOM: ", log.Ldate|log.Ltime|log.Lshortfile)
	console = NewConsoleCfg(customLogger, false)
	if console.Logger != customLogger {
		t.Error("Expected custom logger to be set")
	}
}

func TestRegisterCommandWhileRunning(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.Ldate|log.Ltime|log.Lshortfile)
	console := NewConsoleCfg(logger, false)
	console.Running = true

	// Attempt to register a command while running
	console.RegisterCommand("runningTest", func(args []string) error {
		fmt.Println("This should not be registered")
		return nil
	})

	if len(console.Commands) != 0 {
		t.Errorf("Expected 0 commands while running, got %d", len(console.Commands))
	}
}

func TestRegisterCommandConcurrency(t *testing.T) {
	logger := log.New(os.Stdout, "TEST: ", log.Ldate|log.Ltime|log.Lshortfile)
	console := NewConsoleCfg(logger, false)

	// Simulate concurrent registration
	done := make(chan bool)
	go func() {
		console.RegisterCommand("concurrentTest", func(args []string) error {
			fmt.Println("This should be registered once")
			return nil
		})
		done <- true
	}()
	go func() {
		console.RegisterCommand("concurrentTest", func(args []string) error {
			fmt.Println("This should be registered once")
			return nil
		})
		done <- true
	}()

	<-done
	<-done

	if len(console.Commands) != 1 {
		t.Errorf("Expected 1 command after concurrent registration, got %d", len(console.Commands))
	}
}

func TestExtractFlags(t *testing.T) {
	args := []string{"-h", "-p", "8080"}
	cmd := NewCommandRegister("sth", func(strings []string, m map[string]string) error {
		return nil
	}).WithFlag("h", false).WithFlag("p", true)

	res, _, err := extractFlags(args, *cmd)
	if err != nil {
		t.Fatal(err)
	}

	if len(res) != 2 {
		t.Fatalf("Expected 2 flags, got %d", len(res))
	}
	if res["h"] != "" {
		t.Fatalf("Expected h to be empty, got %s", res["h"])
	}
	if res["p"] != "8080" {
		t.Fatalf("Expected p to be 8080, got %s", res["p"])
	}

	args = []string{"-p", "-h"}
	res, _, err = extractFlags(args, *cmd)
	if err != nil {
		t.Fatal(err)
	}

	if res["p"] != "-h" {
		t.Fatalf("Expected p to be -h, got %s", res["p"])
	}

	args = []string{"-p"}
	_, _, err = extractFlags(args, *cmd)
	if err == nil {
		t.Fatalf("Expected extraction to fail due to -p requiring a value")
	}
}
