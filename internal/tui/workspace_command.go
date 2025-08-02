package tui

import (
	"fmt"
	"strings"
	"time"
)

// WorkspaceCommand handles workspace-related operations
type WorkspaceCommand struct {
	workspaceManager *WorkspaceManager
}

// NewWorkspaceCommand creates a new workspace command handler
func NewWorkspaceCommand(workspaceManager *WorkspaceManager) *WorkspaceCommand {
	return &WorkspaceCommand{
		workspaceManager: workspaceManager,
	}
}

// GetHelp returns help text for the workspace command
func (wc *WorkspaceCommand) GetHelp() string {
	return "Manage workspaces for organizing queries, job templates, and settings"
}

// GetUsage returns usage information for the workspace command
func (wc *WorkspaceCommand) GetUsage() string {
	return "workspace <subcommand> [args...]"
}

// Execute processes workspace commands
func (wc *WorkspaceCommand) Execute(ctx *ShellContext) error {
	if len(ctx.Args) < 2 {
		return wc.showUsage()
	}

	subcommand := ctx.Args[1]

	switch subcommand {
	case "create":
		return wc.handleCreate(ctx.Args[2:])
	case "list", "ls":
		return wc.handleList(ctx.Args[2:])
	case "switch", "use":
		return wc.handleSwitch(ctx.Args[2:])
	case "delete", "remove", "rm":
		return wc.handleDelete(ctx.Args[2:])
	case "current":
		return wc.handleCurrent()
	case "info", "show":
		return wc.handleInfo(ctx.Args[2:])
	case "export":
		return wc.handleExport(ctx.Args[2:])
	case "import":
		return wc.handleImport(ctx.Args[2:])
	case "stats":
		return wc.handleStats()
	case "search":
		return wc.handleSearch(ctx.Args[2:])
	case "query":
		return wc.handleQuery(ctx.Args[2:])
	default:
		return fmt.Errorf("unknown workspace subcommand: %s", subcommand)
	}
}

// GetCompletions provides tab completion for workspace commands
func (wc *WorkspaceCommand) GetCompletions(partial string, args []string) []string {
	if len(args) == 0 {
		// Complete subcommands
		subcommands := []string{"create", "list", "switch", "delete", "current", "info", "export", "import", "stats", "search", "query"}
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
		case "switch", "use", "delete", "remove", "rm", "info", "show", "export":
			// Complete with workspace names
			return wc.getWorkspaceCompletions(partial)
		}
	}

	return []string{}
}

// handleCreate creates a new workspace
func (wc *WorkspaceCommand) handleCreate(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: workspace create <name> [description]")
	}

	name := args[0]
	description := ""

	if len(args) > 1 {
		description = strings.Join(args[1:], " ")
	}

	return wc.workspaceManager.CreateWorkspace(name, description)
}

// handleList lists all workspaces
func (wc *WorkspaceCommand) handleList(args []string) error {
	workspaces := wc.workspaceManager.ListWorkspaces()

	if len(workspaces) == 0 {
		fmt.Println("No workspaces found")
		return nil
	}

	current := wc.workspaceManager.GetCurrentWorkspace()
	currentName := ""
	if current != nil {
		currentName = current.Name
	}

	fmt.Printf("%-20s %-30s %-12s %-8s %s\n", "NAME", "DESCRIPTION", "LAST USED", "USAGE", "CURRENT")
	fmt.Println(strings.Repeat("-", 85))

	for _, ws := range workspaces {
		marker := ""
		if ws.Name == currentName {
			marker = "✓"
		}

		description := ws.Description
		if len(description) > 28 {
			description = description[:25] + "..."
		}

		lastUsed := ws.LastUsed.Format("2006-01-02")
		if time.Since(ws.LastUsed) < 24*time.Hour {
			lastUsed = ws.LastUsed.Format("15:04")
		}

		fmt.Printf("%-20s %-30s %-12s %-8d %s\n",
			ws.Name, description, lastUsed, ws.UsageCount, marker)
	}

	return nil
}

