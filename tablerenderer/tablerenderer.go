package tablerenderer

import (
	"fmt"
	"html/template"
	"reflect"
	"strings"
)

// TableData represents the data structure for rendering tables
type TableData struct {
	Headers []string        `json:"headers"`
	Rows    [][]interface{} `json:"rows"`
	Data    interface{}     `json:"data,omitempty"` // New field for struct slices
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

// extractHeadersFromStruct extracts field names from a struct type to use as headers
func extractHeadersFromStruct(structType reflect.Type) []string {
	var headers []string
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		// Use json tag if available, otherwise use field name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			// Remove omitempty and other options
			tagName := strings.Split(tag, ",")[0]
			if tagName != "" {
				headers = append(headers, tagName)
			} else {
				headers = append(headers, field.Name)
			}
		} else {
			headers = append(headers, field.Name)
		}
	}
	return headers
}

// convertStructSliceToRows converts a slice of structs to [][]interface{}
func convertStructSliceToRows(data interface{}) ([]string, [][]interface{}, error) {
	v := reflect.ValueOf(data)

	// Handle pointer to slice
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Slice {
		return nil, nil, fmt.Errorf("data must be a slice of structs")
	}

	if v.Len() == 0 {
		return []string{}, [][]interface{}{}, nil
	}

	// Get the type of the first element to extract headers
	firstElem := v.Index(0)
	if firstElem.Kind() == reflect.Ptr {
		firstElem = firstElem.Elem()
	}

	if firstElem.Kind() != reflect.Struct {
		return nil, nil, fmt.Errorf("slice elements must be structs")
	}

	structType := firstElem.Type()
	headers := extractHeadersFromStruct(structType)

	// Convert each struct to a row
	rows := make([][]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Ptr {
			elem = elem.Elem()
		}

		row := make([]interface{}, elem.NumField())
		for j := 0; j < elem.NumField(); j++ {
			row[j] = elem.Field(j).Interface()
		}
		rows[i] = row
	}

	return headers, rows, nil
}

// RenderHTML renders table data as HTML string
func (r *Renderer) RenderHTML(data TableData) (string, error) {

	var headers []string
	var rows [][]interface{}
	var err error

	// If Data field is provided (struct slice), use it and auto-generate headers/rows
	if data.Data != nil {
		headers, rows, err = convertStructSliceToRows(data.Data)
		if err != nil {
			return "", fmt.Errorf("failed to convert struct data: %w", err)
		}
		// Override with manual headers if provided
		if len(data.Headers) > 0 {
			headers = data.Headers
		}
	} else {
		// Use traditional Headers and Rows fields
		headers = data.Headers
		rows = data.Rows
	}

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
		Headers:    headers,
		Rows:       rows,
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
