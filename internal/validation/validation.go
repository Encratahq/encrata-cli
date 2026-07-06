package validation

import "fmt"

func Timeout(timeout int) error {
	if timeout < 0 || timeout > 60000 {
		return fmt.Errorf("timeout must be between 0 and 60000 milliseconds")
	}
	return nil
}

func Threshold(threshold float64) error {
	if threshold < 0 || threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}
	return nil
}
