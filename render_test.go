package formatter

import (
	"os"
	"testing"
)

func TestFormatter(t *testing.T) {
	source, err := os.ReadFile("sample.md")
	if err != nil {
		t.Fatal(err)
	}

	err = Format(source, os.Stdout)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFormatter2(t *testing.T) {
	source := []byte(`
| ddd | ssss |
| --- | ---- |
| ssss | ssss |
{#id .class}
`)

	// STXHeader = false
	err := Format(source, os.Stdout)
	if err != nil {
		t.Fatal(err)
	}
}