// handleSwitch switches to a different workspace
func (wc *WorkspaceCommand) handleSwitch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: workspace switch <name>")
	}

	name := args[0]
	return wc.workspaceManager.SwitchWorkspace(name)
}

// handleDelete removes a workspace
func (wc *WorkspaceCommand) handleDelete(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: workspace delete <name>")
	}

	name := args[0]

	// Confirm deletion
	current := wc.workspaceManager.GetCurrentWorkspace()
	if current != nil && current.Name == name {
		fmt.Printf("Warning: You are about to delete the current workspace '%s'\n", name)
	}

	return wc.workspaceManager.DeleteWorkspace(name)
}

// handleCurrent shows the current workspace
func (wc *WorkspaceCommand) handleCurrent() error {
	current := wc.workspaceManager.GetCurrentWorkspace()
	if current == nil {
		fmt.Println("No workspace is currently active")
		return nil
	}

	fmt.Printf("Current workspace: %s\n", current.Name)
	if current.Description != "" {
		fmt.Printf("Description: %s\n", current.Description)
	}
	fmt.Printf("Created: %s\n", current.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("Last used: %s\n", current.LastUsed.Format("2006-01-02 15:04:05"))
	fmt.Printf("Usage count: %d\n", current.UsageCount)
	fmt.Printf("Saved queries: %d\n", len(current.SavedQueries))
	fmt.Printf("Job templates: %d\n", len(current.JobTemplates))

	return nil
}

// handleInfo shows detailed information about a workspace
func (wc *WorkspaceCommand) handleInfo(args []string) error {
	var name string
	if len(args) == 0 {
		// Show current workspace info
		current := wc.workspaceManager.GetCurrentWorkspace()
		if current == nil {
			return fmt.Errorf("no workspace specified and no current workspace")
		}
		name = current.Name
	} else {
		name = args[0]
	}

	workspaces := wc.workspaceManager.ListWorkspaces()
	var workspace *Workspace
	for _, ws := range workspaces {
		if ws.Name == name {
			workspace = ws
			break
		}
	}

	if workspace == nil {
		return fmt.Errorf("workspace '%s' not found", name)
	}

	fmt.Printf("Workspace: %s\n", workspace.Name)
	fmt.Printf("Description: %s\n", workspace.Description)
	fmt.Printf("Created: %s\n", workspace.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("Last used: %s\n", workspace.LastUsed.Format("2006-01-02 15:04:05"))
	fmt.Printf("Usage count: %d\n", workspace.UsageCount)
	fmt.Printf("Tags: %s\n", strings.Join(workspace.Tags, ", "))

	fmt.Printf("\nSettings:\n")
	fmt.Printf("  Default data source: %s\n", workspace.Settings.DefaultDataSource)
	fmt.Printf("  Auto complete: %t\n", workspace.Settings.AutoComplete)
	fmt.Printf("  Show timing: %t\n", workspace.Settings.ShowTiming)
	fmt.Printf("  Pagination size: %d\n", workspace.Settings.PaginationSize)
	fmt.Printf("  Output format: %s\n", workspace.Settings.OutputFormat)
	fmt.Printf("  Theme: %s\n", workspace.Settings.Theme)

	fmt.Printf("\nSaved queries (%d):\n", len(workspace.SavedQueries))
	for name, query := range workspace.SavedQueries {
		fmt.Printf("  - %s: %s\n", name, query.Description)
	}

	fmt.Printf("\nJob templates (%d):\n", len(workspace.JobTemplates))
	for name, template := range workspace.JobTemplates {
		fmt.Printf("  - %s: %s\n", name, template.Description)
	}

	return nil
}

// handleExport exports a workspace to a file
func (wc *WorkspaceCommand) handleExport(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: workspace export <workspace_name> <filename>")
	}

	workspaceName := args[0]
	filename := args[1]

	return wc.workspaceManager.ExportWorkspace(workspaceName, filename)
}

// handleImport imports a workspace from a file
func (wc *WorkspaceCommand) handleImport(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: workspace import <filename>")
	}

	filename := args[0]
	return wc.workspaceManager.ImportWorkspace(filename)
}

