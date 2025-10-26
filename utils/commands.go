package utils

import (
	"os"
	"os/exec"
	"strings"
	"fmt"
	"bufio"
	"strconv"
	"errors"
)

func RunCommand(command string, withOutput bool, withError bool) error{
	cmd := exec.Command("sh", "-c", command)
	if withOutput {
		cmd.Stdout = os.Stdout
	}
	if withError {
		cmd.Stderr = os.Stderr
	}
	cmd.Stdin = os.Stdin
	cmdErr := cmd.Run()
	if cmdErr != nil {
		return cmdErr
	}
	return nil
}

func RunCommandWithOutput(command string) (error, string){
	cmd := exec.Command("sh", "-c", command)
	var output strings.Builder
	cmd.Stdout = &output
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmdErr := cmd.Run()
	return cmdErr, output.String()
}

func RunMultipleCommands(commands []string, withOutput bool, withError bool) error{
	for _, command := range commands {
		err := RunCommand(command, withOutput, withError)
		if err != nil {
			return err
		}
	}
	return nil
}

func ReadInput(msg string, data interface{}) error{
	fmt.Print(msg)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	input = strings.TrimSuffix(input, "\n")

	for input == "" {
		PrintError("Input cannot be empty.")
		fmt.Print(msg)
		input, err = reader.ReadString('\n')
		if err != nil {
			return err
		}
	}

	input = strings.TrimSuffix(input, "\n")

	switch v := data.(type) {
	case *string:
		*v = input
	case *bool:
		*v = input == "y"
	case *int:
		*v, err = strconv.Atoi(input)
		return err
	default:
		return errors.New("Invalid data type")
	}
	
	return nil
}