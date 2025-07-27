package command

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// Command represents a parsed command with arguments and flags
type Command struct {
	Name     string                 `json:"name"`
	Args     []string               `json:"args"`
	Flags    map[string]interface{} `json:"flags"`
	RawInput string                 `json:"raw_input"`
	Position int                    `json:"position"` // Position in input for error reporting
}

// FlagSpec defines the specification for a command flag
type FlagSpec struct {
	Type        string `json:"type"`        // "string", "int", "bool", "float"
	Short       string `json:"short"`       // Short flag name (e.g., "v" for -v)
	Description string `json:"description"` // Help text for flag
	Default     interface{} `json:"default"` // Default value
	Required    bool   `json:"required"`    // Whether flag is required
}

// CommandSpec defines the specification for a command
type CommandSpec struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Usage       string              `json:"usage"`
	Category    string              `json:"category"`
	Aliases     []string            `json:"aliases"`
	MinArgs     int                 `json:"min_args"`
	MaxArgs     int                 `json:"max_args"` // -1 for unlimited
	Flags       map[string]FlagSpec `json:"flags"`
	Examples    []string            `json:"examples"`
}

// Parser handles command parsing with advanced features
type Parser struct {
	specs map[string]*CommandSpec
}

// NewParser creates a new command parser
func NewParser() *Parser {
	return &Parser{
		specs: make(map[string]*CommandSpec),
	}
}

// RegisterCommand registers a command specification
func (p *Parser) RegisterCommand(spec *CommandSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("command name cannot be empty")
	}
	
	if _, exists := p.specs[spec.Name]; exists {
		return fmt.Errorf("command %s already registered", spec.Name)
	}
	
	// Register main command name
	p.specs[spec.Name] = spec
	
	// Register aliases
	for _, alias := range spec.Aliases {
		if _, exists := p.specs[alias]; exists {
			return fmt.Errorf("alias %s conflicts with existing command", alias)
		}
		p.specs[alias] = spec
	}
	
	return nil
}

// Parse parses a command string into a Command struct
func (p *Parser) Parse(input string) (*Command, error) {
	if strings.TrimSpace(input) == "" {
		return nil, fmt.Errorf("empty command")
	}
	
	tokens, err := p.tokenize(input)
	if err != nil {
		return nil, fmt.Errorf("tokenization error: %w", err)
	}
	
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no command found")
	}
	
	cmd := &Command{
		Name:     tokens[0].Value,
		Args:     []string{},
		Flags:    make(map[string]interface{}),
		RawInput: input,
		Position: tokens[0].Position,
	}
	
	spec, exists := p.specs[cmd.Name]
	if !exists {
		return cmd, fmt.Errorf("unknown command: %s", cmd.Name)
	}
	
	return p.parseWithSpec(cmd, tokens[1:], spec)
}

// Token represents a parsed token
type Token struct {
	Type     string // "arg", "flag", "value"
	Value    string
	Position int
}

// tokenize breaks input into tokens handling quotes and escapes
func (p *Parser) tokenize(input string) ([]Token, error) {
	var tokens []Token
	var current strings.Builder
	var inQuotes bool
	var quoteChar rune
	var escaped bool
	position := 0
	
	for i, r := range input {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		
		if r == '\\' {
			escaped = true
			continue
		}
		
		if !inQuotes && (r == '"' || r == '\'') {
			inQuotes = true
			quoteChar = r
			continue
		}
		
		if inQuotes && r == quoteChar {
			inQuotes = false
			continue
		}
		
		if !inQuotes && unicode.IsSpace(r) {
			if current.Len() > 0 {
				tokens = append(tokens, Token{
					Type:     p.getTokenType(current.String()),
					Value:    current.String(),
					Position: position,
				})
				current.Reset()
				position = i + 1
			}
			continue
		}
		
		current.WriteRune(r)
	}
	
	if inQuotes {
		return nil, fmt.Errorf("unterminated quote at position %d", position)
	}
	
	if current.Len() > 0 {
		tokens = append(tokens, Token{
			Type:     p.getTokenType(current.String()),
			Value:    current.String(),
			Position: position,
		})
	}
	
	return tokens, nil
}

// getTokenType determines the type of a token
func (p *Parser) getTokenType(value string) string {
	if strings.HasPrefix(value, "--") {
		return "long_flag"
	}
	if strings.HasPrefix(value, "-") && len(value) > 1 {
		return "short_flag"
	}
	return "arg"
}