// handleStats shows workspace statistics
func (wc *WorkspaceCommand) handleStats() error {
	stats := wc.workspaceManager.GetWorkspaceStats()

	fmt.Printf("Workspace Statistics:\n")
	fmt.Printf("  Total workspaces: %d\n", stats.TotalWorkspaces)
	fmt.Printf("  Total saved queries: %d\n", stats.TotalQueries)
	fmt.Printf("  Total job templates: %d\n", stats.TotalTemplates)
	fmt.Printf("  Total usage: %d\n", stats.TotalUsage)

	if stats.TotalWorkspaces > 0 {
		fmt.Printf("  Average queries per workspace: %.2f\n", float64(stats.TotalQueries)/float64(stats.TotalWorkspaces))
		fmt.Printf("  Average usage per workspace: %.2f\n", float64(stats.TotalUsage)/float64(stats.TotalWorkspaces))
	}

	return nil
}

// handleSearch searches across workspaces
func (wc *WorkspaceCommand) handleSearch(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: workspace search <query>")
	}

	query := strings.Join(args, " ")
	results := wc.workspaceManager.Search(query)

	if len(results.Workspaces) == 0 && len(results.Queries) == 0 && len(results.Templates) == 0 {
		fmt.Printf("No results found for '%s'\n", query)
		return nil
	}

	fmt.Printf("Search results for '%s':\n\n", query)

	if len(results.Workspaces) > 0 {
		fmt.Printf("Workspaces (%d):\n", len(results.Workspaces))
		for _, ws := range results.Workspaces {
			fmt.Printf("  - %s\n", ws)
		}
		fmt.Println()
	}

	if len(results.Queries) > 0 {
		fmt.Printf("Saved Queries (%d):\n", len(results.Queries))
		for _, q := range results.Queries {
			fmt.Printf("  - %s/%s: %s\n", q.WorkspaceName, q.QueryName, q.Description)
		}
		fmt.Println()
	}

	if len(results.Templates) > 0 {
		fmt.Printf("Job Templates (%d):\n", len(results.Templates))
		for _, t := range results.Templates {
			fmt.Printf("  - %s/%s: %s\n", t.WorkspaceName, t.TemplateName, t.Description)
		}
		fmt.Println()
	}

	return nil
}

// handleQuery manages saved queries in the current workspace
func (wc *WorkspaceCommand) handleQuery(args []string) error {
	if len(args) == 0 {
		return wc.showQueryUsage()
	}

	subcommand := args[0]

	switch subcommand {
	case "save":
		return wc.handleSaveQuery(args[1:])
	case "list", "ls":
		return wc.handleListQueries()
	case "show":
		return wc.handleShowQuery(args[1:])
	case "delete", "remove", "rm":
		return wc.handleDeleteQuery(args[1:])
	default:
		return fmt.Errorf("unknown query subcommand: %s", subcommand)
	}
}

// handleSaveQuery saves a query to the current workspace
func (wc *WorkspaceCommand) handleSaveQuery(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: workspace query save <name> <query> [description]")
	}

	name := args[0]
	query := args[1]
	description := ""

	if len(args) > 2 {
		description = strings.Join(args[2:], " ")
	}

	current := wc.workspaceManager.GetCurrentWorkspace()
	if current == nil {
		return fmt.Errorf("no active workspace")
	}

	return wc.workspaceManager.SaveQuery(name, query, current.Settings.DefaultDataSource, description, []string{})
}

// handleListQueries lists all saved queries in the current workspace
func (wc *WorkspaceCommand) handleListQueries() error {
	current := wc.workspaceManager.GetCurrentWorkspace()
	if current == nil {
		return fmt.Errorf("no active workspace")
	}

	if len(current.SavedQueries) == 0 {
		fmt.Println("No saved queries in current workspace")
		return nil
	}

	fmt.Printf("Saved queries in workspace '%s':\n", current.Name)
	fmt.Printf("%-20s %-15s %-8s %s\n", "NAME", "DATA SOURCE", "USAGE", "DESCRIPTION")
	fmt.Println(strings.Repeat("-", 70))

	for _, query := range current.SavedQueries {
		description := query.Description
		if len(description) > 25 {
			description = description[:22] + "..."
		}

		fmt.Printf("%-20s %-15s %-8d %s\n",
			query.Name, query.DataSource, query.UsageCount, description)
	}

	return nil
}

