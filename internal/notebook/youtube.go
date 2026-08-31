package notebook

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"ai-tutor/internal/extension"
	"ai-tutor/internal/utils"
)

// YouTubeIngestResult holds yt-dlp extracted payload.
type YouTubeIngestResult struct {
	VideoID         string `json:"video_id"`
	Title           string `json:"title"`
	Uploader        string `json:"uploader"`
	DurationSeconds int    `json:"duration_seconds"`
	Chapters        []struct {
		ChapterIndex int    `json:"chapter_index"`
		Title        string `json:"title"`
		StartSeconds int    `json:"start_seconds"`
		EndSeconds   int    `json:"end_seconds"`
		Transcript   string `json:"transcript"`
	} `json:"chapters"`
	Error string `json:"error,omitempty"`
}

// IngestYouTubeVideo runs the youtube extension and feeds directly into standard ExtractedDocument.
func (s *Service) IngestYouTubeVideo(ctx context.Context, videoURL string, runner *extension.Runner, ext *extension.Extension) (*ExtractedDocument, *YouTubeIngestResult, error) {
	cleanURL := strings.TrimSpace(videoURL)
	if cleanURL == "" {
		return nil, nil, fmt.Errorf("video URL or ID cannot be empty")
	}
	if runner == nil {
		runner = extension.NewRunner()
	}

	entrypoint := "ingest.py"
	if ext != nil && ext.Entrypoint() != "" {
		entrypoint = ext.Entrypoint()
	} else {
		extDir := extension.ResolveExtensionsDir("")
		ext = &extension.Extension{
			Manifest: extension.Manifest{ID: "youtube", Name: "YouTube", Runtime: "python", Entrypoint: "ingest.py"},
			Dir:      filepath.Join(extDir, "youtube"),
		}
	}

	pythonPath, err := extension.FindExtensionPython(ext)
	if err != nil {
		return nil, nil, fmt.Errorf("python is required for youtube ingestion: %w", err)
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
	}

	outputBytes, err := runner.Run(ctx, ext, pythonPath, entrypoint, cleanURL)
	if err != nil && len(outputBytes) == 0 {
		return nil, nil, fmt.Errorf("youtube extension failed: %w", err)
	}

	var res YouTubeIngestResult
	if err := json.Unmarshal(outputBytes, &res); err != nil {
		return nil, nil, fmt.Errorf("invalid youtube output: %w", err)
	}
	if res.Error != "" {
		return nil, nil, fmt.Errorf("%s", res.Error)
	}
	if len(res.Chapters) == 0 {
		return nil, nil, fmt.Errorf("no chapters or transcript found for video")
	}

	// ponytail: Map chapters 1:1 into standard sections so downstream Quiz/FSRS/RAG need 0 custom code
	doc := &ExtractedDocument{
		Title:     res.Title,
		PageCount: len(res.Chapters),
		Sections:  make([]ExtractedSection, 0, len(res.Chapters)),
	}

	totalWords := 0
	for i, ch := range res.Chapters {
		title := utils.CleanTopicTitle(ch.Title)
		if title == "" {
			title = fmt.Sprintf("Chapter %d", i+1)
		}
		words := strings.Fields(ch.Transcript)
		totalWords += len(words)
		doc.Sections = append(doc.Sections, ExtractedSection{
			Heading: fmt.Sprintf("%s (%02d:%02d - %02d:%02d)", title, ch.StartSeconds/60, ch.StartSeconds%60, ch.EndSeconds/60, ch.EndSeconds%60),
			Text:    ch.Transcript,
			PageNum: i + 1,
		})
	}
	doc.WordCount = totalWords

	return doc, &res, nil
}

// DownloadYouTubeVideo runs yt-dlp to download a video file asynchronously in the background.
func (s *Service) DownloadYouTubeVideo(ctx context.Context, videoURL string, outputPath string, quality string, runner *extension.Runner, ext *extension.Extension) error {
	cleanURL := strings.TrimSpace(videoURL)
	if cleanURL == "" {
		return fmt.Errorf("video URL cannot be empty")
	}
	if runner == nil {
		runner = extension.NewRunner()
	}

	entrypoint := "ingest.py"
	if ext != nil && ext.Entrypoint() != "" {
		entrypoint = ext.Entrypoint()
	} else {
		extDir := extension.ResolveExtensionsDir("")
		ext = &extension.Extension{
			Manifest: extension.Manifest{ID: "youtube", Name: "YouTube", Runtime: "python", Entrypoint: "ingest.py"},
			Dir:      filepath.Join(extDir, "youtube"),
		}
	}

	pythonPath, err := extension.FindExtensionPython(ext)
	if err != nil {
		return fmt.Errorf("python is required for youtube download: %w", err)
	}

	outputBytes, err := runner.Run(ctx, ext, pythonPath, entrypoint, cleanURL, "--download-video", outputPath, "--quality", quality)
	if err != nil {
		return fmt.Errorf("video download failed: %w (output: %s)", err, string(outputBytes))
	}
	return nil
}

