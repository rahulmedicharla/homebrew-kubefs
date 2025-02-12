package utils

import (
	"os"
	"os/exec"
	"strings"
	"fmt"
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

func ReadInput(msg string) (string, error){
	var input string
	fmt.Print(msg)
	_, err := fmt.Scanln(&input)
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)
	return input, nil

}