// parseWithSpec parses tokens according to command specification
func (p *Parser) parseWithSpec(cmd *Command, tokens []Token, spec *CommandSpec) (*Command, error) {
	// Initialize flags with defaults
	for flagName, flagSpec := range spec.Flags {
		if flagSpec.Default != nil {
			cmd.Flags[flagName] = flagSpec.Default
		}
	}
	
	i := 0
	for i < len(tokens) {
		token := tokens[i]
		
		switch token.Type {
		case "long_flag":
			flagName := strings.TrimPrefix(token.Value, "--")
			consumed, err := p.parseFlag(cmd, flagName, tokens, i, spec, false)
			if err != nil {
				return cmd, err
			}
			i += consumed
			
		case "short_flag":
			flagChars := strings.TrimPrefix(token.Value, "-")
			// Handle combined short flags like -vf
			for j, char := range flagChars {
				flagName := p.findShortFlag(string(char), spec)
				if flagName == "" {
					return cmd, fmt.Errorf("unknown flag: -%c at position %d", char, token.Position+j+1)
				}
				
				// For the last flag, check if it needs a value
				if j == len(flagChars)-1 {
					consumed, err := p.parseFlag(cmd, flagName, tokens, i, spec, true)
					if err != nil {
						return cmd, err
					}
					i += consumed
				} else {
					// Short flag in middle, must be boolean
					flagSpec, exists := spec.Flags[flagName]
					if !exists || flagSpec.Type != "bool" {
						return cmd, fmt.Errorf("non-boolean flag -%c cannot be combined at position %d", char, token.Position+j+1)
					}
					cmd.Flags[flagName] = true
				}
			}
			
		case "arg":
			cmd.Args = append(cmd.Args, token.Value)
			i++
		}
	}
	
	// Validate argument count
	if len(cmd.Args) < spec.MinArgs {
		return cmd, fmt.Errorf("command %s requires at least %d arguments, got %d", spec.Name, spec.MinArgs, len(cmd.Args))
	}
	
	if spec.MaxArgs >= 0 && len(cmd.Args) > spec.MaxArgs {
		return cmd, fmt.Errorf("command %s accepts at most %d arguments, got %d", spec.Name, spec.MaxArgs, len(cmd.Args))
	}
	
	// Validate required flags
	for flagName, flagSpec := range spec.Flags {
		if flagSpec.Required {
			if _, exists := cmd.Flags[flagName]; !exists {
				return cmd, fmt.Errorf("required flag --%s is missing", flagName)
			}
		}
	}
	
	return cmd, nil
}

// parseFlag parses a single flag and its value
func (p *Parser) parseFlag(cmd *Command, flagName string, tokens []Token, index int, spec *CommandSpec, isShort bool) (int, error) {
	flagSpec, exists := spec.Flags[flagName]
	if !exists {
		prefix := "--"
		if isShort {
			prefix = "-"
		}
		return 0, fmt.Errorf("unknown flag: %s%s", prefix, flagName)
	}
	
	consumed := 1 // Always consume the flag token
	
	switch flagSpec.Type {
	case "bool":
		cmd.Flags[flagName] = true
		
	case "string":
		if index+1 >= len(tokens) || tokens[index+1].Type != "arg" {
			return 0, fmt.Errorf("flag --%s requires a string value", flagName)
		}
		cmd.Flags[flagName] = tokens[index+1].Value
		consumed = 2
		
	case "int":
		if index+1 >= len(tokens) || tokens[index+1].Type != "arg" {
			return 0, fmt.Errorf("flag --%s requires an integer value", flagName)
		}
		val, err := strconv.Atoi(tokens[index+1].Value)
		if err != nil {
			return 0, fmt.Errorf("flag --%s requires an integer value, got: %s", flagName, tokens[index+1].Value)
		}
		cmd.Flags[flagName] = val
		consumed = 2
		
	case "float":
		if index+1 >= len(tokens) || tokens[index+1].Type != "arg" {
			return 0, fmt.Errorf("flag --%s requires a float value", flagName)
		}
		val, err := strconv.ParseFloat(tokens[index+1].Value, 64)
		if err != nil {
			return 0, fmt.Errorf("flag --%s requires a float value, got: %s", flagName, tokens[index+1].Value)
		}
		cmd.Flags[flagName] = val
		consumed = 2
		
	default:
		return 0, fmt.Errorf("unsupported flag type: %s", flagSpec.Type)
	}
	
	return consumed, nil
}

