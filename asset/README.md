# AI Tutor RAG Runtime Assets

This directory contains the binary and model assets required for local vector embeddings and semantic search.

---

## 1. Asset Manifest & Provenance

| Asset File | Product / Model | Version / Type | Upstream Source | SHA-256 Checksum |
| :--- | :--- | :--- | :--- | :--- |
| **`model_int8.onnx`** | Nomic Embed Text v1.5 | INT8 Quantized ONNX (768-d) | [nomic-ai/nomic-embed-text-v1.5](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5) | `B4342336DEBAEA79DE872370664B0AAEB67DEA4605513D00EE236EA871A81F27` |
| **`tokenizer.json`** | Hugging Face Fast Tokenizer | WordPiece / BERT-style | [nomic-ai/nomic-embed-text-v1.5/tokenizer.json](https://huggingface.co/nomic-ai/nomic-embed-text-v1.5/blob/main/tokenizer.json) | `FFB28886478B9B17A8C06F4FE6741B970D5DD3DE13330CCC3B9F686DF8A0545A` |
| **`onnxruntime.dll`** | Microsoft ONNX Runtime | `1.24.20260203.3.470ae16` (Win x64) | [microsoft/onnxruntime Releases](https://github.com/microsoft/onnxruntime/releases) | `8A1AAD8D59D02A5337D4E3F5BBD1158C3F7BF84FE3B3F0052F957DD3E75A91CB` |
| **`vec0.dll`** | `sqlite-vec` SQLite extension | `v0.1.9` (Win x64) | [asg017/sqlite-vec Releases](https://github.com/asg017/sqlite-vec/releases/tag/v0.1.9) | `FCF98662A7AD9DCE394B96A88F91032047823831B951C76636787C312A6476E6` |

---

## 2. Usage & Pipeline Details

- **Embedding Generation**: Managed via `internal/embeddings/onnx.go` using `github.com/yalue/onnxruntime_go` and `github.com/sugarme/tokenizer`.
- **Query Prefixing**:
  - Search queries must be prefixed with: `search_query: `
  - Document chunks must be prefixed with: `search_document: `
- **Vector Storage & Retrieval**: Managed via SQLite using the `vec0` virtual table from `vec0.dll` loaded at connection start (`internal/db/store.go`).

---

## 3. Packaging & Distribution

Release archives (`rag-assets.zip`) are automatically packaged and attached to GitHub releases.

To bundle these assets along with `manifest.json`:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\prepare_release_assets.ps1
```

To sync/download dependencies automatically during local developer setup:
```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\windows-sync-deps.ps1
```
