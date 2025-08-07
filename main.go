package main

import (
	"fmt"
	"log"

	"github.com/faiakak/table-renderer/tablerenderer"
)

func main() {
	// Create a new table renderer
	renderer := tablerenderer.NewRenderer()

	// Example 1: Simple table with user data
	fmt.Println("=== Example 1: User Data Table ===")
	userData := tablerenderer.TableData{
		Headers: []string{"ID", "Name", "Email", "Age"},
		Rows: [][]interface{}{
			{1, "John Doe", "john@example.com", 30},
			{2, "Jane Smith", "jane@example.com", 25},
			{3, "Bob Wilson", "bob@example.com", 35},
		},
		Options: tablerenderer.TableOptions{
			CSSClass:   "user-table",
			ID:         "users",
			Striped:    true,
			Bordered:   true,
			Responsive: true,
		},
	}

	// Render as HTML
	htmlOutput, err := renderer.RenderHTML(userData)
	if err != nil {
		log.Fatal("Error rendering HTML:", err)
	}
	fmt.Println("HTML Output:")
	fmt.Println(htmlOutput)
	fmt.Println()

}
