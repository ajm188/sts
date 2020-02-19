package main

import (
	"errors"
)

func NoLoggerInContext() error { return errors.New("No logger found in context.") }
