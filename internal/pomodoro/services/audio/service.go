package audio

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ai-tutor/internal/pomodoro/domain"
	"ai-tutor/internal/pomodoro/infra/events"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
)

// Service handles MP3 playback (single file loop or shuffle folder).
// Silent-fails on missing / invalid files — never crashes the app.
type Service struct {
	mu        sync.Mutex
	emitter   events.Emitter
	stopCh    chan struct{}
	vol       float64         // 0.0 – 1.0
	volCtrl   *effects.Volume // currently-active volume control (nil when idle)
	state     domain.AudioStatePayload
	chimeData []byte // embedded notification sound
}

// New creates a Service. Call SetEmitter after the Wails context is available.
func New() *Service {
	return &Service{
		vol:     0.7,
		emitter: events.Noop{},
		state:   domain.AudioStatePayload{State: domain.AudioIdle},
	}
}

// SetChimeData sets the embedded chime audio data.
func (s *Service) SetChimeData(data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chimeData = data
}

// SetEmitter replaces the emitter (called from App.startup with the live Wails emitter).
func (s *Service) SetEmitter(e events.Emitter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emitter = e
}

// PlayLooping streams a single MP3 file in an infinite loop.
func (s *Service) PlayLooping(filePath string) {
	s.Stop()
	stopCh := make(chan struct{})
	s.mu.Lock()
	s.stopCh = stopCh
	s.mu.Unlock()

	trackName := filepath.Base(filePath)
	s.emitState(domain.AudioPlaying, trackName, "Looping")

	go func() {
		for {
			select {
			case <-stopCh:
				return
			default:
				if err := s.playFile(filePath, stopCh); err != nil {
					s.emitState(domain.AudioStopped, trackName, "Error: "+err.Error())
					return
				}
			}
		}
	}()
}

// PlayShuffleFolder scans a folder for MP3s, shuffles, and plays sequentially.
func (s *Service) PlayShuffleFolder(folder string) {
	s.Stop()
	tracks, err := scanMP3s(folder)
	if err != nil || len(tracks) == 0 {
		s.emitState(domain.AudioStopped, "", "No MP3s found")
		return
	}

	stopCh := make(chan struct{})
	s.mu.Lock()
	s.stopCh = stopCh
	s.mu.Unlock()

	info := fmt.Sprintf("Shuffle folder · %d tracks", len(tracks))
	shuffleStrings(tracks)
	s.emitState(domain.AudioPlaying, filepath.Base(tracks[0]), info)

	go func() {
		idx := 0
		for {
			select {
			case <-stopCh:
				return
			default:
				track := tracks[idx%len(tracks)]
				s.emitState(domain.AudioPlaying, filepath.Base(track), info)
				_ = s.playFile(track, stopCh)
				idx++
				if idx%len(tracks) == 0 {
					shuffleStrings(tracks)
				}
			}
		}
	}()
}

// Stop halts all playback immediately.
func (s *Service) Stop() {
	s.mu.Lock()
	ch := s.stopCh
	s.stopCh = nil
	s.volCtrl = nil
	s.mu.Unlock()

	// Close the stop channel outside the lock so goroutines can exit.
	if ch != nil {
		close(ch)
	}
	// Clear any buffered audio in the speaker.
	speaker.Clear()
	s.emitState(domain.AudioStopped, "", "")
}

// SetVolume adjusts playback volume (0–100 from the frontend).
// Updates take effect immediately on the currently-playing stream.
func (s *Service) SetVolume(v int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v < 0 {
		v = 0
	}
	if v > 100 {
		v = 100
	}
	s.vol = float64(v) / 100.0
	if s.volCtrl != nil {
		speaker.Lock()
		s.volCtrl.Volume = linearToLog(s.vol)
		s.volCtrl.Silent = s.vol == 0
		speaker.Unlock()
	}
}

// GetState returns the current audio state for the frontend.
func (s *Service) GetState() domain.AudioStatePayload {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// PlayChime plays the embedded notification chime sound (non-blocking).
// Does not stop background music; plays independently.
func (s *Service) PlayChime() {
	go s.playChimeAsync()
}

func (s *Service) playChimeAsync() {
	s.mu.Lock()
	chimeData := s.chimeData
	s.mu.Unlock()

	if len(chimeData) == 0 {
		return // chime not embedded
	}

	reader := bytes.NewReader(chimeData)
	streamer, format, err := mp3.Decode(io.NopCloser(reader))
	if err != nil {
		return // silent fail
	}
	defer streamer.Close()

	_ = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	// Play at current volume
	s.mu.Lock()
	vol := &effects.Volume{
		Streamer: streamer,
		Base:     2,
		Volume:   linearToLog(s.vol),
		Silent:   s.vol == 0,
	}
	s.mu.Unlock()

	done := make(chan struct{})
	speaker.Play(beep.Seq(vol, beep.Callback(func() {
		close(done)
	})))

	<-done
}

// ── internal ─────────────────────────────────────────────────────────────────

func (s *Service) playFile(path string, stopCh chan struct{}) error {
	// Bail out early if stop was already requested.
	select {
	case <-stopCh:
		return nil
	default:
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return err
	}
	defer streamer.Close()

	_ = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))

	// Wrap streamer with volume control so the slider has audible effect.
	s.mu.Lock()
	vol := &effects.Volume{
		Streamer: streamer,
		Base:     2,
		Volume:   linearToLog(s.vol),
		Silent:   s.vol == 0,
	}
	s.volCtrl = vol
	s.mu.Unlock()

	// Check if stop was requested before starting playback
	select {
	case <-stopCh:
		return nil
	default:
	}

	done := make(chan struct{})
	speaker.Play(beep.Seq(vol, beep.Callback(func() {
		close(done)
	})))

	select {
	case <-done:
		return nil
	case <-stopCh:
		speaker.Clear()
		return nil
	}
}

// linearToLog converts a linear volume (0.0–1.0) to a logarithmic gain
// suitable for effects.Volume (Base 2). 1.0 → 0 dB, 0.5 → −1, 0 → silent.
func linearToLog(v float64) float64 {
	if v <= 0 {
		return -6 // effectively inaudible
	}
	return math.Log2(v)
}

func (s *Service) emitState(state domain.AudioPlaybackState, track, info string) {
	s.mu.Lock()
	s.state = domain.AudioStatePayload{State: state, TrackName: track, TrackInfo: info}
	payload := s.state
	emitter := s.emitter
	s.mu.Unlock()
	emitter.Emit("audioStateChanged", payload)
}

func scanMP3s(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) == ".mp3" {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

func shuffleStrings(sl []string) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(sl), func(i, j int) { sl[i], sl[j] = sl[j], sl[i] })
}
