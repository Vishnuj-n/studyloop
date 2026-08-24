/**
 * Dynamic registry of notebook source importers provided by extensions.
 * To add a new importer extension (e.g. ArXiv, Notion, Audio Transcriber),
 * simply add an entry here and register its modal/handler component.
 */
export const NOTEBOOK_IMPORTERS = [
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
