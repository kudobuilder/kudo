package prompt

import (
	"strings"

	"github.com/manifoldco/promptui"
)

// WithOptions prompts for option selection, first element in slice is default
func WithOptions(label string, options []string, addLabel string) (string, error) {

	// addLabel allows control to add more than what is in the list
	allowOther := addLabel != ""
	if allowOther {
		var err error
		var result string
		index := -1
		for index < 0 {
			prompt := promptui.SelectWithAdd{
				Label:    label,
				Items:    options,
				AddLabel: addLabel,
			}

			index, result, err = prompt.Run()
			if index == -1 {
				// lets not force reselection, just return the enter value
				return strings.TrimSpace(result), nil
			}
		}

		if err != nil {
			return "", err
		}
		return strings.TrimSpace(result), nil
	}

	prompt := promptui.Select{
		Label: label,
		Items: options,
	}

	_, result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil

}

// input is output rune, in other words, no cursor
func cursor(input []rune) []rune {
	return input
}

// WithDefault prompts for a response to a label
func WithDefault(label string, defaultStr string) (string, error) {
	return WithValidator(label, defaultStr, nil)
}

// WithValidator prompts for a response to a label with a validation function
func WithValidator(label string, defaultStr string, validate promptui.ValidateFunc) (string, error) {
	prompt := promptui.Prompt{
		Label:    label,
		Default:  defaultStr,
		Validate: validate,
		Pointer:  cursor,
	}
	result, err := prompt.Run()

	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result), nil
}

// Confirm prompts for Y/N question with label and returns true or false for confirmation
func Confirm(label string) bool {
	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil {
		return false
	}
	return strings.ToLower(result) == "y"
}
