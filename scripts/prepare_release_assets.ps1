# prepare_release_assets.ps1
# AI Tutor: Release asset packaging helper script
# Generates manifest.json with SHA-256 checksums and creates build\bin\rag-assets.zip

$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
Set-Location $projectRoot

$appVersion = "v1.0.0"
if (Test-Path ".\internal\app\VERSION") {
    $appVersion = (Get-Content ".\internal\app\VERSION" -Raw).Trim()
} elseif (Test-Path ".\VERSION") {
    $appVersion = (Get-Content ".\VERSION" -Raw).Trim()
}
if (-not $appVersion.StartsWith("v")) {
    $appVersion = "v$appVersion"
}
$normalizedVersion = $appVersion.TrimStart("v")

Write-Host "=========================================="
Write-Host "AI Tutor: Packaging RAG Assets ($appVersion)"
Write-Host "=========================================="

$assetDir = ".\asset"
$outDir = ".\build\bin"

if (!(Test-Path $outDir)) {
    New-Item -ItemType Directory -Force -Path $outDir | Out-Null
}

$files = @("tokenizer.json", "model_int8.onnx", "onnxruntime.dll", "vec0.dll")

# Check that all source files exist
foreach ($file in $files) {
    $path = Join-Path $assetDir $file
    if (!(Test-Path $path)) {
        Write-Host "Error: Required asset file missing: $path" -ForegroundColor Red
        exit 1
    }
}

# Calculate SHA256 hashes
$hashes = @{}
foreach ($file in $files) {
    $path = Join-Path $assetDir $file
    Write-Host "Calculating SHA-256 hash for $file..."
    $hashVal = (Get-FileHash -Path $path -Algorithm SHA256).Hash.ToUpper()
    $hashes[$file] = $hashVal
}

# Component metadata map
$components = [ordered]@{
    "model_int8.onnx" = [ordered]@{
        model        = "nomic-ai/nomic-embed-text-v1.5"
        quantization = "INT8 (onnxruntime.quant)"
        dimension    = 768
        source       = "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5"
        sha256       = $hashes["model_int8.onnx"]
        sizeBytes    = (Get-Item (Join-Path $assetDir "model_int8.onnx")).Length
    }
    "tokenizer.json" = [ordered]@{
        type      = "Hugging Face Tokenizer (Fast/WordPiece)"
        source    = "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5/blob/main/tokenizer.json"
        sha256    = $hashes["tokenizer.json"]
        sizeBytes = (Get-Item (Join-Path $assetDir "tokenizer.json")).Length
    }
    "onnxruntime.dll" = [ordered]@{
        product   = "Microsoft ONNX Runtime (Win x64)"
        version   = "1.24.20260203.3.470ae16"
        source    = "https://github.com/microsoft/onnxruntime/releases"
        sha256    = $hashes["onnxruntime.dll"]
        sizeBytes = (Get-Item (Join-Path $assetDir "onnxruntime.dll")).Length
    }
    "vec0.dll" = [ordered]@{
        product   = "sqlite-vec extension (Win x64)"
        version   = "v0.1.9"
        source    = "https://github.com/asg017/sqlite-vec/releases/tag/v0.1.9"
        sha256    = $hashes["vec0.dll"]
        sizeBytes = (Get-Item (Join-Path $assetDir "vec0.dll")).Length
    }
}

# Generate manifest.json content
$manifest = [ordered]@{
    assetVersion  = $normalizedVersion
    downloadUrl   = "https://github.com/Vishnuj-n/studyloop/releases/download/$appVersion/rag-assets.zip"
    requiredFiles = $files
    components    = $components
    fileHashes    = $hashes
}

# Save manifest.json inside asset directory
$manifestPath = Join-Path $assetDir "manifest.json"
$manifest | ConvertTo-Json -Depth 5 | Set-Content -Path $manifestPath -Force
Write-Host "Generated manifest.json at $manifestPath"

# Zip them up
$zipPath = Join-Path $outDir "rag-assets.zip"
if (Test-Path $zipPath) {
    Remove-Item $zipPath -Force
}

Write-Host "Creating zip archive at $zipPath..."
$compressFiles = $files | ForEach-Object { Join-Path $assetDir $_ }
$compressFiles += $manifestPath

# Create Zip Archive
Compress-Archive -Path $compressFiles -DestinationPath $zipPath -Force

# Retain manifest.json in asset directory for repository source tracking

Write-Host ""
Write-Host "==========================================" -ForegroundColor Green
Write-Host "Success! Generated rag-assets.zip in build\bin\" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Green
Write-Host ""
