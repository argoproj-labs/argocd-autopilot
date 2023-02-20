package util

import (
	"reflect"
	"testing"
)

func TestCleanSliceWhiteSpaces(t *testing.T) {
	testcases := []struct {
		input []string
		expected []string
	}{
		{
			input: []string{"org", "repo"},
			expected: []string{"org", "repo"},
		},
		{
			input: []string{"org", ""},
			expected: []string{"org"},
		},
	}

	for _, testcase := range testcases {
		res := CleanSliceWhiteSpaces(testcase.input)
		if !reflect.DeepEqual(res, testcase.expected) {
			t.Errorf("slice expected to be %v, but got %v", testcase.expected, res)
		}
	}
}
