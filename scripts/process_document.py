#!/usr/bin/env python3
"""
Document processing pipeline for Academic Quote Extractor.

Parses a document file using local Docling, chunks it with HybridChunker,
and applies post-process overlap. Outputs JSON lines to stdout.

Usage:
    python3 scripts/process_document.py --file paper.pdf
    python3 scripts/process_document.py --file paper.pdf --max-tokens 800 --overlap 200
    python3 scripts/process_document.py --file paper.pdf --stage parse   # Parse only (debug)
    python3 scripts/process_document.py --file paper.pdf --stage chunk   # Chunk only (reads DoclingDocument JSON from stdin)

Output format (JSON lines to stdout):
    {"id": "#/chunks/0", "text": "...", "page_num": 1, "section_path": ["Ch1", "Sec1"]}
    {"id": "#/chunks/1", "text": "...", "page_num": 1, "section_path": ["Ch1", "Sec1"]}
    ...

Exit codes:
    0: Success
    1: User error (file not found, invalid args)
    2: System error (dependency missing, parse failure)
"""

import argparse
import json
import sys
import os

# Add the scripts directory to the path so we can import lib modules
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from lib.parser import parse_document
from lib.chunker import chunk_document
from lib.overlap import add_overlap


def main():
    parser = argparse.ArgumentParser(
        description="Process academic documents for quote extraction"
    )
    parser.add_argument(
        "--file", required=True, help="Path to the document file (PDF, DOCX, etc.)"
    )
    parser.add_argument(
        "--max-tokens",
        type=int,
        default=800,
        help="Maximum tokens per chunk (default: 800)",
    )
    parser.add_argument(
        "--overlap",
        type=int,
        default=200,
        help="Overlap tokens between chunks (default: 200, 0 to disable)",
    )
    parser.add_argument(
        "--stage",
        choices=["all", "parse", "chunk"],
        default="all",
        help="Processing stage: all (default), parse (output DoclingDocument JSON), chunk (read DoclingDocument from stdin)",
    )
    parser.add_argument(
        "--debug",
        action="store_true",
        help="Enable debug output to stderr",
    )

    args = parser.parse_args()

    try:
        if args.stage == "parse":
            # Parse only: output DoclingDocument as JSON
            run_parse(args)
        elif args.stage == "chunk":
            # Chunk only: read DoclingDocument JSON from stdin
            run_chunk(args)
        else:
            # Full pipeline: parse + chunk + overlap
            run_all(args)
    except FileNotFoundError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        sys.exit(130)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        if args.debug:
            import traceback
            traceback.print_exc(file=sys.stderr)
        sys.exit(2)


def run_parse(args):
    """Parse document and output DoclingDocument as JSON."""
    doc = parse_document(args.file, debug=args.debug)
    # Export as JSON
    doc_json = doc.export_to_dict()
    json.dump(doc_json, sys.stdout, default=str)
    sys.stdout.write("\n")

    if args.debug:
        print(f"[DEBUG] Exported DoclingDocument JSON", file=sys.stderr)


def run_chunk(args):
    """Read DoclingDocument JSON from stdin and output chunks."""
    from docling_core.types.doc.document import DoclingDocument

    input_data = sys.stdin.read()
    if not input_data.strip():
        print("Error: No input provided on stdin", file=sys.stderr)
        sys.exit(1)

    doc_dict = json.loads(input_data)
    doc = DoclingDocument.model_validate(doc_dict)

    chunks = list(chunk_document(doc, max_tokens=args.max_tokens, debug=args.debug))

    if args.overlap > 0:
        chunks = add_overlap(chunks, overlap_tokens=args.overlap, debug=args.debug)

    for chunk in chunks:
        print(json.dumps(chunk))


def run_all(args):
    """Full pipeline: parse + chunk + overlap."""
    if args.debug:
        print(f"[DEBUG] Processing: {args.file}", file=sys.stderr)
        print(f"[DEBUG] max_tokens={args.max_tokens}, overlap={args.overlap}", file=sys.stderr)

    # Step 1: Parse
    doc = parse_document(args.file, debug=args.debug)

    # Step 2: Chunk
    chunks = list(chunk_document(doc, max_tokens=args.max_tokens, debug=args.debug))

    if args.debug:
        print(f"[DEBUG] Pre-overlap chunks: {len(chunks)}", file=sys.stderr)

    # Step 3: Overlap
    if args.overlap > 0:
        chunks = add_overlap(chunks, overlap_tokens=args.overlap, debug=args.debug)

    # Output as JSON lines
    for chunk in chunks:
        print(json.dumps(chunk))

    if args.debug:
        print(f"[DEBUG] Output {len(chunks)} chunks", file=sys.stderr)


if __name__ == "__main__":
    main()
