# Solution: Deep PDF Bookmark/Page Indexing Fix

## Root causes

- Deep PDF Markdown was split into synthetic section numbers instead of real PDF pages, so bookmark ranges such as pages 31–36 produced no chunks.
- Queue tasks can retain a previous topic ID after syllabus confirmation regenerates topic IDs from edited titles.
- Deep extraction output preserved Markdown heading markers (`#`, `##`, `###`) in chunk text.

## Fix

- The deep PDF extension emits explicit `<!-- page: N -->` markers for each PDF page.
- Go ingestion uses those markers as real `ExtractedSection.PageNum` values and removes heading-only lines from stored text.
- `CompleteReading` falls back to notebook-owned chunks within the task page range when the topic ID is stale.

PDF bookmarks remain the syllabus source of truth; deep extraction supplies improved page text only.

## Verification

- Added a regression test for page preservation and heading removal.
- Python extension syntax validation passes.
- Go compile-only verification was run with `go test -run=^$ ./internal/...`.
