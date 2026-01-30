package docling

// DoclingDocument represents the complete parsed document from Docling
type DoclingDocument struct {
	Name        string              `json:"name"`
	Origin      Origin              `json:"origin"`
	Texts       []TextItem          `json:"texts"`
	Tables      []TableItem         `json:"tables,omitempty"`
	Pictures    []PictureItem       `json:"pictures,omitempty"`
	PageHeaders map[string]PageInfo `json:"page_headers,omitempty"`
}

// Origin contains document source information
type Origin struct {
	Filename  string `json:"filename"`
	Mimetype  string `json:"mimetype"`
	URI       string `json:"uri,omitempty"`
	PageCount int    `json:"page_count,omitempty"`
}

// TextItem represents a text element in the document
type TextItem struct {
	SelfRef  string       `json:"self_ref"` // Reference ID e.g., "#/texts/42"
	Text     string       `json:"text"`     // Actual text content
	Label    string       `json:"label"`    // Type: paragraph, title, section_header, etc.
	Prov     []Provenance `json:"prov"`     // Location information
	Parent   *RefItem     `json:"parent,omitempty"`
	Children []RefItem    `json:"children,omitempty"`
}

// RefItem is a reference to another item
type RefItem struct {
	Ref string `json:"$ref"`
}

// Provenance contains location information for a text item
type Provenance struct {
	PageNo int   `json:"page_no"`
	BBox   *BBox `json:"bbox,omitempty"`
}

// BBox represents a bounding box in the document
type BBox struct {
	L           float64 `json:"l"`
	T           float64 `json:"t"`
	R           float64 `json:"r"`
	B           float64 `json:"b"`
	CoordOrigin string  `json:"coord_origin,omitempty"`
}

// TableItem represents a table in the document
type TableItem struct {
	SelfRef string       `json:"self_ref"`
	Data    TableData    `json:"data"`
	Prov    []Provenance `json:"prov"`
}

// TableData contains the table structure
type TableData struct {
	Rows  int          `json:"num_rows"`
	Cols  int          `json:"num_cols"`
	Cells [][]CellData `json:"cells"`
}

// CellData represents a table cell
type CellData struct {
	Text    string `json:"text"`
	RowSpan int    `json:"row_span,omitempty"`
	ColSpan int    `json:"col_span,omitempty"`
}

// PictureItem represents an image in the document
type PictureItem struct {
	SelfRef string       `json:"self_ref"`
	Prov    []Provenance `json:"prov"`
}

// PageInfo contains page-level metadata
type PageInfo struct {
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// ConvertResponse is the response from the Docling convert endpoint
type ConvertResponse struct {
	Document DoclingDocument `json:"document"`
	Status   string          `json:"status"`
	Errors   []string        `json:"errors,omitempty"`
}

// HealthResponse is the response from the health endpoint
type HealthResponse struct {
	Status string `json:"status"`
}
