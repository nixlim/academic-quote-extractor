"""
Document chunker module.
Splits a DoclingDocument into semantic chunks using HybridChunker.
"""

import sys
from typing import Iterator

try:
    from docling_core.types.doc.document import DoclingDocument
    from docling_core.transforms.chunker.hybrid_chunker import HybridChunker
    from docling_core.transforms.chunker.tokenizer.huggingface import (
        HuggingFaceTokenizer,
    )
    from transformers import AutoTokenizer
except ImportError as e:
    print(
        f"Error: required packages not installed: {e}\n"
        "Run: pip install docling docling-core transformers",
        file=sys.stderr,
    )
    sys.exit(1)


# Default tokenizer model for token counting
_TOKENIZER_MODEL = "sentence-transformers/all-MiniLM-L6-v2"


def create_chunker(max_tokens: int = 800) -> HybridChunker:
    """
    Create a HybridChunker with token-aware splitting.

    Args:
        max_tokens: Maximum tokens per chunk (default 800)

    Returns:
        Configured HybridChunker instance
    """
    tokenizer = HuggingFaceTokenizer(
        tokenizer=AutoTokenizer.from_pretrained(_TOKENIZER_MODEL),
        max_tokens=max_tokens,
    )
    return HybridChunker(
        tokenizer=tokenizer,
        merge_peers=True,
    )


def extract_page_number(chunk) -> int | None:
    """Extract page number from chunk metadata."""
    if hasattr(chunk, "meta") and chunk.meta:
        if hasattr(chunk.meta, "doc_items") and chunk.meta.doc_items:
            for item in chunk.meta.doc_items:
                if hasattr(item, "prov") and item.prov:
                    for prov in item.prov:
                        if hasattr(prov, "page_no"):
                            return prov.page_no
    return None


def extract_section_path(chunk) -> list[str]:
    """Extract section headings from chunk metadata."""
    section_path = []
    if hasattr(chunk, "meta") and chunk.meta:
        if hasattr(chunk.meta, "headings") and chunk.meta.headings:
            section_path = list(chunk.meta.headings)
    return section_path


def extract_bbox(chunk) -> dict | None:
    """Extract bounding box from chunk metadata."""
    if hasattr(chunk, "meta") and chunk.meta:
        if hasattr(chunk.meta, "doc_items") and chunk.meta.doc_items:
            for item in chunk.meta.doc_items:
                if hasattr(item, "prov") and item.prov:
                    for prov in item.prov:
                        if hasattr(prov, "bbox") and prov.bbox:
                            return {
                                "l": prov.bbox.l,
                                "t": prov.bbox.t,
                                "r": prov.bbox.r,
                                "b": prov.bbox.b,
                                "coord_origin": getattr(
                                    prov.bbox, "coord_origin", "BOTTOMLEFT"
                                ),
                            }
    return None


def chunk_document(
    doc: DoclingDocument, max_tokens: int = 800, debug: bool = False
) -> Iterator[dict]:
    """
    Split a DoclingDocument into semantic chunks.

    Args:
        doc: Parsed DoclingDocument
        max_tokens: Maximum tokens per chunk
        debug: If True, print progress to stderr

    Yields:
        dict with keys: id, text, page_num, section_path, bbox (optional)
    """
    chunker = create_chunker(max_tokens=max_tokens)

    if debug:
        print(f"[DEBUG] Chunking with max_tokens={max_tokens}", file=sys.stderr)

    count = 0
    for i, chunk in enumerate(chunker.chunk(doc)):
        chunk_id = f"#/chunks/{i}"
        text = chunk.text if hasattr(chunk, "text") else str(chunk)

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

        count += 1
        yield output

    if debug:
        print(f"[DEBUG] Generated {count} chunks", file=sys.stderr)
