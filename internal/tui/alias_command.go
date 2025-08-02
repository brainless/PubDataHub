package tui

import (
	"fmt"
	"strconv"
	"strings"
)

// AliasCommand handles alias-related operations
type AliasCommand struct {
	aliasManager *AliasManager
}

// NewAliasCommand creates a new alias command handler
func NewAliasCommand(aliasManager *AliasManager) *AliasCommand {
	return &AliasCommand{
		aliasManager: aliasManager,
	}
}

// GetHelp returns help text for the alias command
func (ac *AliasCommand) GetHelp() string {
	return "Manage command aliases for creating shortcuts to complex commands"
}

// GetUsage returns usage information for the alias command
func (ac *AliasCommand) GetUsage() string {
	return "alias <subcommand> [args...]"
}

// Execute processes alias commands
func (ac *AliasCommand) Execute(ctx *ShellContext) error {
	if len(ctx.Args) < 2 {
		return ac.showUsage()
	}

	subcommand := ctx.Args[1]

	switch subcommand {
	case "add", "create":
		return ac.handleAdd(ctx.Args[2:])
	case "remove", "delete", "rm":
		return ac.handleRemove(ctx.Args[2:])
	case "list", "ls":
		return ac.handleList(ctx.Args[2:])
	case "show":
		return ac.handleShow(ctx.Args[2:])
	case "update", "edit":
		return ac.handleUpdate(ctx.Args[2:])
	case "stats":
		return ac.handleStats()
	case "export":
		return ac.handleExport(ctx.Args[2:])
	case "import":
		return ac.handleImport(ctx.Args[2:])
	case "popular":
		return ac.handlePopular(ctx.Args[2:])
	default:
		return fmt.Errorf("unknown alias subcommand: %s", subcommand)
	}
}

// GetCompletions provides tab completion for alias commands
func (ac *AliasCommand) GetCompletions(partial string, args []string) []string {
	if len(args) == 0 {
		// Complete subcommands
		subcommands := []string{"add", "remove", "list", "show", "update", "stats", "export", "import", "popular"}
		var completions []string
		for _, cmd := range subcommands {
			if partial == "" || strings.HasPrefix(cmd, partial) {
				completions = append(completions, cmd)
			}
		}
		return completions
	}

	if len(args) == 1 {
		subcommand := args[0]
		switch subcommand {
		case "remove", "delete", "rm", "show", "update", "edit":
			// Complete with existing alias names
			return ac.aliasManager.GetCompletions(partial)
		}
	}

	return []string{}
}

// handleAdd creates a new alias
func (ac *AliasCommand) handleAdd(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: alias add <name> <command> [description]")
	}

	name := args[0]
	command := args[1]
	description := ""

	if len(args) > 2 {
		description = strings.Join(args[2:], " ")
	}

	return ac.aliasManager.AddAlias(name, command, description)
}

// handleRemove deletes an existing alias
func (ac *AliasCommand) handleRemove(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: alias remove <name>")
	}

	name := args[0]
	return ac.aliasManager.RemoveAlias(name)
}

// handleList shows all aliases
func (ac *AliasCommand) handleList(args []string) error {
	aliases := ac.aliasManager.ListAliases()

	if len(aliases) == 0 {
		fmt.Println("No aliases defined")
		return nil
	}

	fmt.Printf("%-15s %-30s %-10s %s\n", "NAME", "COMMAND", "USAGE", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))

	for _, alias := range aliases {
		description := alias.Description
		if len(description) > 25 {
			description = description[:22] + "..."
		}

		command := alias.Command
		if len(command) > 28 {
			command = command[:25] + "..."
		}

		fmt.Printf("%-15s %-30s %-10d %s\n", alias.Name, command, alias.Usage, description)
	}

	return nil
}

