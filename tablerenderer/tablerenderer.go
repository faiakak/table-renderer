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
	Search     *Search     `json:"search,omitempty"`
}

// Pagination holds pagination configuration
type Pagination struct {
	Enabled         bool   `json:"enabled"`
	PageSize        int    `json:"page_size"`
	CurrentPage     int    `json:"current_page"`
	ShowControls    bool   `json:"show_controls"`
	ShowInfo        bool   `json:"show_info"`
	ShowPageSizer   bool   `json:"show_page_sizer,omitempty"`  // Show page size dropdown
	PageSizeOptions []int  `json:"page_size_options,omitempty"` // Available page size options
	BaseURL         string `json:"base_url,omitempty"`       // Base URL for pagination links
	QueryParam      string `json:"query_param,omitempty"`    // Query parameter name for page (default: "page")
	PreserveQuery   bool   `json:"preserve_query,omitempty"` // Whether to preserve other query parameters
	TotalCount      int    `json:"total_count,omitempty"`    // Total records (for database pagination)
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

// Search holds search configuration for server-side search
type Search struct {
	Enabled       bool     `json:"enabled"`
	SearchTerm    string   `json:"search_term,omitempty"`    // Current search term
	Placeholder   string   `json:"placeholder,omitempty"`    // Search input placeholder
	SearchColumns []string `json:"search_columns,omitempty"` // Columns to search (empty = all columns)
	CaseSensitive bool     `json:"case_sensitive,omitempty"` // Case sensitive search
	BaseURL       string   `json:"base_url,omitempty"`       // Base URL for search
	QueryParam    string   `json:"query_param,omitempty"`    // Query parameter name (default: "search")
	MinLength     int      `json:"min_length,omitempty"`     // Minimum search length (default: 1)
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

// generatePageSizeHTML generates HTML for page size dropdown
func (r *Renderer) generatePageSizeHTML(pagination *Pagination, currentQueryParams map[string]string) string {
	if pagination == nil || !pagination.ShowPageSizer {
		return ""
	}

	// Default page size options if not specified
	options := pagination.PageSizeOptions
	if len(options) == 0 {
		options = []int{10, 25, 50, 100}
	}

	// Set defaults for URL generation
	baseURL := pagination.BaseURL
	if baseURL == "" {
		baseURL = ""
	}

	// Helper function to generate URL for a page size while preserving other query parameters
	generateURL := func(pageSize int) string {
		params := make([]string, 0)
		
		// Add page size parameter
		params = append(params, fmt.Sprintf("page_size=%d", pageSize))
		
		// Reset to page 1 when changing page size
		params = append(params, "page=1")
		
		// Add other preserved parameters (except page and page_size)
		for key, value := range currentQueryParams {
			if key != "page" && key != "page_size" {
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
	html.WriteString(`<div class="page-size-control d-flex align-items-center mb-3">`)
	html.WriteString(`<label for="page-size-select" class="form-label me-2 mb-0">Show:</label>`)
	html.WriteString(`<select id="page-size-select" class="form-select form-select-sm" style="width: auto;" onchange="window.location.href=this.value">`)
	
	for _, size := range options {
		selected := ""
		if size == pagination.PageSize {
			selected = " selected"
		}
		html.WriteString(fmt.Sprintf(`<option value="%s"%s>%d entries</option>`, 
			generateURL(size), selected, size))
	}
	
	html.WriteString(`</select>`)
	html.WriteString(`</div>`)
	
	return html.String()
}

// generateSearchHTML generates HTML for search input
func (r *Renderer) generateSearchHTML(search *Search, currentQueryParams map[string]string) string {
	if search == nil || !search.Enabled {
		return ""
	}

	// Set defaults
	placeholder := search.Placeholder
	if placeholder == "" {
		placeholder = "Search all columns..."
	}
	queryParam := search.QueryParam
	if queryParam == "" {
		queryParam = "search"
	}
	baseURL := search.BaseURL
	if baseURL == "" {
		baseURL = ""
	}

	// Get current search term
	searchTerm := search.SearchTerm

	// Build form action URL with preserved parameters
	actionParams := make([]string, 0)
	for key, value := range currentQueryParams {
		if key != queryParam && key != "page" { // Exclude search param and reset page
			actionParams = append(actionParams, fmt.Sprintf("%s=%s", key, value))
		}
	}
	
	var actionURL string
	if len(actionParams) > 0 {
		queryString := strings.Join(actionParams, "&")
		if baseURL == "" {
			actionURL = "?" + queryString
		} else if strings.Contains(baseURL, "?") {
			actionURL = baseURL + "&" + queryString
		} else {
			actionURL = baseURL + "?" + queryString
		}
	} else {
		actionURL = baseURL
		if actionURL == "" {
			actionURL = ""
		}
	}

	var html strings.Builder
	html.WriteString(`<div class="search-control mb-3">`)
	html.WriteString(`<form method="GET" action="` + actionURL + `" class="d-flex align-items-center">`)
	
	// Add hidden fields for preserved parameters
	for key, value := range currentQueryParams {
		if key != queryParam && key != "page" {
			html.WriteString(fmt.Sprintf(`<input type="hidden" name="%s" value="%s">`, key, value))
		}
	}
	
	html.WriteString(`<div class="input-group" style="max-width: 300px;">`)
	html.WriteString(fmt.Sprintf(`<input type="text" name="%s" class="form-control" placeholder="%s" value="%s">`, 
		queryParam, placeholder, searchTerm))
	html.WriteString(`<button class="btn btn-outline-secondary" type="submit">`)
	html.WriteString(`<i class="fas fa-search"></i> Search`)
	html.WriteString(`</button>`)
	
	// Clear search button if there's a search term
	if searchTerm != "" {
		clearURL := actionURL
		if clearURL == "" {
			clearURL = "?"
		} else if !strings.Contains(clearURL, "?") {
			clearURL += "?"
		} else {
			clearURL += "&"
		}
		// Add preserved parameters for clear URL
		clearParams := make([]string, 0)
		for key, value := range currentQueryParams {
			if key != queryParam && key != "page" {
				clearParams = append(clearParams, fmt.Sprintf("%s=%s", key, value))
			}
		}
		if len(clearParams) > 0 {
			if strings.HasSuffix(clearURL, "?") {
				clearURL += strings.Join(clearParams, "&")
			} else {
				clearURL += strings.Join(clearParams, "&")
			}
		} else {
			clearURL = strings.TrimSuffix(clearURL, "?")
			clearURL = strings.TrimSuffix(clearURL, "&")
			if clearURL == "" {
				clearURL = "/"
			}
		}
		
		html.WriteString(fmt.Sprintf(`<a href="%s" class="btn btn-outline-danger" title="Clear search">`, clearURL))
		html.WriteString(`<i class="fas fa-times"></i>`)
		html.WriteString(`</a>`)
	}
	
	html.WriteString(`</div>`)
	html.WriteString(`</form>`)
	html.WriteString(`</div>`)

	return html.String()
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
{{if .ShowSearch}}{{.SearchHTML}}{{end}}
<div class="d-flex justify-content-between align-items-center mb-2">
	<div>{{if .ShowPageSizer}}{{.PageSizerHTML}}{{end}}</div>
	<div>{{if .ShowPaginationInfo}}{{.PaginationInfo}}{{end}}</div>
</div>
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
			// Add current page size to preserve it in pagination links
			if paginationInfo.PageSize > 0 {
				currentParams["page_size"] = fmt.Sprintf("%d", paginationInfo.PageSize)
			}
			// Add current search term to preserve it in pagination links
			if data.Options.Search != nil && data.Options.Search.Enabled && data.Options.Search.SearchTerm != "" {
				searchParam := data.Options.Search.QueryParam
				if searchParam == "" {
					searchParam = "search"
				}
				currentParams[searchParam] = data.Options.Search.SearchTerm
			}
			paginationControls = r.generatePaginationHTML(paginationInfo, data.Options.Pagination, currentParams)
		}
		if showPaginationInfo {
			paginationInfoHTML = r.generatePaginationInfoHTML(paginationInfo)
		}
	}

	// Generate page size control HTML
	var pageSizerHTML string
	var showPageSizer bool

	if data.Options.Pagination != nil && data.Options.Pagination.Enabled && data.Options.Pagination.ShowPageSizer {
		showPageSizer = true
		// Parse current query parameters to preserve them in page size links
		currentParams := r.parseQueryParams(data.Options.Pagination.BaseURL)
		if data.Options.Sorting != nil && data.Options.Sorting.Enabled {
			if data.Options.Sorting.SortBy != "" {
				currentParams["sort_by"] = data.Options.Sorting.SortBy
			}
			if data.Options.Sorting.SortOrder != "" {
				currentParams["sort_order"] = data.Options.Sorting.SortOrder
			}
		}
		// Add current search term to preserve it in page size links
		if data.Options.Search != nil && data.Options.Search.Enabled && data.Options.Search.SearchTerm != "" {
			searchParam := data.Options.Search.QueryParam
			if searchParam == "" {
				searchParam = "search"
			}
			currentParams[searchParam] = data.Options.Search.SearchTerm
		}
		pageSizerHTML = r.generatePageSizeHTML(data.Options.Pagination, currentParams)
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
			// Add current page size to preserve it in sorting links
			if paginationInfo.PageSize > 0 {
				currentParams["page_size"] = fmt.Sprintf("%d", paginationInfo.PageSize)
			}
		}
		// Add current search term to preserve it in sorting links
		if data.Options.Search != nil && data.Options.Search.Enabled && data.Options.Search.SearchTerm != "" {
			searchParam := data.Options.Search.QueryParam
			if searchParam == "" {
				searchParam = "search"
			}
			currentParams[searchParam] = data.Options.Search.SearchTerm
		}
		
		sortLinks = r.generateSortLinks(headers, data.Options.Sorting, currentParams)
	} else {
		// Create empty sort links for non-sortable tables
		sortLinks = make([]string, len(headers))
	}

	// Generate search control HTML
	var searchHTML string
	var showSearch bool
	var currentSearchTerm string

	if data.Options.Search != nil && data.Options.Search.Enabled {
		showSearch = true
		currentSearchTerm = data.Options.Search.SearchTerm
		// Parse current query parameters to preserve them in search
		currentParams := r.parseQueryParams(data.Options.Search.BaseURL)
		if data.Options.Sorting != nil && data.Options.Sorting.Enabled {
			if data.Options.Sorting.SortBy != "" {
				currentParams["sort_by"] = data.Options.Sorting.SortBy
			}
			if data.Options.Sorting.SortOrder != "" {
				currentParams["sort_order"] = data.Options.Sorting.SortOrder
			}
		}
		if data.Options.Pagination != nil && data.Options.Pagination.Enabled {
			// Add current page size to preserve it in search
			if paginationInfo.PageSize > 0 {
				currentParams["page_size"] = fmt.Sprintf("%d", paginationInfo.PageSize)
			}
		}
		searchHTML = r.generateSearchHTML(data.Options.Search, currentParams)
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
		PageSizerHTML          template.HTML
		ShowPageSizer          bool
		SearchHTML             template.HTML
		ShowSearch             bool
		CurrentSearchTerm      string
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
		PageSizerHTML:          template.HTML(pageSizerHTML),
		ShowPageSizer:          showPageSizer,
		SearchHTML:             template.HTML(searchHTML),
		ShowSearch:             showSearch,
		CurrentSearchTerm:      currentSearchTerm,
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

// ParsePageSizeFromQuery extracts page size from URL query string
// This is a helper function for web applications
func ParsePageSizeFromQuery(queryString string, defaultPageSize int) int {
	// Simple query parameter parsing
	if queryString == "" {
		return defaultPageSize
	}

	// Remove leading '?' if present
	queryString = strings.TrimPrefix(queryString, "?")

	// Split by '&' to get individual parameters
	params := strings.Split(queryString, "&")
	for _, param := range params {
		if strings.Contains(param, "=") {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) == 2 && parts[0] == "page_size" {
				if pageSize, err := strconv.Atoi(parts[1]); err == nil && pageSize > 0 {
					return pageSize
				}
			}
		}
	}

	return defaultPageSize
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
				Enabled:         true,
				PageSize:        pageSize,
				CurrentPage:     currentPage,
				ShowControls:    true,
				ShowInfo:        true,
				ShowPageSizer:   true,                          // Enable page size control
				PageSizeOptions: []int{10, 25, 50, 100},      // Default page size options
				BaseURL:         baseURL, // Use base URL without query params for pagination
				QueryParam:      "page",
				PreserveQuery:   true,
				TotalCount:      totalCount, // Important: set total count for database pagination
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

// CreatePaginatedDataWithSortingAndSearch creates database pagination data with sorting and search
func CreatePaginatedDataWithSortingAndSearch(data interface{}, totalCount int, baseURL string, queryString string, pageSize int, enableSorting bool, enableSearch bool, searchTerm string) DatabasePaginatedData {
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
				Enabled:         true,
				PageSize:        pageSize,
				CurrentPage:     currentPage,
				ShowControls:    true,
				ShowInfo:        true,
				ShowPageSizer:   true,                          // Enable page size control
				PageSizeOptions: []int{10, 25, 50, 100},      // Default page size options
				BaseURL:         baseURL, // Use base URL without query params for pagination
				QueryParam:      "page",
				PreserveQuery:   true,
				TotalCount:      totalCount, // Important: set total count for database pagination
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

	if enableSearch {
		result.Options.Search = &Search{
			Enabled:     true,
			SearchTerm:  searchTerm,
			Placeholder: "Search all columns...",
			BaseURL:     baseURL, // Use base URL without query params for search
			QueryParam:  "search",
			MinLength:   1,
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

// ParseSearchFromQuery parses search term from query string
func ParseSearchFromQuery(rawQuery string, defaultSearchParam string) string {
	if rawQuery == "" {
		return ""
	}

	searchParam := defaultSearchParam
	if searchParam == "" {
		searchParam = "search"
	}

	// Parse the raw query string
	params := strings.Split(rawQuery, "&")
	for _, param := range params {
		if strings.Contains(param, "=") {
			parts := strings.SplitN(param, "=", 2)
			if len(parts) == 2 && parts[0] == searchParam {
				// URL decode the search term (basic decoding)
				decoded := strings.Replace(parts[1], "+", " ", -1)
				return decoded
			}
		}
	}

	return ""
}
