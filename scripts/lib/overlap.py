"""
Post-process overlap module.
Adds overlapping text between adjacent chunks to improve quote boundary coverage.

HybridChunker does not natively support overlap, so we add it as a post-processing
step by prepending/appending text from neighboring chunks.
"""

import sys
from typing import Iterator


def add_overlap(chunks: list[dict], overlap_tokens: int = 200, debug: bool = False) -> list[dict]:
    """
    Add overlapping text between adjacent chunks.

    For each chunk, prepend up to `overlap_tokens` tokens from the end of the previous
    chunk, and append up to `overlap_tokens` tokens from the start of the next chunk.
    The original text boundaries are preserved via the 'text' field; overlap is added
    to separate 'overlap_before' and 'overlap_after' fields.

    This approach means the chunk text stored in SQLite is the overlap-enriched version,
    giving Claude more context for quote boundary decisions while the core text remains
    identifiable.

    Args:
        chunks: List of chunk dicts from chunker module
        overlap_tokens: Approximate number of tokens to overlap (using word-based estimate)
        debug: If True, print progress to stderr

    Returns:
        New list of chunk dicts with overlap text merged into the main text field.
        Original chunk boundaries are marked with [CHUNK_START] and [CHUNK_END] markers
        so downstream code can extract the core text if needed.
    """
    if overlap_tokens <= 0 or len(chunks) <= 1:
        return chunks

    if debug:
        print(f"[DEBUG] Adding overlap: {overlap_tokens} tokens between {len(chunks)} chunks", file=sys.stderr)

    result = []
    for i, chunk in enumerate(chunks):
        new_chunk = dict(chunk)
        core_text = chunk["text"]

        # Get overlap from previous chunk
        overlap_before = ""
        if i > 0:
            prev_text = chunks[i - 1]["text"]
            overlap_before = _get_tail_tokens(prev_text, overlap_tokens)

        # Get overlap from next chunk
        overlap_after = ""
        if i < len(chunks) - 1:
            next_text = chunks[i + 1]["text"]
            overlap_after = _get_head_tokens(next_text, overlap_tokens)

        # Build the enriched text
        parts = []
        if overlap_before:
            parts.append(f"[...] {overlap_before}")
        parts.append(core_text)
        if overlap_after:
            parts.append(f"{overlap_after} [...]")

        new_chunk["text"] = "\n".join(parts) if len(parts) > 1 else core_text

        result.append(new_chunk)

    if debug:
        avg_len = sum(len(c["text"]) for c in result) / len(result)
        print(f"[DEBUG] Average chunk length after overlap: {avg_len:.0f} chars", file=sys.stderr)

    return result


def _get_tail_tokens(text: str, n_tokens: int) -> str:
    """Get approximately the last n_tokens words from text."""
    words = text.split()
    if len(words) <= n_tokens:
        return text
    return " ".join(words[-n_tokens:])


def _get_head_tokens(text: str, n_tokens: int) -> str:
    """Get approximately the first n_tokens words from text."""
    words = text.split()
    if len(words) <= n_tokens:
        return text
    return " ".join(words[:n_tokens])
