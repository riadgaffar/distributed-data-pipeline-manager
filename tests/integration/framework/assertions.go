package framework

import "testing"

type TestAssertions struct {
	t *testing.T
}

func NewTestAssertions(t *testing.T) *TestAssertions {
	return &TestAssertions{t: t}
}

func (a *TestAssertions) AssertEqual(expected, actual interface{}, message string) {
	if expected != actual {
		a.t.Errorf("%s: expected %v, got %v", message, expected, actual)
	}
}
