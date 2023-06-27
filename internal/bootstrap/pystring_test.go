package bootstrap

import (
	"testing"
)

func Test_evalPythonStringValue(t *testing.T) {
	cases := []struct {
		name      string
		in        string
		out       string
		expectErr bool
	}{
		{
			name:      "double quoted string",
			in:        `"hello'world'"`,
			out:       "hello'world'",
			expectErr: false,
		},
		{
			name:      "single quoted string",
			in:        `'hello"world"'`,
			out:       "hello\"world\"",
			expectErr: false,
		},
		{
			name:      "raw string + double quoted",
			in:        `r"hello\'world'"`,
			out:       "hello\\'world'",
			expectErr: false,
		},
		{
			name:      "raw string + single quoted",
			in:        `r'hello\"world"'`,
			out:       "hello\\\"world\"",
			expectErr: false,
		},
		{
			name:      "unicode string",
			in:        `"你好世界👨‍👩‍👦"`,
			out:       "你好世界👨‍👩‍👦",
			expectErr: false,
		},
		{
			name:      "regex",
			in:        `r"or[@a-z][a-z_0-9\.\[\]\"'-]"`,
			out:       "or[@a-z][a-z_0-9\\.\\[\\]\\\"'-]",
			expectErr: false,
		},
	}

	for idx := range cases {
		c := cases[idx]
		t.Run(c.name, func(t *testing.T) {
			out, err := evalPythonStringValue(c.in)
			if err != nil && !c.expectErr {
				t.Errorf("unexpected error: %v", err)
			}
			if err == nil && c.expectErr {
				t.Errorf("expected error, got nil")
			}
			if out != c.out {
				t.Errorf("expected %s, got %s", c.out, out)
			}
		})
	}
}