// handleShow displays details of a specific alias
func (ac *AliasCommand) handleShow(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: alias show <name>")
	}

	name := args[0]
	alias, exists := ac.aliasManager.GetAlias(name)
	if !exists {
		return fmt.Errorf("alias '%s' not found", name)
	}

	fmt.Printf("Alias: %s\n", alias.Name)
	fmt.Printf("Command: %s\n", alias.Command)
	fmt.Printf("Description: %s\n", alias.Description)
	fmt.Printf("Usage Count: %d\n", alias.Usage)
	fmt.Printf("Created: %s\n", alias.Created)

	return nil
}

// handleUpdate modifies an existing alias
func (ac *AliasCommand) handleUpdate(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("usage: alias update <name> <command> [description]")
	}

	name := args[0]
	command := args[1]
	description := ""

	if len(args) > 2 {
		description = strings.Join(args[2:], " ")
	}

	return ac.aliasManager.UpdateAlias(name, command, description)
}

// handleStats shows alias usage statistics
func (ac *AliasCommand) handleStats() error {
	stats := ac.aliasManager.GetAliasStats()

	fmt.Printf("Alias Statistics:\n")
	fmt.Printf("  Total Aliases: %d\n", stats.TotalAliases)
	fmt.Printf("  Total Usage: %d\n", stats.TotalUsage)
	fmt.Printf("  Average Usage: %.2f\n", stats.AverageUsage)

	if stats.MostUsed != "" {
		fmt.Printf("  Most Used: %s\n", stats.MostUsed)
	}

	if stats.LeastUsed != "" {
		fmt.Printf("  Least Used: %s\n", stats.LeastUsed)
	}

	return nil
}

// handleExport saves aliases to a file
func (ac *AliasCommand) handleExport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: alias export <filename>")
	}

	filename := args[0]
	return ac.aliasManager.ExportAliases(filename)
}

// handleImport loads aliases from a file
func (ac *AliasCommand) handleImport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: alias import <filename>")
	}

	filename := args[0]
	return ac.aliasManager.ImportAliases(filename)
}

// handlePopular shows most frequently used aliases
func (ac *AliasCommand) handlePopular(args []string) error {
	limit := 10
	if len(args) > 0 {
		if l, err := strconv.Atoi(args[0]); err == nil && l > 0 {
			limit = l
		}
	}

	aliases := ac.aliasManager.GetPopularAliases(limit)

	if len(aliases) == 0 {
		fmt.Println("No aliases defined")
		return nil
	}

	fmt.Printf("Top %d Most Popular Aliases:\n", len(aliases))
	fmt.Printf("%-15s %-30s %-10s %s\n", "NAME", "COMMAND", "USAGE", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 80))

	for i, alias := range aliases {
		description := alias.Description
		if len(description) > 25 {
			description = description[:22] + "..."
		}

		command := alias.Command
		if len(command) > 28 {
			command = command[:25] + "..."
		}

		fmt.Printf("%2d. %-12s %-30s %-10d %s\n", i+1, alias.Name, command, alias.Usage, description)
	}

	return nil
}

// showUsage displays command usage information
func (ac *AliasCommand) showUsage() error {
	fmt.Println("Alias Command Usage:")
	fmt.Println("  alias add <name> <command> [description]  - Create a new alias")
	fmt.Println("  alias remove <name>                       - Remove an alias")
	fmt.Println("  alias list                                - List all aliases")
	fmt.Println("  alias show <name>                         - Show alias details")
	fmt.Println("  alias update <name> <command> [desc]      - Update an alias")
	fmt.Println("  alias stats                               - Show usage statistics")
	fmt.Println("  alias popular [limit]                     - Show most used aliases")
	fmt.Println("  alias export <filename>                   - Export aliases to file")
	fmt.Println("  alias import <filename>                   - Import aliases from file")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  alias add hn 'download hackernews' 'Download Hacker News data'")
	fmt.Println("  alias add top10 'query hackernews \"SELECT title FROM items ORDER BY score DESC LIMIT 10\"'")
	fmt.Println("  alias remove hn")
	fmt.Println("  alias popular 5")

	return nil
}
