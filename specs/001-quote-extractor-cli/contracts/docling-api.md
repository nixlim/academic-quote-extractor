# Docling-serve API Contract

**Service**: Docling-serve
**Base URL**: http://localhost:5001
**Version**: latest (quay.io/docling-project/docling-serve)

## Endpoints

### POST /v1/convert/file

Convert a document file to DoclingDocument JSON.

**Request:**
- Content-Type: `multipart/form-data`
- Body: File upload with key `file`

**Request Example (curl):**
```bash
curl -X POST http://localhost:5001/v1/convert/file \
  -F "file=@document.pdf"
```

**Response (200 OK):**
```json
{
  "name": "document.pdf",
  "origin": {
    "filename": "document.pdf",
    "mimetype": "application/pdf"
  },
  "texts": [
    {
      "self_ref": "#/texts/0",
      "text": "Chapter 1: Introduction",
      "label": "section_header",
      "prov": [
        {
          "page_no": 1,
          "bbox": {
            "l": 72.0,
            "t": 720.0,
            "r": 300.0,
            "b": 740.0,
            "coord_origin": "BOTTOMLEFT"
          },
          "charspan": [0, 23]
        }
      ]
    },
    {
      "self_ref": "#/texts/1",
      "text": "This thesis examines the impact of social media on political discourse...",
      "label": "paragraph",
      "prov": [
        {
          "page_no": 1,
          "bbox": {
            "l": 72.0,
            "t": 650.0,
            "r": 540.0,
            "b": 700.0,
            "coord_origin": "BOTTOMLEFT"
          },
          "charspan": [0, 75]
        }
      ]
    }
  ],
  "tables": [],
  "pictures": [],
  "body": {
    "self_ref": "#/body",
    "children": [
      {"$ref": "#/texts/0"},
      {"$ref": "#/texts/1"}
    ]
  }
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Original filename |
| `origin.filename` | string | Source filename |
| `origin.mimetype` | string | Detected MIME type |
| `texts` | array | Array of TextItem objects |
| `texts[].self_ref` | string | Unique reference ID (e.g., "#/texts/42") |
| `texts[].text` | string | Extracted text content |
| `texts[].label` | string | Element type: paragraph, section_header, list_item, etc. |
| `texts[].prov` | array | Provenance information |
| `texts[].prov[].page_no` | int | 1-indexed page number |
| `texts[].prov[].bbox` | object | Bounding box coordinates |
| `tables` | array | Extracted tables (not used in MVP) |
| `pictures` | array | Extracted images (not used in MVP) |
| `body` | object | Document structure tree |

**Error Responses:**

| Status | Description |
|--------|-------------|
| 400 | Invalid file or unsupported format |
| 500 | Processing error |

**Error Response Example:**
```json
{
  "detail": "Unsupported file format: .xyz"
}
```

---

### GET /health

Health check endpoint.

**Response (200 OK):**
```json
{
  "status": "healthy"
}
```

---

## Go Client Interface

```go
// internal/docling/client.go

type Client struct {
    baseURL    string
    httpClient *http.Client
}

type DoclingDocument struct {
    Name   string     `json:"name"`
    Origin Origin     `json:"origin"`
    Texts  []TextItem `json:"texts"`
}

type Origin struct {
    Filename string `json:"filename"`
    Mimetype string `json:"mimetype"`
}

type TextItem struct {
    SelfRef string       `json:"self_ref"`
    Text    string       `json:"text"`
    Label   string       `json:"label"`
    Prov    []Provenance `json:"prov"`
}

type Provenance struct {
    PageNo  int     `json:"page_no"`
    BBox    BBox    `json:"bbox"`
    Charspan []int  `json:"charspan"`
}

type BBox struct {
    L           float64 `json:"l"`
    T           float64 `json:"t"`
    R           float64 `json:"r"`
    B           float64 `json:"b"`
    CoordOrigin string  `json:"coord_origin"`
}

// ConvertFile uploads a file and returns the parsed document
func (c *Client) ConvertFile(filepath string) (*DoclingDocument, error)

// Health checks if the service is available
func (c *Client) Health() error
```

---

## Contract Test Cases

```go
// tests/contract/docling_test.go

func TestDoclingConvertPDF(t *testing.T) {
    // Given: A valid PDF file
    // When: POST /v1/convert/file
    // Then: Response contains texts[] with self_ref and prov
}

func TestDoclingConvertDOCX(t *testing.T) {
    // Given: A valid DOCX file
    // When: POST /v1/convert/file
    // Then: Response contains texts[] with self_ref and prov
}

func TestDoclingUnsupportedFormat(t *testing.T) {
    // Given: An unsupported file format
    // When: POST /v1/convert/file
    // Then: Response is 400 with error detail
}

func TestDoclingHealth(t *testing.T) {
    // Given: Docling service is running
    // When: GET /health
    // Then: Response is 200 with status "healthy"
}
```
