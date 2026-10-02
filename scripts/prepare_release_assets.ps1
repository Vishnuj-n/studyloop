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

$expectedHashes = @{
    "model_int8.onnx" = "B4342336DEBAEA79DE872370664B0AAEB67DEA4605513D00EE236EA871A81F27"
    "tokenizer.json"  = "FFB28886478B9B17A8C06F4FE6741B970D5DD3DE13330CCC3B9F686DF8A0545A"
    "onnxruntime.dll" = "8A1AAD8D59D02A5337D4E3F5BBD1158C3F7BF84FE3B3F0052F957DD3E75A91CB"
    "vec0.dll"        = "FCF98662A7AD9DCE394B96A88F91032047823831B951C76636787C312A6476E6"
}

# Component metadata map
$components = [ordered]@{}
foreach ($file in $files) {
    $itemPath = Join-Path $assetDir $file
    $size = (Get-Item $itemPath).Length
    $hash = $hashes[$file]
    $isVerified = ($hash -eq $expectedHashes[$file])

    if ($file -eq "model_int8.onnx") {
        if ($isVerified) {
            $components[$file] = [ordered]@{
                model        = "nomic-ai/nomic-embed-text-v1.5"
                quantization = "INT8 (onnxruntime.quant)"
                dimension    = 768
                source       = "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5"
                sha256       = $hash
                sizeBytes    = $size
            }
        } else {
            $components[$file] = [ordered]@{
                model        = "custom / unverified model"
                sha256       = $hash
                sizeBytes    = $size
            }
        }
    } elseif ($file -eq "tokenizer.json") {
        if ($isVerified) {
            $components[$file] = [ordered]@{
                type      = "Hugging Face Tokenizer (Fast/WordPiece)"
                source    = "https://huggingface.co/nomic-ai/nomic-embed-text-v1.5/blob/main/tokenizer.json"
                sha256    = $hash
                sizeBytes = $size
            }
        } else {
            $components[$file] = [ordered]@{
                type      = "custom / unverified tokenizer"
                sha256    = $hash
                sizeBytes = $size
            }
        }
    } elseif ($file -eq "onnxruntime.dll") {
        if ($isVerified) {
            $components[$file] = [ordered]@{
                product   = "Microsoft ONNX Runtime (Win x64)"
                version   = "1.24.20260203.3.470ae16"
                source    = "https://github.com/microsoft/onnxruntime/releases"
                sha256    = $hash
                sizeBytes = $size
            }
        } else {
            $components[$file] = [ordered]@{
                product   = "Custom / modified ONNX Runtime"
                version   = "custom"
                source    = "local"
                sha256    = $hash
                sizeBytes = $size
            }
        }
    } elseif ($file -eq "vec0.dll") {
        if ($isVerified) {
            $components[$file] = [ordered]@{
                product   = "sqlite-vec extension (Win x64)"
                version   = "v0.1.9"
                source    = "https://github.com/asg017/sqlite-vec/releases/tag/v0.1.9"
                sha256    = $hash
                sizeBytes = $size
            }
        } else {
            $components[$file] = [ordered]@{
                product   = "Custom / modified sqlite-vec extension"
                version   = "custom"
                source    = "local"
                sha256    = $hash
                sizeBytes = $size
            }
        }
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