// findShortFlag finds the flag name for a short flag character
func (p *Parser) findShortFlag(char string, spec *CommandSpec) string {
	for flagName, flagSpec := range spec.Flags {
		if flagSpec.Short == char {
			return flagName
		}
	}
	return ""
}

// GetCommandSpecs returns all registered command specifications
func (p *Parser) GetCommandSpecs() map[string]*CommandSpec {
	// Return a copy to prevent modification
	specs := make(map[string]*CommandSpec)
	for name, spec := range p.specs {
		specs[name] = spec
	}
	return specs
}

// GetCompletions returns possible completions for a partial command
func (p *Parser) GetCompletions(partial string) []string {
	var completions []string
	seen := make(map[string]bool)
	
	for name, spec := range p.specs {
		// Only include main command names, not aliases
		if name == spec.Name && strings.HasPrefix(name, partial) {
			if !seen[name] {
				completions = append(completions, name)
				seen[name] = true
			}
		}
	}
	
	return completions
}

// Validate validates a command against its specification
func (p *Parser) Validate(cmd *Command) error {
	spec, exists := p.specs[cmd.Name]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
	
	// Validate argument count
	if len(cmd.Args) < spec.MinArgs {
		return fmt.Errorf("command %s requires at least %d arguments, got %d", spec.Name, spec.MinArgs, len(cmd.Args))
	}
	
	if spec.MaxArgs >= 0 && len(cmd.Args) > spec.MaxArgs {
		return fmt.Errorf("command %s accepts at most %d arguments, got %d", spec.Name, spec.MaxArgs, len(cmd.Args))
	}
	
	// Validate flags
	for flagName, value := range cmd.Flags {
		flagSpec, exists := spec.Flags[flagName]
		if !exists {
			return fmt.Errorf("unknown flag: %s", flagName)
		}
		
		// Validate flag type
		if !p.isValidType(value, flagSpec.Type) {
			return fmt.Errorf("flag %s has invalid type: expected %s, got %T", flagName, flagSpec.Type, value)
		}
	}
	
	return nil
}

// isValidType checks if a value matches the expected type
func (p *Parser) isValidType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "int":
		_, ok := value.(int)
		return ok
	case "float":
		_, ok := value.(float64)
		return ok
	case "bool":
		_, ok := value.(bool)
		return ok
	default:
		return false
	}
}

// GetCommandHelp generates help text for a command
func (p *Parser) GetCommandHelp(commandName string) (string, error) {
	spec, exists := p.specs[commandName]
	if !exists {
		return "", fmt.Errorf("unknown command: %s", commandName)
	}
	
	var help strings.Builder
	
	help.WriteString(fmt.Sprintf("Command: %s\n", spec.Name))
	help.WriteString(fmt.Sprintf("Description: %s\n", spec.Description))
	
	if spec.Usage != "" {
		help.WriteString(fmt.Sprintf("Usage: %s\n", spec.Usage))
	}
	
	if spec.Category != "" {
		help.WriteString(fmt.Sprintf("Category: %s\n", spec.Category))
	}
	
	if len(spec.Aliases) > 0 {
		help.WriteString(fmt.Sprintf("Aliases: %s\n", strings.Join(spec.Aliases, ", ")))
	}
	
	if len(spec.Flags) > 0 {
		help.WriteString("\nFlags:\n")
		for flagName, flagSpec := range spec.Flags {
			shortFlag := ""
			if flagSpec.Short != "" {
				shortFlag = fmt.Sprintf(", -%s", flagSpec.Short)
			}
			
			required := ""
			if flagSpec.Required {
				required = " (required)"
			}
			
			defaultVal := ""
			if flagSpec.Default != nil {
				defaultVal = fmt.Sprintf(" (default: %v)", flagSpec.Default)
			}
			
			help.WriteString(fmt.Sprintf("  --%s%s: %s%s%s\n", 
				flagName, shortFlag, flagSpec.Description, required, defaultVal))
		}
	}
	
	if len(spec.Examples) > 0 {
		help.WriteString("\nExamples:\n")
		for _, example := range spec.Examples {
			help.WriteString(fmt.Sprintf("  %s\n", example))
		}
	}
	
	return help.String(), nil
}