// handleShowQuery shows details of a saved query
func (wc *WorkspaceCommand) handleShowQuery(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: workspace query show <name>")
	}

	name := args[0]
	query, err := wc.workspaceManager.GetSavedQuery(name)
	if err != nil {
		return err
	}

	fmt.Printf("Query: %s\n", query.Name)
	fmt.Printf("Data source: %s\n", query.DataSource)
	fmt.Printf("Description: %s\n", query.Description)
	fmt.Printf("SQL: %s\n", query.Query)
	fmt.Printf("Tags: %s\n", strings.Join(query.Tags, ", "))
	fmt.Printf("Created: %s\n", query.Created.Format("2006-01-02 15:04:05"))
	fmt.Printf("Last used: %s\n", query.LastUsed.Format("2006-01-02 15:04:05"))
	fmt.Printf("Usage count: %d\n", query.UsageCount)
	fmt.Printf("Favorite: %t\n", query.IsFavorite)

	return nil
}

// handleDeleteQuery removes a saved query
func (wc *WorkspaceCommand) handleDeleteQuery(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: workspace query delete <name>")
	}

	name := args[0]
	current := wc.workspaceManager.GetCurrentWorkspace()
	if current == nil {
		return fmt.Errorf("no active workspace")
	}

	if _, exists := current.SavedQueries[name]; !exists {
		return fmt.Errorf("query '%s' not found", name)
	}

	delete(current.SavedQueries, name)
	fmt.Printf("Deleted query '%s' from workspace '%s'\n", name, current.Name)
	return nil
}

// getWorkspaceCompletions returns workspace names for completion
func (wc *WorkspaceCommand) getWorkspaceCompletions(partial string) []string {
	workspaces := wc.workspaceManager.ListWorkspaces()
	var completions []string

	for _, ws := range workspaces {
		if partial == "" || strings.HasPrefix(ws.Name, partial) {
			completions = append(completions, ws.Name)
		}
	}

	return completions
}

// showUsage displays command usage information
func (wc *WorkspaceCommand) showUsage() error {
	fmt.Println("Workspace Command Usage:")
	fmt.Println("  workspace create <name> [description]     - Create a new workspace")
	fmt.Println("  workspace list                            - List all workspaces")
	fmt.Println("  workspace switch <name>                   - Switch to a workspace")
	fmt.Println("  workspace delete <name>                   - Delete a workspace")
	fmt.Println("  workspace current                         - Show current workspace")
	fmt.Println("  workspace info [name]                     - Show workspace details")
	fmt.Println("  workspace export <name> <file>            - Export workspace to file")
	fmt.Println("  workspace import <file>                   - Import workspace from file")
	fmt.Println("  workspace stats                           - Show workspace statistics")
	fmt.Println("  workspace search <query>                  - Search across workspaces")
	fmt.Println("  workspace query <subcommand>              - Manage saved queries")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  workspace create analytics 'Data analysis workspace'")
	fmt.Println("  workspace switch analytics")
	fmt.Println("  workspace query save top_stories 'SELECT title FROM items ORDER BY score DESC LIMIT 10'")

	return nil
}

// showQueryUsage displays query subcommand usage
func (wc *WorkspaceCommand) showQueryUsage() error {
	fmt.Println("Workspace Query Command Usage:")
	fmt.Println("  workspace query save <name> <query> [desc] - Save a query to workspace")
	fmt.Println("  workspace query list                       - List saved queries")
	fmt.Println("  workspace query show <name>                - Show query details")
	fmt.Println("  workspace query delete <name>              - Delete a saved query")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  workspace query save top10 'SELECT title FROM items ORDER BY score DESC LIMIT 10' 'Top stories'")
	fmt.Println("  workspace query show top10")

	return nil
}
