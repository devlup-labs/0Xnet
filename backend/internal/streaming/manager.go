package streaming

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// StreamManager manages active ffmpeg HLS streams per session.
type StreamManager struct {
	mu      sync.RWMutex
	streams map[string]*streamInfo // sessionID → info
}

type streamInfo struct {
	cmd       *exec.Cmd
	outputDir string
	filePath  string
}

// NewStreamManager creates a new StreamManager.
func NewStreamManager() *StreamManager {
	return &StreamManager{
		streams: make(map[string]*streamInfo),
	}
}

// Start begins HLS transcoding of the given file for a session.
// Returns the relative URL path for the HLS playlist.
func (sm *StreamManager) Start(sessionID, filePath string) (string, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Already streaming for this session?
	if _, exists := sm.streams[sessionID]; exists {
		return fmt.Sprintf("/stream/%s/index.m3u8", sessionID), nil
	}

	// Validate input file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", filePath)
	}

	// Find ffmpeg
	ffmpegPath, err := findFFmpeg()
	if err != nil {
		return "", err
	}

	// Create temp output directory
	outputDir := filepath.Join(os.TempDir(), "0xnet-hls", sessionID)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output dir: %w", err)
	}

	// Build and start ffmpeg
	args := buildFFmpegArgs(filePath, outputDir)
	cmd := exec.Command(ffmpegPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Printf("🎬 [Stream] Starting ffmpeg for session %s: %s %v", sessionID, ffmpegPath, args)

	if err := cmd.Start(); err != nil {
		os.RemoveAll(outputDir)
		return "", fmt.Errorf("failed to start ffmpeg: %w", err)
	}

	sm.streams[sessionID] = &streamInfo{
		cmd:       cmd,
		outputDir: outputDir,
		filePath:  filePath,
	}

	// Wait for ffmpeg to finish in background (cleanup on completion)
	go func() {
		if err := cmd.Wait(); err != nil {
			log.Printf("⚠️ [Stream] ffmpeg exited for session %s: %v", sessionID, err)
		} else {
			log.Printf("✅ [Stream] ffmpeg completed for session %s", sessionID)
		}
	}()

	playlistURL := fmt.Sprintf("/stream/%s/index.m3u8", sessionID)
	log.Printf("🎬 [Stream] ffmpeg started, playlist will be at %s", playlistURL)
	return playlistURL, nil
}

// WaitForPlaylist blocks until the HLS playlist file exists on disk
// (i.e. ffmpeg has written the first segment). Returns false if the
// timeout is hit or the stream was stopped before the file appeared.
func (sm *StreamManager) WaitForPlaylist(sessionID string, timeout time.Duration) bool {
	sm.mu.RLock()
	info, exists := sm.streams[sessionID]
	sm.mu.RUnlock()
	if !exists {
		return false
	}

	playlistPath := filepath.Join(info.outputDir, "index.m3u8")
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if fi, err := os.Stat(playlistPath); err == nil && fi.Size() > 0 {
			log.Printf("✅ [Stream] Playlist ready for session %s (%d bytes)", sessionID, fi.Size())
			return true
		}
		// Check if stream was stopped while waiting
		sm.mu.RLock()
		_, stillActive := sm.streams[sessionID]
		sm.mu.RUnlock()
		if !stillActive {
			return false
		}
		time.Sleep(250 * time.Millisecond)
	}

	log.Printf("⚠️ [Stream] Playlist wait timed out for session %s", sessionID)
	return false
}

// Stop kills the ffmpeg process for a session and cleans up temp files.
func (sm *StreamManager) Stop(sessionID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	info, exists := sm.streams[sessionID]
	if !exists {
		return
	}

	// Kill the ffmpeg process
	if info.cmd.Process != nil {
		log.Printf("🛑 [Stream] Stopping ffmpeg for session %s", sessionID)
		info.cmd.Process.Kill()
	}

	// Remove temp HLS files
	os.RemoveAll(info.outputDir)
	delete(sm.streams, sessionID)
	log.Printf("🧹 [Stream] Cleaned up session %s", sessionID)
}

// IsStreaming returns true if a stream is active for the session.
func (sm *StreamManager) IsStreaming(sessionID string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	_, exists := sm.streams[sessionID]
	return exists
}

// GetOutputDir returns the HLS output directory for a session.
func (sm *StreamManager) GetOutputDir(sessionID string) string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if info, ok := sm.streams[sessionID]; ok {
		return info.outputDir
	}
	return ""
}

// StopAll kills all active streams (used on shutdown).
func (sm *StreamManager) StopAll() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for sid, info := range sm.streams {
		if info.cmd.Process != nil {
			info.cmd.Process.Kill()
		}
		os.RemoveAll(info.outputDir)
		log.Printf("🧹 [Stream] Shutdown cleanup: session %s", sid)
	}
	sm.streams = make(map[string]*streamInfo)
}
