/**
 * Dynamic registry of notebook source importers provided by extensions.
 * To add a new importer extension (e.g. ArXiv, Notion, Audio Transcriber),
 * simply add an entry here and register its modal/handler component.
 */
export const NOTEBOOK_IMPORTERS = [
  {
    id: 'deep_pdf',
    name: 'Deep Structured PDF',
    icon: '⚡',
    badge: 'Pro',
    description: 'Deep high-speed Markdown extraction with tables, headers, and code blocks.',
    fileType: 'pdf',
  },
  {
    id: 'youtube',
    name: 'YouTube Lecture',
    icon: '🎥',
    description: 'Fetch video transcript & auto-generate structured chapter notes from a YouTube URL.',
    modalName: 'youtube',
  },
]

export function getAvailableImporters(isExtensionActive) {
  return NOTEBOOK_IMPORTERS.filter((importer) => isExtensionActive(importer.id))
}
