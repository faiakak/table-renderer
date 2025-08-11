package tablerenderer

import (
	"fmt"
	"html/template"
	"reflect"
	"strconv"
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
	CSSClass   string      `json:"css_class,omitempty"`
	ID         string      `json:"id,omitempty"`
	Striped    bool        `json:"striped,omitempty"`
	Bordered   bool        `json:"bordered,omitempty"`
	Responsive bool        `json:"responsive,omitempty"`
	Style      string      `json:"style,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
}

// Pagination holds pagination configuration
type Pagination struct {
	Enabled       bool   `json:"enabled"`
	PageSize      int    `json:"page_size"`
	CurrentPage   int    `json:"current_page"`
	ShowControls  bool   `json:"show_controls"`
	ShowInfo      bool   `json:"show_info"`
	BaseURL       string `json:"base_url,omitempty"`       // Base URL for pagination links
	QueryParam    string `json:"query_param,omitempty"`    // Query parameter name for page (default: "page")
	PreserveQuery bool   `json:"preserve_query,omitempty"` // Whether to preserve other query parameters
}

// Renderer is the main struct for rendering tables
type Renderer struct {
	// Remove unused template field to fix lint warning
}

// NewRenderer creates a new table renderer instance
func NewRenderer() *Renderer {
	return &Renderer{}
}

// PaginationInfo holds information about current pagination state
type PaginationInfo struct {
	CurrentPage int
	TotalPages  int
	TotalRows   int
	PageSize    int
	StartRow    int
	EndRow      int
}

// calculatePagination calculates pagination information
func (r *Renderer) calculatePagination(totalRows int, pagination *Pagination) PaginationInfo {
	if pagination == nil || !pagination.Enabled || pagination.PageSize <= 0 {
		return PaginationInfo{
			CurrentPage: 1,
			TotalPages:  1,
			TotalRows:   totalRows,
			PageSize:    totalRows,
			StartRow:    1,
			EndRow:      totalRows,
		}
	}

	currentPage := pagination.CurrentPage
	if currentPage < 1 {
		currentPage = 1
	}

	totalPages := (totalRows + pagination.PageSize - 1) / pagination.PageSize
	if currentPage > totalPages {
		currentPage = totalPages
	}

	startRow := (currentPage-1)*pagination.PageSize + 1
	endRow := currentPage * pagination.PageSize
	if endRow > totalRows {
		endRow = totalRows
	}

	return PaginationInfo{
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		TotalRows:   totalRows,
		PageSize:    pagination.PageSize,
		StartRow:    startRow,
		EndRow:      endRow,
	}
}

// paginateRows returns the rows for the current page
func (r *Renderer) paginateRows(rows [][]interface{}, pagination *Pagination) [][]interface{} {
	if pagination == nil || !pagination.Enabled || pagination.PageSize <= 0 {
		return rows
	}

	totalRows := len(rows)
	paginationInfo := r.calculatePagination(totalRows, pagination)

	startIndex := paginationInfo.StartRow - 1
	endIndex := paginationInfo.EndRow

	if startIndex >= totalRows {
		return [][]interface{}{}
	}

	if endIndex > totalRows {
		endIndex = totalRows
	}

	return rows[startIndex:endIndex]
}

// generatePaginationHTML generates HTML for pagination controls
func (r *Renderer) generatePaginationHTML(paginationInfo PaginationInfo, pagination *Pagination) string {
	if paginationInfo.TotalPages <= 1 {
		return ""
	}

	// Set defaults for URL generation
	baseURL := pagination.BaseURL
	if baseURL == "" {
		baseURL = ""
	}
	queryParam := pagination.QueryParam
	if queryParam == "" {
		queryParam = "page"
	}

	// Helper function to generate URL for a page
	generateURL := func(page int) string {
		if baseURL == "" {
			return fmt.Sprintf("?%s=%d", queryParam, page)
		}
		if strings.Contains(baseURL, "?") {
			return fmt.Sprintf("%s&%s=%d", baseURL, queryParam, page)
		}
		return fmt.Sprintf("%s?%s=%d", baseURL, queryParam, page)
	}

	var html strings.Builder

	html.WriteString(`<nav aria-label="Table pagination">`)
	html.WriteString(`<ul class="pagination">`)

	// Previous button
	if paginationInfo.CurrentPage > 1 {
		html.WriteString(fmt.Sprintf(`<li class="page-item"><a class="page-link" href="%s">Previous</a></li>`,
			generateURL(paginationInfo.CurrentPage-1)))
	} else {
		html.WriteString(`<li class="page-item disabled"><span class="page-link">Previous</span></li>`)
	}

	// Page numbers
	start := 1
	end := paginationInfo.TotalPages

	// Show only 5 pages around current page for large datasets
	if paginationInfo.TotalPages > 5 {
		start = paginationInfo.CurrentPage - 2
		if start < 1 {
			start = 1
		}
		end = start + 4
		if end > paginationInfo.TotalPages {
			end = paginationInfo.TotalPages
			start = end - 4
			if start < 1 {
				start = 1
			}
		}
	}

	for i := start; i <= end; i++ {
		if i == paginationInfo.CurrentPage {
			html.WriteString(fmt.Sprintf(`<li class="page-item active"><span class="page-link">%d</span></li>`, i))
		} else {
			html.WriteString(fmt.Sprintf(`<li class="page-item"><a class="page-link" href="%s">%d</a></li>`,
				generateURL(i), i))
		}
	}

	// Next button
	if paginationInfo.CurrentPage < paginationInfo.TotalPages {
		html.WriteString(fmt.Sprintf(`<li class="page-item"><a class="page-link" href="%s">Next</a></li>`,
			generateURL(paginationInfo.CurrentPage+1)))
	} else {
		html.WriteString(`<li class="page-item disabled"><span class="page-link">Next</span></li>`)
	}

	html.WriteString(`</ul>`)
	html.WriteString(`</nav>`)

	return html.String()
}

// generatePaginationInfoHTML generates HTML showing pagination information
func (r *Renderer) generatePaginationInfoHTML(paginationInfo PaginationInfo) string {
	if paginationInfo.TotalRows == 0 {
		return `<div class="pagination-info">No records found</div>`
	}

	return fmt.Sprintf(`<div class="pagination-info">Showing %d to %d of %d entries</div>`,
		paginationInfo.StartRow, paginationInfo.EndRow, paginationInfo.TotalRows)
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

	// Calculate pagination info
	totalRows := len(rows)
	paginationInfo := r.calculatePagination(totalRows, data.Options.Pagination)

	// Apply pagination to rows
	paginatedRows := r.paginateRows(rows, data.Options.Pagination)

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

	// Enhanced HTML template with pagination support
	htmlTemplate := `
<div class="table-container">
{{if .ShowPaginationInfo}}{{.PaginationInfo}}{{end}}
<table class="{{.CSSClasses}}"{{if .ID}} id="{{.ID}}"{{end}}{{if .Style}} style="{{.Style}}"{{end}}>
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
</table>
{{if .ShowPaginationControls}}{{.PaginationControls}}{{end}}
</div>`

	tmpl, err := template.New("table").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Generate pagination HTML
	var paginationControls, paginationInfoHTML string
	var showPaginationControls, showPaginationInfo bool

	if data.Options.Pagination != nil && data.Options.Pagination.Enabled {
		showPaginationControls = data.Options.Pagination.ShowControls
		showPaginationInfo = data.Options.Pagination.ShowInfo

		if showPaginationControls {
			paginationControls = r.generatePaginationHTML(paginationInfo, data.Options.Pagination)
		}
		if showPaginationInfo {
			paginationInfoHTML = r.generatePaginationInfoHTML(paginationInfo)
		}
	}

	// Prepare template data
	templateData := struct {
		Headers                []string
		Rows                   [][]interface{}
		CSSClasses             string
		ID                     string
		Style                  template.CSS
		PaginationControls     template.HTML
		PaginationInfo         template.HTML
		ShowPaginationControls bool
		ShowPaginationInfo     bool
	}{
		Headers:                headers,
		Rows:                   paginatedRows,
		CSSClasses:             strings.Join(cssClasses, " "),
		ID:                     data.Options.ID,
		Style:                  template.CSS(data.Options.Style),
		PaginationControls:     template.HTML(paginationControls),
		PaginationInfo:         template.HTML(paginationInfoHTML),
		ShowPaginationControls: showPaginationControls,
		ShowPaginationInfo:     showPaginationInfo,
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

// RenderFullPage renders a complete HTML page with table and pagination (server-side only)
func (r *Renderer) RenderFullPage(data TableData, title string) (string, error) {
	tableHTML, err := r.RenderHTML(data)
	if err != nil {
		return "", err
	}

	pageTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.1.3/dist/css/bootstrap.min.css" rel="stylesheet">
    <style>
        .table-container {
            margin: 20px 0;
        }
        .pagination-info {
            margin-bottom: 10px;
            font-size: 14px;
            color: #666;
        }
        .pagination {
            justify-content: center;
            margin-top: 15px;
        }
        .pagination .page-link {
            color: #0d6efd;
            text-decoration: none;
        }
        .pagination .page-link:hover {
            color: #0a58ca;
            background-color: #e9ecef;
        }
    </style>
</head>
<body>
    <div class="container mt-4">
        <h1>{{.Title}}</h1>
        {{.TableHTML}}
        <div class="mt-3">
            <small class="text-muted">Server-side rendered pagination - no JavaScript required!</small>
        </div>
    </div>
</body>
</html>`

	tmpl, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse page template: %w", err)
	}

	templateData := struct {
		Title     string
		TableHTML template.HTML
	}{
		Title:     title,
		TableHTML: template.HTML(tableHTML),
	}

	var result strings.Builder
	err = tmpl.Execute(&result, templateData)
	if err != nil {
		return "", fmt.Errorf("failed to execute page template: %w", err)
	}

	return result.String(), nil
}

