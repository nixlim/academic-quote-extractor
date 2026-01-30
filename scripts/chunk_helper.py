#!/usr/bin/env python3
"""
Chunking helper script for Academic Quote Extractor.
Reads a Docling document from stdin and outputs chunks as JSON lines.

Usage:
    echo '{"name": "doc.pdf", ...}' | python3 chunk_helper.py

Output format (JSON lines):
    {"id": "#/chunks/0", "text": "...", "page_num": 1, "section_path": ["Ch1", "Sec1"]}
"""

import json
import sys
from typing import Iterator

try:
    from docling_core.types.doc.document import DoclingDocument
    from docling_core.transforms.chunker.hierarchical_chunker import HierarchicalChunker
except ImportError:
    print("Error: docling packages not installed. Run: pip install docling docling-core", file=sys.stderr)
    sys.exit(1)


def create_chunker() -> HierarchicalChunker:
    """Create a HierarchicalChunker with configured parameters."""
    return HierarchicalChunker()


def extract_page_number(chunk) -> int | None:
    """Extract page number from chunk metadata."""
    if hasattr(chunk, 'meta') and chunk.meta:
        if hasattr(chunk.meta, 'doc_items') and chunk.meta.doc_items:
            for item in chunk.meta.doc_items:
                if hasattr(item, 'prov') and item.prov:
                    for prov in item.prov:
                        if hasattr(prov, 'page_no'):
                            return prov.page_no
    return None


def extract_section_path(chunk) -> list[str]:
    """Extract section headings from chunk metadata."""
    section_path = []
    if hasattr(chunk, 'meta') and chunk.meta:
        if hasattr(chunk.meta, 'headings') and chunk.meta.headings:
            section_path = list(chunk.meta.headings)
    return section_path


def extract_bbox(chunk) -> dict | None:
    """Extract bounding box from chunk metadata."""
    if hasattr(chunk, 'meta') and chunk.meta:
        if hasattr(chunk.meta, 'doc_items') and chunk.meta.doc_items:
            for item in chunk.meta.doc_items:
                if hasattr(item, 'prov') and item.prov:
                    for prov in item.prov:
                        if hasattr(prov, 'bbox') and prov.bbox:
                            return {
                                "l": prov.bbox.l,
                                "t": prov.bbox.t,
                                "r": prov.bbox.r,
                                "b": prov.bbox.b,
                                "coord_origin": getattr(prov.bbox, 'coord_origin', 'BOTTOMLEFT')
                            }
    return None


def chunk_document(doc_json: dict) -> Iterator[dict]:
    """Process a Docling document and yield chunks."""
    # Parse the document
    doc = DoclingDocument.model_validate(doc_json)
    
    # Create chunker
    chunker = create_chunker()
    
    # Generate chunks
    for i, chunk in enumerate(chunker.chunk(doc)):
        chunk_id = f"#/chunks/{i}"
        
        # Extract text
        text = chunk.text if hasattr(chunk, 'text') else str(chunk)
        
        # Extract metadata
        page_num = extract_page_number(chunk)
        section_path = extract_section_path(chunk)
        bbox = extract_bbox(chunk)
        
        output = {
            "id": chunk_id,
            "text": text,
            "page_num": page_num,
            "section_path": section_path,
        }
        
        if bbox:
            output["bbox"] = bbox
        
        yield output


def main():
    """Main entry point."""
    try:
        # Read JSON from stdin
        input_data = sys.stdin.read()
        if not input_data.strip():
            print("Error: No input provided on stdin", file=sys.stderr)
            sys.exit(1)
        
        doc_json = json.loads(input_data)
        
        # Process and output chunks
        for chunk in chunk_document(doc_json):
            print(json.dumps(chunk))
    
    except json.JSONDecodeError as e:
        print(f"Error: Invalid JSON input: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
