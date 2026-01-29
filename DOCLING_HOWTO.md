# Docling How-To Guide

Docling is a powerful document processing library that simplifies parsing diverse formats (including advanced PDF understanding) and provides seamless integrations with the gen AI ecosystem.

**Version**: 2.70.0  
**Documentation**: https://docling-project.github.io/docling/  
**GitHub**: https://github.com/docling-project/docling

---

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Supported Formats](#supported-formats)
4. [Basic Usage](#basic-usage)
5. [PDF Data Extraction](#pdf-data-extraction)
6. [Advanced Configuration](#advanced-configuration)
7. [OCR Options](#ocr-options)
8. [Exporting Data](#exporting-data)
9. [CLI Usage](#cli-usage)
10. [Integrations](#integrations)

---

## Installation

### Basic Installation

```bash
pip install docling
```

**Requirements**: Python 3.10 or higher (Python 3.9 support dropped in v2.70.0)

**Platforms**: macOS, Linux, Windows (x86_64 and arm64)

### Optional Dependencies

```bash
# For Tesseract OCR
pip install docling[tesseract]

# For additional OCR backends
pip install docling[ocr]
```

---

## Quick Start

### Minimal Python Example

```python
from docling.document_converter import DocumentConverter

# Source can be a local file path or URL
source = "https://arxiv.org/pdf/2408.09869"

# Create converter and convert
converter = DocumentConverter()
result = converter.convert(source)

# Export to Markdown
print(result.document.export_to_markdown())
```

### CLI Quick Start

```bash
# Convert a PDF to Markdown
docling https://arxiv.org/pdf/2206.01062

# Use Vision Language Model (VLM) pipeline
docling --pipeline vlm --vlm-model granite_docling https://arxiv.org/pdf/2206.01062
```

---

## Supported Formats

### Input Formats

| Format | Description |
|--------|-------------|
| PDF | Full support with layout analysis, OCR, table detection |
| DOCX, XLSX, PPTX | MS Office 2007+ formats |
| Markdown | Plain text markup |
| AsciiDoc | Technical documentation format |
| HTML, XHTML | Web pages |
| CSV | Comma-separated values |
| PNG, JPEG, TIFF, BMP, WEBP | Image formats |
| WebVTT | Video text tracks |
| USPTO XML | Patent documents |
| JATS XML | Academic articles |

### Output Formats

| Format | Description |
|--------|-------------|
| Markdown | Human-readable markup |
| HTML | With image embedding or referencing |
| JSON | Lossless serialization of DoclingDocument |
| Text | Plain text without formatting |
| DocTags | Efficient markup for document content/layout |

---

## Basic Usage

### Converting a Single Document

```python
from docling.document_converter import DocumentConverter

# From local file
converter = DocumentConverter()
result = converter.convert("/path/to/document.pdf")

# From URL
result = converter.convert("https://example.com/document.pdf")

# Access the converted document
doc = result.document
```

### Converting Multiple Documents (Batch)

```python
from docling.document_converter import DocumentConverter

converter = DocumentConverter()
sources = [
    "document1.pdf",
    "document2.pdf",
    "https://example.com/document3.pdf"
]

# Convert all documents
for source in sources:
    result = converter.convert(source)
    print(f"Converted: {result.input.file}")
```

### Converting from Binary Stream

```python
from io import BytesIO
from docling.datamodel.base_models import DocumentStream
from docling.document_converter import DocumentConverter

# Your binary PDF data
buf = BytesIO(your_binary_stream)
source = DocumentStream(name="my_doc.pdf", stream=buf)

converter = DocumentConverter()
result = converter.convert(source)
```

---

## PDF Data Extraction

Docling excels at extracting structured data from PDFs. Here's how to extract different types of content:

### Extract All Text

```python
from docling.document_converter import DocumentConverter

converter = DocumentConverter()
result = converter.convert("document.pdf")

# Get plain text
text = result.document.export_to_markdown(strict_text=True)
print(text)
```

### Extract Tables

```python
import pandas as pd
from docling.document_converter import DocumentConverter

converter = DocumentConverter()
result = converter.convert("document.pdf")

# Iterate through all tables
for idx, table in enumerate(result.document.tables):
    # Export to pandas DataFrame
    df = table.export_to_dataframe(doc=result.document)
    
    # Save as CSV
    df.to_csv(f"table_{idx}.csv")
    
    # Save as HTML
    with open(f"table_{idx}.html", "w") as f:
        f.write(table.export_to_html(doc=result.document))
    
    # Print as Markdown
    print(df.to_markdown())
```

### Extract Images/Figures

```python
from docling.document_converter import DocumentConverter

converter = DocumentConverter()
result = converter.convert("document.pdf")

# Access all pictures/figures
for idx, picture in enumerate(result.document.pictures):
    print(f"Picture {idx}: {picture}")
    # Pictures include bounding box information
```

### Access Document Structure

```python
from docling.document_converter import DocumentConverter

converter = DocumentConverter()
result = converter.convert("document.pdf")
doc = result.document

# Access text items (paragraphs, headings, etc.)
for text_item in doc.texts:
    print(f"Type: {text_item.label}, Text: {text_item.text[:50]}...")

# Access document hierarchy
# doc.body - main document content tree
# doc.furniture - headers, footers, etc.
# doc.groups - container items like lists, chapters
```

---

## Advanced Configuration

### Configure PDF Pipeline

```python
from docling.datamodel.base_models import InputFormat
from docling.datamodel.pipeline_options import PdfPipelineOptions, TableStructureOptions
from docling.document_converter import DocumentConverter, PdfFormatOption

# Create pipeline options
pipeline_options = PdfPipelineOptions()
pipeline_options.do_ocr = True
pipeline_options.do_table_structure = True
pipeline_options.table_structure_options = TableStructureOptions(
    do_cell_matching=True
)

# Create converter with options
converter = DocumentConverter(
    format_options={
        InputFormat.PDF: PdfFormatOption(pipeline_options=pipeline_options)
    }
)

result = converter.convert("document.pdf")
```

### Set Page/File Limits

```python
from docling.document_converter import DocumentConverter

converter = DocumentConverter()

# Limit pages and file size
result = converter.convert(
    "large_document.pdf",
    max_num_pages=100,
    max_file_size=20971520  # 20MB
)
```

### Use Accelerator Options

```python
from docling.datamodel.accelerator_options import AcceleratorDevice, AcceleratorOptions
from docling.datamodel.pipeline_options import PdfPipelineOptions

pipeline_options = PdfPipelineOptions()
pipeline_options.accelerator_options = AcceleratorOptions(
    num_threads=4,
    device=AcceleratorDevice.AUTO  # AUTO, CPU, CUDA, MPS
)
```

### Table Structure Options

```python
from docling.datamodel.pipeline_options import PdfPipelineOptions, TableStructureOptions, TableFormerMode

pipeline_options = PdfPipelineOptions()
pipeline_options.do_table_structure = True
pipeline_options.table_structure_options = TableStructureOptions(
    do_cell_matching=True,  # Map structure to PDF cells
    mode=TableFormerMode.ACCURATE  # FAST or ACCURATE
)
```

---

## OCR Options

### Enable OCR with EasyOCR (Default)

```python
from docling.datamodel.pipeline_options import PdfPipelineOptions

pipeline_options = PdfPipelineOptions()
pipeline_options.do_ocr = True
pipeline_options.ocr_options.lang = ["en"]  # Language codes: ["en"], ["es"], ["en", "de"]
```

### Use Tesseract OCR

```python
from docling.datamodel.pipeline_options import PdfPipelineOptions, TesseractOcrOptions

pipeline_options = PdfPipelineOptions()
pipeline_options.do_ocr = True
pipeline_options.ocr_options = TesseractOcrOptions()
```

### Use macOS Native OCR (OcrMac)

```python
from docling.datamodel.pipeline_options import PdfPipelineOptions, OcrMacOptions

pipeline_options = PdfPipelineOptions()
pipeline_options.do_ocr = True
pipeline_options.ocr_options = OcrMacOptions()
```

### Disable OCR

```python
pipeline_options = PdfPipelineOptions()
pipeline_options.do_ocr = False
```

---

## Exporting Data

### Export to Multiple Formats

```python
import json
from docling.document_converter import DocumentConverter

converter = DocumentConverter()
result = converter.convert("document.pdf")
doc = result.document

# Export to Markdown
markdown = doc.export_to_markdown()

# Export to plain text
text = doc.export_to_markdown(strict_text=True)

# Export to HTML
html = doc.export_to_html()

# Export to JSON (lossless)
json_dict = doc.export_to_dict()
json_str = json.dumps(json_dict)

# Export to DocTags
doctags = doc.export_to_doctags()
```

### Save Exports to Files

```python
import json
from pathlib import Path
from docling.document_converter import DocumentConverter

converter = DocumentConverter()
result = converter.convert("document.pdf")
doc = result.document

output_dir = Path("output")
output_dir.mkdir(exist_ok=True)

filename = result.input.file.stem

# Save all formats
with open(output_dir / f"{filename}.md", "w") as f:
    f.write(doc.export_to_markdown())

with open(output_dir / f"{filename}.txt", "w") as f:
    f.write(doc.export_to_markdown(strict_text=True))

with open(output_dir / f"{filename}.json", "w") as f:
    json.dump(doc.export_to_dict(), f)

with open(output_dir / f"{filename}.doctags", "w") as f:
    f.write(doc.export_to_doctags())
```

---

## CLI Usage

### Basic Commands

```bash
# Convert a document
docling document.pdf

# Convert from URL
docling https://example.com/document.pdf

# Specify output format
docling --output-format markdown document.pdf

# Specify output directory
docling --output-dir ./output document.pdf
```

### Advanced CLI Options

```bash
# Use VLM pipeline with GraniteDocling
docling --pipeline vlm --vlm-model granite_docling document.pdf

# Use local model artifacts (offline mode)
docling --artifacts-path="/local/path/to/models" document.pdf

# Enable OCR
docling --ocr document.pdf

# Get all options
docling --help
```

### Prefetch Models for Offline Use

```bash
# Download all models
docling-tools models download

# Download specific HuggingFace model
docling-tools models download-hf-repo ds4sd/SmolDocling-256M-preview
```

---

## Integrations

Docling integrates with popular AI/ML frameworks:

### LangChain

```python
from langchain_community.document_loaders import DoclingLoader

loader = DoclingLoader(file_path="document.pdf")
docs = loader.load()
```

### LlamaIndex

```python
from llama_index.readers.docling import DoclingReader

reader = DoclingReader()
documents = reader.load_data(file_path="document.pdf")
```

### Other Integrations

- **Haystack**: Document processing pipelines
- **Crew AI**: Multi-agent systems
- **spaCy**: NLP processing
- **NVIDIA**: GPU acceleration

---

## Environment Variables

```bash
# Set model artifacts path
export DOCLING_ARTIFACTS_PATH="/local/path/to/models"

# Limit CPU threads
export OMP_NUM_THREADS=4
```

---

## Key Features Summary

- **Multi-format support**: PDF, DOCX, PPTX, XLSX, HTML, images, and more
- **Advanced PDF understanding**: Layout analysis, reading order, table structure, formulas
- **OCR support**: EasyOCR, Tesseract, macOS native OCR
- **Table extraction**: Export to pandas DataFrame, CSV, HTML
- **Local execution**: No data sent to remote services by default
- **Unified document model**: DoclingDocument representation
- **Multiple export formats**: Markdown, HTML, JSON, plain text, DocTags

---

## Resources

- **Documentation**: https://docling-project.github.io/docling/
- **GitHub**: https://github.com/docling-project/docling
- **Examples**: https://docling-project.github.io/docling/examples/
- **API Reference**: https://docling-project.github.io/docling/reference/document_converter/
- **Discord**: https://docling.ai/discord
