package utils

import (
	"os"
	"os/exec"
	"strings"
	"fmt"
	"bufio"
)

func RunCommand(command string, withOutput bool, withError bool) error{
	cmd := exec.Command("sh", "-c", command)
	if withOutput {
		cmd.Stdout = os.Stdout
	}
	if withError {
		cmd.Stderr = os.Stderr
	}
	cmdErr := cmd.Run()
	if cmdErr != nil {
		return cmdErr
	}
	return nil
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

func ReadInput(msg string, notNull bool) (string, error){
	fmt.Print(msg)
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	input = strings.TrimSuffix(input, "\n")

	for notNull && input == "" {
		fmt.Println("Input cannot be empty.")
		fmt.Print(msg)
		input, err = reader.ReadString('\n')
		if err != nil {
			return "", err
		}
	}

	input = strings.TrimSuffix(input, "\n")

	return input, nil

}