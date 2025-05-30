package log

import (
	"fmt"
	"time"

	"github.com/briandowns/spinner"
)

// WithSpinner executes the given function while showing a spinner with the specified message.
func WithSpinner(message string, fn func() error) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + message

	err := s.Color("green")
	if err != nil {
		return fmt.Errorf("coloring green: %w", err)
	}

	s.Start()
	s.FinalMSG = message + " \033[32m[done]\033[0m"
	defer s.Stop()

	return fn()
}