// NewPagination creates a new pagination configuration with sensible defaults
func NewPagination(pageSize int, currentPage int) *Pagination {
	if pageSize <= 0 {
		pageSize = 10
	}
	if currentPage <= 0 {
		currentPage = 1
	}

	return &Pagination{
		Enabled:       true,
		PageSize:      pageSize,
		CurrentPage:   currentPage,
		ShowControls:  true,
		ShowInfo:      true,
		BaseURL:       "",     // Will use current URL
		QueryParam:    "page", // Default query parameter
		PreserveQuery: true,   // Preserve other query parameters by default
	}
}

// NewPaginationWithURL creates pagination with custom URL configuration
func NewPaginationWithURL(pageSize int, currentPage int, baseURL string, queryParam string) *Pagination {
	pagination := NewPagination(pageSize, currentPage)
	pagination.BaseURL = baseURL
	if queryParam != "" {
		pagination.QueryParam = queryParam
	}
	return pagination
}

// SetPage creates a copy of TableData with a different current page
func (r *Renderer) SetPage(data TableData, page int) TableData {
	newData := data
	if newData.Options.Pagination != nil {
		newPagination := *newData.Options.Pagination
		newPagination.CurrentPage = page
		newData.Options.Pagination = &newPagination
	}
	return newData
}

