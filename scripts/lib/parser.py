"""
Document parser module.
Converts PDF/DOCX files to Docling DoclingDocument using local Docling library.
"""

import sys
from pathlib import Path

try:
    from docling.document_converter import DocumentConverter
    from docling_core.types.doc.document import DoclingDocument
except ImportError:
    print(
        "Error: docling packages not installed. Run: pip install docling docling-core",
        file=sys.stderr,
    )
    sys.exit(1)


def parse_document(file_path: str, debug: bool = False) -> DoclingDocument:
    """
    Parse a document file into a DoclingDocument.

    Args:
        file_path: Path to the document file (PDF, DOCX, etc.)
        debug: If True, print progress to stderr

    Returns:
        DoclingDocument instance

    Raises:
        FileNotFoundError: If the file doesn't exist
        RuntimeError: If parsing fails
    """
    path = Path(file_path)
    if not path.exists():
        raise FileNotFoundError(f"File not found: {file_path}")

    if debug:
        print(f"[DEBUG] Parsing document: {path.name}", file=sys.stderr)

    converter = DocumentConverter()
    result = converter.convert(str(path))
    doc = result.document

    if debug:
        text_count = len(list(doc.texts)) if hasattr(doc, "texts") else 0
        print(f"[DEBUG] Parsed {text_count} text items", file=sys.stderr)

    return doc
