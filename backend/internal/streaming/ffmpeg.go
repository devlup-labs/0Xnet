package streaming

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// findFFmpeg locates the ffmpeg binary on the system.
func findFFmpeg() (string, error) {
	// Try PATH first
	path, err := exec.LookPath("ffmpeg")
	if err == nil {
		return path, nil
	}
	// Common Windows install locations
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		userProfile := os.Getenv("USERPROFILE")

		candidates := []string{
			`C:\ffmpeg\bin\ffmpeg.exe`,
			`C:\Program Files\ffmpeg\bin\ffmpeg.exe`,
		}

		// Add gyan.dev / user-level install patterns
		if localAppData != "" {
			candidates = append(candidates,
				filepath.Join(localAppData, `ffmpeg\ffmpeg-8.1-full_build\bin\ffmpeg.exe`),
			)
			// Glob for any version: %LOCALAPPDATA%\ffmpeg\ffmpeg-*\bin\ffmpeg.exe
			matches, _ := filepath.Glob(filepath.Join(localAppData, `ffmpeg\ffmpeg-*\bin\ffmpeg.exe`))
			candidates = append(candidates, matches...)
		}
		if userProfile != "" {
			candidates = append(candidates,
				filepath.Join(userProfile, `ffmpeg\bin\ffmpeg.exe`),
			)
		}

		for _, c := range candidates {
			if _, statErr := os.Stat(c); statErr == nil {
				return c, nil
			}
		}
	}
	return "", fmt.Errorf("ffmpeg not found in PATH — install it: https://ffmpeg.org/download.html")
}

// buildFFmpegArgs creates the argument list for the ffmpeg HLS command.
// It uses "-c copy" for fast remuxing when possible (MP4/MOV with H.264),
// and transcodes for other formats (MKV, AVI, etc.).
func buildFFmpegArgs(inputPath, outputDir string) []string {
	ext := filepath.Ext(inputPath)
	playlistPath := filepath.Join(outputDir, "index.m3u8")
	segmentPattern := filepath.Join(outputDir, "seg_%03d.ts")

	// Base args: fast input analysis, then overwrite + input
	args := []string{
		"-y",
		"-fflags", "+genpts+discardcorrupt", // don't stall on bad timestamps
		"-analyzeduration", "2000000",        // cap input analysis to 2 seconds (µs)
		"-probesize", "5000000",              // cap probe to 5 MB (enough for headers)
		"-i", inputPath,
	}

	// For MP4/MOV with likely H.264+AAC we can do a fast copy.
	// For everything else, transcode to H.264+AAC.
	switch ext {
	case ".mp4", ".mov", ".m4v":
		args = append(args,
			"-c", "copy", // remux — instant, original quality
		)
	default:
		// MKV, AVI, WebM, FLV, etc. → transcode
		args = append(args,
			"-c:v", "libx264",
			"-preset", "ultrafast",
			"-crf", "23",
			"-g", "48", // Force keyframes every 48 frames (approx 2s at 24fps)
			"-c:a", "aac",
			"-b:a", "128k",
		)
	}

	// HLS output settings — tuned for fastest time-to-first-frame
	args = append(args,
		"-f", "hls",
		"-hls_time", "2",                              // 2-second segments
		"-hls_init_time", "0",                         // emit first segment ASAP (don't wait for full hls_time)
		"-hls_list_size", "0",                         // keep all segments in the playlist
		"-hls_segment_filename", segmentPattern,
		"-hls_flags", "independent_segments+temp_file", // temp_file: prevent HLS.js reading half-written .ts
		playlistPath,
	)

	return args
}
