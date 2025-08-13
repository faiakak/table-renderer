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

// DatabasePaginatedData represents data for database-level pagination
// This is more efficient as it only contains the current page data
type DatabasePaginatedData struct {
	Headers    []string        `json:"headers"`
	Rows       [][]interface{} `json:"rows"`
	Data       interface{}     `json:"data,omitempty"` // Current page data only
	TotalCount int             `json:"total_count"`    // Total number of records in database
	Options    TableOptions    `json:"options,omitempty"`
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
	Sorting    *Sorting    `json:"sorting,omitempty"`
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
	TotalCount    int    `json:"total_count,omitempty"`    // Total records (for database pagination)
}

// Sorting holds sorting configuration for server-side sorting
type Sorting struct {
	Enabled     bool   `json:"enabled"`
	SortBy      string `json:"sort_by,omitempty"`      // Field name to sort by
	SortOrder   string `json:"sort_order,omitempty"`   // "asc" or "desc"
	BaseURL     string `json:"base_url,omitempty"`     // Base URL for sorting links
	QueryParam  string `json:"query_param,omitempty"`  // Query parameter name for sort (default: "sort_by")
	OrderParam  string `json:"order_param,omitempty"`  // Query parameter name for order (default: "sort_order")
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

// calculatePagination calculates pagination for database-level pagination
// where we know the total count but only have current page data
func (r *Renderer) calculatePagination(currentPageDataCount int, pagination *Pagination) PaginationInfo {
	if pagination == nil || !pagination.Enabled || pagination.PageSize <= 0 {
		return PaginationInfo{
			CurrentPage: 1,
			TotalPages:  1,
			TotalRows:   currentPageDataCount,
			PageSize:    currentPageDataCount,
			StartRow:    1,
			EndRow:      currentPageDataCount,
		}
	}

	// Use TotalCount from pagination config for database pagination
	totalRows := pagination.TotalCount
	if totalRows == 0 {
		totalRows = currentPageDataCount
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

// generatePaginationHTML generates HTML for pagination controls
func (r *Renderer) generatePaginationHTML(paginationInfo PaginationInfo, pagination *Pagination, currentQueryParams map[string]string) string {
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

	// Helper function to generate URL for a page while preserving other query parameters
	generateURL := func(page int) string {
		params := make([]string, 0)
		
		// Add page parameter
		params = append(params, fmt.Sprintf("%s=%d", queryParam, page))
		
		// Add other preserved parameters (like sorting)
		for key, value := range currentQueryParams {
			if key != queryParam { // Don't duplicate page param
				params = append(params, fmt.Sprintf("%s=%s", key, value))
			}
		}
		
		queryString := strings.Join(params, "&")
		
		if baseURL == "" {
			return "?" + queryString
		}
		if strings.Contains(baseURL, "?") {
			return baseURL + "&" + queryString
		}
		return baseURL + "?" + queryString
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

// RenderHTML renders table data with database-level pagination
// This method expects only the current page data and uses TotalCount from pagination config
func (r *Renderer) RenderHTML(data DatabasePaginatedData) (string, error) {
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

	// Calculate pagination info using database pagination method
	currentPageDataCount := len(rows)
	paginationInfo := r.calculatePagination(currentPageDataCount, data.Options.Pagination)

	// For database pagination, we don't paginate the rows (they're already paginated)
	// We use the rows as-is since they represent only the current page

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

	// Enhanced HTML template with pagination and sorting support
	htmlTemplate := `
<div class="table-container">
{{if .ShowPaginationInfo}}{{.PaginationInfo}}{{end}}
<table class="{{.CSSClasses}}"{{if .ID}} id="{{.ID}}"{{end}}{{if .Style}} style="{{.Style}}"{{end}}>
	<thead>
		<tr>
			{{range $index, $header := .Headers}}
			<th>
				{{if $.SortingEnabled}}
					<a href="{{index $.SortLinks $index}}" style="text-decoration: none; color: inherit;">
						{{$header}}
						{{if eq $.CurrentSortBy $header}}
							{{if eq $.CurrentSortOrder "asc"}}
								<span style="font-size: 0.8em;">▲</span>
							{{else}}
								<span style="font-size: 0.8em;">▼</span>
							{{end}}
						{{else}}
							<span style="font-size: 0.8em; color: #ccc;">⬍</span>
						{{end}}
					</a>
				{{else}}
					{{$header}}
				{{end}}
			</th>
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
			// Parse current query parameters to preserve them in pagination links
			currentParams := r.parseQueryParams(data.Options.Pagination.BaseURL)
			if data.Options.Sorting != nil && data.Options.Sorting.Enabled {
				if data.Options.Sorting.SortBy != "" {
					currentParams["sort_by"] = data.Options.Sorting.SortBy
				}
				if data.Options.Sorting.SortOrder != "" {
					currentParams["sort_order"] = data.Options.Sorting.SortOrder
				}
			}
			paginationControls = r.generatePaginationHTML(paginationInfo, data.Options.Pagination, currentParams)
		}
		if showPaginationInfo {
			paginationInfoHTML = r.generatePaginationInfoHTML(paginationInfo)
		}
	}

	// Generate sorting links and data
	var sortLinks []string
	var sortingEnabled bool
	var currentSortBy, currentSortOrder string

	if data.Options.Sorting != nil && data.Options.Sorting.Enabled {
		sortingEnabled = true
		currentSortBy = data.Options.Sorting.SortBy
		currentSortOrder = data.Options.Sorting.SortOrder
		
		// Parse current query parameters to preserve them in sorting links
		currentParams := r.parseQueryParams(data.Options.Sorting.BaseURL)
		if data.Options.Pagination != nil && data.Options.Pagination.Enabled {
			// Preserve current page in sorting links
			currentParams["page"] = fmt.Sprintf("%d", data.Options.Pagination.CurrentPage)
		}
		
		sortLinks = r.generateSortLinks(headers, data.Options.Sorting, currentParams)
	} else {
		// Create empty sort links for non-sortable tables
		sortLinks = make([]string, len(headers))
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
		SortingEnabled         bool
		SortLinks              []string
		CurrentSortBy          string
		CurrentSortOrder       string
	}{
		Headers:                headers,
		Rows:                   rows, // Use rows as-is (already paginated at database level)
		CSSClasses:             strings.Join(cssClasses, " "),
		ID:                     data.Options.ID,
		Style:                  template.CSS(data.Options.Style),
		PaginationControls:     template.HTML(paginationControls),
		PaginationInfo:         template.HTML(paginationInfoHTML),
		ShowPaginationControls: showPaginationControls,
		ShowPaginationInfo:     showPaginationInfo,
		SortingEnabled:         sortingEnabled,
		SortLinks:              sortLinks,
		CurrentSortBy:          currentSortBy,
		CurrentSortOrder:       currentSortOrder,
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

// CreatePaginatedData creates DatabasePaginatedData for database-level pagination
func CreatePaginatedData(data interface{}, totalCount int, baseURL string, queryString string, pageSize int) DatabasePaginatedData {
	currentPage := ParsePageFromQuery(queryString, "page")

	return DatabasePaginatedData{
		Data:       data, // Only current page data
		TotalCount: totalCount,
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
				TotalCount:    totalCount, // Important: set total count for database pagination
			},
		},
	}
}

// CreatePaginatedDataWithSorting creates DatabasePaginatedData with both pagination and sorting support
func CreatePaginatedDataWithSorting(data interface{}, totalCount int, baseURL string, queryString string, pageSize int, enableSorting bool) DatabasePaginatedData {
	currentPage := ParsePageFromQuery(queryString, "page")
	sortBy, sortOrder := ParseSortFromQuery(queryString, "sort_by", "sort_order")

	result := DatabasePaginatedData{
		Data:       data, // Only current page data
		TotalCount: totalCount,
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
				BaseURL:       baseURL, // Use base URL without query params for pagination
				QueryParam:    "page",
				PreserveQuery: true,
				TotalCount:    totalCount, // Important: set total count for database pagination
			},
		},
	}

	if enableSorting {
		result.Options.Sorting = &Sorting{
			Enabled:    true,
			SortBy:     sortBy,
			SortOrder:  sortOrder,
			BaseURL:    baseURL, // Use base URL without query params for sorting
			QueryParam: "sort_by",
			OrderParam: "sort_order",
		}
	}

	return result
}

// CalculateDatabaseOffset calculates OFFSET for database queries
func CalculateDatabaseOffset(page int, pageSize int) int {
	if page < 1 {
		page = 1
	}
	return (page - 1) * pageSize
}

// CalculateDatabaseLimit returns the LIMIT for database queries (same as page size)
func CalculateDatabaseLimit(pageSize int) int {
	if pageSize <= 0 {
		return 10 // default page size
	}
	return pageSize
}

// generateSortLinks generates sorting URLs for each column header
func (r *Renderer) generateSortLinks(headers []string, sorting *Sorting, currentQueryParams map[string]string) []string {
	if sorting == nil || !sorting.Enabled {
		return make([]string, len(headers))
	}

	// Set defaults for URL generation
	baseURL := sorting.BaseURL
	if baseURL == "" {
		baseURL = ""
	}
	sortParam := sorting.QueryParam
	if sortParam == "" {
		sortParam = "sort_by"
	}
	orderParam := sorting.OrderParam
	if orderParam == "" {
		orderParam = "sort_order"
	}

	sortLinks := make([]string, len(headers))

	for i, header := range headers {
		// Determine sort order for this column
		sortOrder := "asc"
		if sorting.SortBy == header && sorting.SortOrder == "asc" {
			sortOrder = "desc" // Toggle to desc if already sorting asc
		}

		// Build parameters list preserving existing ones (except page - sorting resets to page 1)
		params := make([]string, 0)
		
		// Add sorting parameters
		params = append(params, fmt.Sprintf("%s=%s", sortParam, header))
		params = append(params, fmt.Sprintf("%s=%s", orderParam, sortOrder))
		
		// Add other preserved parameters but exclude page and sort params
		for key, value := range currentQueryParams {
			if key != sortParam && key != orderParam && key != "page" { // Exclude page to reset pagination
				params = append(params, fmt.Sprintf("%s=%s", key, value))
			}
		}
		
		queryString := strings.Join(params, "&")

		// Generate URL for this column
		if baseURL == "" {
			sortLinks[i] = "?" + queryString
		} else if strings.Contains(baseURL, "?") {
			sortLinks[i] = baseURL + "&" + queryString
		} else {
			sortLinks[i] = baseURL + "?" + queryString
		}
	}

	return sortLinks
}

// parseQueryParams extracts query parameters from a URL or query string
func (r *Renderer) parseQueryParams(urlOrQuery string) map[string]string {
	params := make(map[string]string)
	
	if urlOrQuery == "" {
		return params
	}
	
	// Extract query part if it's a full URL
	queryString := urlOrQuery
	if strings.Contains(urlOrQuery, "?") {
		parts := strings.Split(urlOrQuery, "?")
		if len(parts) > 1 {
			queryString = parts[1]
		}
	}
	
	// Remove leading '?' if present
	queryString = strings.TrimPrefix(queryString, "?")
	
	if queryString == "" {
		return params
	}
	
	// Split by '&' to get individual parameters
	pairs := strings.Split(queryString, "&")
	for _, pair := range pairs {
		if strings.Contains(pair, "=") {
			parts := strings.SplitN(pair, "=", 2)
			if len(parts) == 2 {
				params[parts[0]] = parts[1]
			}
		}
	}
	
	return params
}

// ParseSortFromQuery extracts sort field and order from URL query string
// This is a helper function for web applications
func ParseSortFromQuery(queryString string, sortParam string, orderParam string) (string, string) {
	if sortParam == "" {
		sortParam = "sort_by"
	}
	if orderParam == "" {
		orderParam = "sort_order"
	}

	// Simple query parameter parsing
	if queryString == "" {
		return "", "asc"
	}

	// Remove leading '?' if present
	queryString = strings.TrimPrefix(queryString, "?")

	var sortBy, sortOrder string
	sortOrder = "asc" // default

	// Split by '&' to get individual parameters
	params := strings.Split(queryString, "&")
	for _, param := range params {
		if strings.Contains(param, "=") {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) == 2 {
				if parts[0] == sortParam {
					sortBy = parts[1]
				} else if parts[0] == orderParam {
					if parts[1] == "desc" {
						sortOrder = "desc"
					}
				}
			}
		}
	}

	return sortBy, sortOrder
}
