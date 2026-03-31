package ui

import (
	"errors"
	"testing"
)

func TestWithSpinnerSuccess(t *testing.T) {
	called := false
	err := WithSpinner("Working...", func() error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("WithSpinner returned error: %v", err)
	}
	if !called {
		t.Error("function was not called")
	}
}

func TestWithSpinnerError(t *testing.T) {
	want := errors.New("something failed")
	got := WithSpinner("Working...", func() error {
		return want
	})
	if !errors.Is(got, want) {
		t.Errorf("WithSpinner error = %v, want %v", got, want)
	}
}