// ParsePageFromQuery extracts page number from URL query string
// This is a helper function for web applications
func ParsePageFromQuery(queryString string, paramName string) int {
	if paramName == "" {
		paramName = "page"
	}

	// Simple query parameter parsing
	if queryString == "" {
		return 1
	}

	// Remove leading '?' if present
	queryString = strings.TrimPrefix(queryString, "?")

	// Split by '&' to get individual parameters
	params := strings.Split(queryString, "&")
	for _, param := range params {
		if strings.Contains(param, "=") {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) == 2 && parts[0] == paramName {
				if page, err := strconv.Atoi(parts[1]); err == nil && page > 0 {
					return page
				}
			}
		}
	}

	return 1
}

// CreatePaginatedData creates TableData with pagination based on URL query
func CreatePaginatedData(data interface{}, baseURL string, queryString string, pageSize int) TableData {
	currentPage := ParsePageFromQuery(queryString, "page")

	return TableData{
		Data: data,
		Options: TableOptions{
			Responsive: true,
			Striped:    true,
			Bordered:   true,
			Pagination: &Pagination{
				Enabled:       true,
				PageSize:      pageSize,
				CurrentPage:   currentPage,
				ShowControls:  true,
				ShowInfo:      true,
				BaseURL:       baseURL,
				QueryParam:    "page",
				PreserveQuery: true,
			},
		},
	}
}
