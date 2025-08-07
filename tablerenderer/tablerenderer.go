package tablerenderer

import (
	"fmt"
	"html/template"
	"strings"
)

// TableData represents the data structure for rendering tables
type TableData struct {
	Headers []string        `json:"headers"`
	Rows    [][]interface{} `json:"rows"`
	Options TableOptions    `json:"options,omitempty"`
}

// TableOptions holds configuration for table rendering
type TableOptions struct {
	CSSClass   string `json:"css_class,omitempty"`
	ID         string `json:"id,omitempty"`
	Striped    bool   `json:"striped,omitempty"`
	Bordered   bool   `json:"bordered,omitempty"`
	Responsive bool   `json:"responsive,omitempty"`
}

// Renderer is the main struct for rendering tables
type Renderer struct {
	template *template.Template
}

// NewRenderer creates a new table renderer instance
func NewRenderer() *Renderer {
	return &Renderer{}
}

// RenderHTML renders table data as HTML string
func (r *Renderer) RenderHTML(data TableData) (string, error) {
	// Build CSS classes
	cssClasses := []string{"table"}

	if data.Options.CSSClass != "" {
		cssClasses = append(cssClasses, data.Options.CSSClass)
	}
	if data.Options.Striped {
		cssClasses = append(cssClasses, "table-striped")
	}
	if data.Options.Bordered {
		cssClasses = append(cssClasses, "table-bordered")
	}

	// Simple HTML template
	htmlTemplate := `
<table class="{{.CSSClasses}}"{{if .ID}} id="{{.ID}}"{{end}}>
	<thead>
		<tr>
			{{range .Headers}}
			<th>{{.}}</th>
			{{end}}
		</tr>
	</thead>
	<tbody>
		{{range .Rows}}
		<tr>
			{{range .}}
			<td>{{.}}</td>
			{{end}}
		</tr>
		{{end}}
	</tbody>
</table>`

	tmpl, err := template.New("table").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Prepare template data
	templateData := struct {
		Headers    []string
		Rows       [][]interface{}
		CSSClasses string
		ID         string
	}{
		Headers:    data.Headers,
		Rows:       data.Rows,
		CSSClasses: strings.Join(cssClasses, " "),
		ID:         data.Options.ID,
	}

	var result strings.Builder
	err = tmpl.Execute(&result, templateData)
	if err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	// Wrap in responsive div if needed
	if data.Options.Responsive {
		return fmt.Sprintf(`<div class="table-responsive">%s</div>`, result.String()), nil
	}

	return result.String(), nil
}
