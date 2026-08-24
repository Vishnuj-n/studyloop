/**
 * Dynamic registry of notebook source importers provided by extensions.
 * To add a new importer extension (e.g. ArXiv, Notion, Audio Transcriber),
 * simply add an entry here and register its modal/handler component.
 */
export const NOTEBOOK_IMPORTERS = [
  {
    id: 'fast_pdf',
    name: 'Fast Structured PDF',
    icon: '⚡',
    badge: 'Pro',
    description: 'High-speed Markdown extraction with tables, headers, and code blocks.',
    fileType: 'pdf',
  },
  {
    id: 'docling_pdf',
    name: 'Docling Deep AI PDF',
    icon: '🔬',
    badge: 'Pro',
    description: 'Deep neural ingestion with OCR, LaTeX formulas, and multi-column tables.',
    fileType: 'pdf',
  },
  {
    id: 'youtube_transcripts',
    name: 'YouTube Lecture',
    icon: '🎥',
    badge: 'Extension',
    description: 'Fetch video transcript & auto-generate structured chapter notes from a YouTube URL.',
    modalName: 'youtube',
  },
]

export function getAvailableImporters(isExtensionActive) {
  return NOTEBOOK_IMPORTERS.filter((importer) => isExtensionActive(importer.id))
}
