package command

import (
	"reflect"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	parser := NewParser()

	// Register test command
	spec := &CommandSpec{
		Name:    "test",
		MinArgs: 1,
		MaxArgs: 3,
		Flags: map[string]FlagSpec{
			"verbose": {Type: "bool", Short: "v"},
			"count":   {Type: "int", Short: "c"},
			"output":  {Type: "string", Short: "o"},
			"rate":    {Type: "float"},
		},
	}

	err := parser.RegisterCommand(spec)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		want    *Command
		wantErr bool
	}{
		{
			name:  "simple command with args",
			input: "test arg1 arg2",
			want: &Command{
				Name:     "test",
				Args:     []string{"arg1", "arg2"},
				Flags:    map[string]interface{}{},
				RawInput: "test arg1 arg2",
			},
			wantErr: false,
		},
		{
			name:  "command with boolean flag",
			input: "test arg1 --verbose",
			want: &Command{
				Name:     "test",
				Args:     []string{"arg1"},
				Flags:    map[string]interface{}{"verbose": true},
				RawInput: "test arg1 --verbose",
			},
			wantErr: false,
		},
		{
			name:  "command with short flag",
			input: "test arg1 -v",
			want: &Command{
				Name:     "test",
				Args:     []string{"arg1"},
				Flags:    map[string]interface{}{"verbose": true},
				RawInput: "test arg1 -v",
			},
			wantErr: false,
		},
		{
			name:  "command with string flag",
			input: "test arg1 --output file.txt",
			want: &Command{
				Name:     "test",
				Args:     []string{"arg1"},
				Flags:    map[string]interface{}{"output": "file.txt"},
				RawInput: "test arg1 --output file.txt",
			},
			wantErr: false,
		},
		{
			name:  "command with int flag",
			input: "test arg1 --count 42",
			want: &Command{
				Name:     "test",
				Args:     []string{"arg1"},
				Flags:    map[string]interface{}{"count": 42},
				RawInput: "test arg1 --count 42",
			},
			wantErr: false,
		},
		{
			name:  "command with float flag",
			input: "test arg1 --rate 3.14",
			want: &Command{
				Name:     "test",
				Args:     []string{"arg1"},
				Flags:    map[string]interface{}{"rate": 3.14},
				RawInput: "test arg1 --rate 3.14",
			},
			wantErr: false,
		},
		{
			name:  "command with quoted argument",
			input: `test "quoted arg" 'single quoted'`,
			want: &Command{
				Name:     "test",
				Args:     []string{"quoted arg", "single quoted"},
				Flags:    map[string]interface{}{},
				RawInput: `test "quoted arg" 'single quoted'`,
			},
			wantErr: false,
		},
		{
			name:  "command with combined short flags",
			input: "test arg1 -vc 10",
			want: &Command{
				Name:     "test",
				Args:     []string{"arg1"},
				Flags:    map[string]interface{}{"verbose": true, "count": 10},
				RawInput: "test arg1 -vc 10",
			},
			wantErr: false,
		},
		{
			name:    "too few arguments",
			input:   "test",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "too many arguments",
			input:   "test arg1 arg2 arg3 arg4",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unknown command",
			input:   "unknown arg1",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unknown flag",
			input:   "test arg1 --unknown",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "unterminated quote",
			input:   `test "unterminated`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != nil {
				// Clear position for comparison
				got.Position = 0
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("Parser.Parse() = %+v, want %+v", got, tt.want)
				}
			}
		})
	}
}

func TestParser_Tokenize(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name    string
		input   string
		want    []Token
		wantErr bool
	}{
		{
			name:  "simple tokens",
			input: "cmd arg1 arg2",
			want: []Token{
				{Type: "arg", Value: "cmd", Position: 0},
				{Type: "arg", Value: "arg1", Position: 4},
				{Type: "arg", Value: "arg2", Position: 9},
			},
			wantErr: false,
		},
		{
			name:  "flags",
			input: "cmd --long -s",
			want: []Token{
				{Type: "arg", Value: "cmd", Position: 0},
				{Type: "long_flag", Value: "--long", Position: 4},
				{Type: "short_flag", Value: "-s", Position: 11},
			},
			wantErr: false,
		},
		{
			name:  "quoted strings",
			input: `cmd "quoted arg" 'single'`,
			want: []Token{
				{Type: "arg", Value: "cmd", Position: 0},
				{Type: "arg", Value: "quoted arg", Position: 4},
				{Type: "arg", Value: "single", Position: 17},
			},
			wantErr: false,
		},
		{
			name:  "escaped characters",
			input: `cmd "escaped \"quote\""`,
			want: []Token{
				{Type: "arg", Value: "cmd", Position: 0},
				{Type: "arg", Value: `escaped "quote"`, Position: 4},
			},
			wantErr: false,
		},
		{
			name:    "unterminated quote",
			input:   `cmd "unterminated`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.tokenize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.tokenize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parser.tokenize() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParser_GetCompletions(t *testing.T) {
	parser := NewParser()

	// Register test commands
	specs := []*CommandSpec{
		{Name: "download", Description: "Download data"},
		{Name: "delete", Description: "Delete data"},
		{Name: "config", Description: "Configure system"},
	}

	for _, spec := range specs {
		err := parser.RegisterCommand(spec)
		if err != nil {
			t.Fatalf("Failed to register command: %v", err)
		}
	}

	tests := []struct {
		name    string
		partial string
		want    []string
	}{
		{
			name:    "no partial",
			partial: "",
			want:    []string{"download", "delete", "config"},
		},
		{
			name:    "partial match",
			partial: "d",
			want:    []string{"download", "delete"},
		},
		{
			name:    "single match",
			partial: "dow",
			want:    []string{"download"},
		},
		{
			name:    "no match",
			partial: "xyz",
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parser.GetCompletions(tt.partial)
			if !slicesEqual(got, tt.want) {
				t.Errorf("Parser.GetCompletions() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCommandSpec_Validation(t *testing.T) {
	parser := NewParser()

	// Register command with required flag
	spec := &CommandSpec{
		Name:    "test",
		MinArgs: 1,
		MaxArgs: 2,
		Flags: map[string]FlagSpec{
			"required": {Type: "string", Required: true},
			"optional": {Type: "int", Default: 42},
		},
	}

	err := parser.RegisterCommand(spec)
	if err != nil {
		t.Fatalf("Failed to register command: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "valid command with required flag",
			input:   "test arg1 --required value",
			wantErr: false,
		},
		{
			name:    "missing required flag",
			input:   "test arg1",
			wantErr: true,
		},
		{
			name:    "command with default flag",
			input:   "test arg1 --required value --optional 100",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parser.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parser.Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Helper function to compare slices without order
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	countA := make(map[string]int)
	countB := make(map[string]int)

	for _, item := range a {
		countA[item]++
	}

	for _, item := range b {
		countB[item]++
	}

	return reflect.DeepEqual(countA, countB)
}
