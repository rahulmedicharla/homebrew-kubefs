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

func RemoveHost(ipAddress string, domain string) error{
	PrintWarning("Removing host entry from /etc/hosts file. This requires sudo permissions.")
	
	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command("sudo", bin, "remove", "host-entry", ipAddress, domain)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run command with sudo: %w", err)
	}
	return nil
}

func AddHost(ipAddress string, domain string) error{
	PrintWarning("Adding host entry to /etc/hosts file. This requires sudo permissions.")

	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	cmd := exec.Command("sudo", bin, "create", "host-entry", ipAddress, domain)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run command with sudo: %w", err)
	}
	return nil
	
}

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