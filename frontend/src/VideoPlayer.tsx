import React, { useRef, useEffect, useCallback, useState } from 'react'
import Hls from 'hls.js'
import { motion } from 'framer-motion'

interface VideoPlayerProps {
  /** Full URL — can be blob: for direct playback or http: .m3u8 for HLS */
  playlistUrl: string
  /** true = this user controls playback and broadcasts sync events */
  isHost: boolean
  /** WebSocket ref for sending/receiving sync messages */
  ws: React.RefObject<WebSocket | null>
  /** Called when the host stops the stream */
  onStreamEnd?: () => void
}

const VideoPlayer: React.FC<VideoPlayerProps> = ({
  playlistUrl,
  isHost,
  ws,
  onStreamEnd,
}) => {
  const videoRef = useRef<HTMLVideoElement>(null)
  const hlsRef = useRef<Hls | null>(null)
  const ignoreNextSync = useRef(false)
  const [loading, setLoading] = useState(true)

  // ── Attach source to <video> ──────────────────────────
  useEffect(() => {
    const video = videoRef.current
    if (!video || !playlistUrl) return

    setLoading(true)

    // ── Direct playback (blob: URL or raw file URL) ─────
    // The host uses this path for instant local playback.
    const isHLS = playlistUrl.endsWith('.m3u8')

    if (!isHLS) {
      video.src = playlistUrl
      video.onloadeddata = () => {
        setLoading(false)
        video.play().catch(() => {})
      }
      return () => {
        video.onloadeddata = null
        video.removeAttribute('src')
        video.load()
      }
    }

    // ── HLS.js playback (guests and fallback) ───────────
    if (Hls.isSupported()) {
      const hls = new Hls({
        enableWorker: true,
        // ── Aggressive manifest retry ──
        // The playlist may not exist yet (ffmpeg is still starting).
        // Retry manifest load every 500 ms, up to 30 times (= 15 s window).
        manifestLoadingTimeOut: 5000,
        manifestLoadingMaxRetry: 30,
        manifestLoadingRetryDelay: 500,
        manifestLoadingMaxRetryTimeout: 5000,
        // ── Level / fragment retry ──
        levelLoadingTimeOut: 5000,
        levelLoadingMaxRetry: 10,
        levelLoadingRetryDelay: 500,
        fragLoadingTimeOut: 10000,
        fragLoadingMaxRetry: 6,
        fragLoadingRetryDelay: 500,
        // ── Fast-start tuning ──
        startLevel: -1,
        maxBufferLength: 10,
        maxMaxBufferLength: 30,
        maxBufferSize: 30 * 1000000,
        maxBufferHole: 0.5,
        lowLatencyMode: false,
        startFragPrefetch: true,
      })

      hls.loadSource(playlistUrl)
      hls.attachMedia(video)

      hls.on(Hls.Events.MANIFEST_PARSED, () => {
        video.play().catch(() => {})
      })

      // Hide spinner once the first frame is decoded
      hls.on(Hls.Events.FRAG_BUFFERED, () => {
        setLoading(false)
      })

      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) {
          console.warn('[HLS] Fatal error:', data.type, data.details)
          if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
            console.log('[HLS] Attempting recovery...')
            hls.startLoad()
          } else if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
            hls.recoverMediaError()
          } else {
            hls.destroy()
          }
        }
      })

      hlsRef.current = hls
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      // Safari native HLS
      video.src = playlistUrl
      video.addEventListener('loadedmetadata', () => {
        setLoading(false)
        video.play().catch(() => {})
      })
    }

    return () => {
      hlsRef.current?.destroy()
      hlsRef.current = null
    }
  }, [playlistUrl])

  // ── Host: broadcast play/pause/seek to peers ─────────
  const broadcastSync = useCallback(
    (action: 'play' | 'pause' | 'seek', currentTime: number) => {
      if (!isHost || !ws.current || ws.current.readyState !== WebSocket.OPEN) return
      ws.current.send(
        JSON.stringify({
          type: 'sync-playback',
          action,
          currentTime,
        })
      )
    },
    [isHost, ws]
  )

  // Attach host control listeners
  useEffect(() => {
    const video = videoRef.current
    if (!video || !isHost) return

    const onPlay = () => broadcastSync('play', video.currentTime)
    const onPause = () => broadcastSync('pause', video.currentTime)
    const onSeeked = () => broadcastSync('seek', video.currentTime)

    video.addEventListener('play', onPlay)
    video.addEventListener('pause', onPause)
    video.addEventListener('seeked', onSeeked)

    return () => {
      video.removeEventListener('play', onPlay)
      video.removeEventListener('pause', onPause)
      video.removeEventListener('seeked', onSeeked)
    }
  }, [isHost, broadcastSync])

  // ── Guest: listen for sync-playback + stream-stopped WS messages ──
  useEffect(() => {
    if (isHost) return

    const handler = (event: MessageEvent) => {
      try {
        const data = JSON.parse(event.data)
        const video = videoRef.current
        if (!video) return

        if (data.type === 'sync-playback') {
          ignoreNextSync.current = true
          const timeDiff = Math.abs(video.currentTime - data.currentTime)
          if (timeDiff > 1) {
            video.currentTime = data.currentTime
          }
          if (data.action === 'play') video.play().catch(() => {})
          if (data.action === 'pause') video.pause()
        }

        if (data.type === 'stream-stopped') {
          onStreamEnd?.()
        }
      } catch {
        // ignore non-JSON
      }
    }

    const socket = ws.current
    socket?.addEventListener('message', handler)
    return () => {
      socket?.removeEventListener('message', handler)
    }
  }, [isHost, ws, onStreamEnd])

  return (
    <motion.div
      className="video-player-wrapper"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
      style={{
        position: 'relative',
        width: '100%',
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#000',
      }}
    >
      <video
        ref={videoRef}
        controls
        playsInline
        style={{
          width: '100%',
          height: '100%',
          objectFit: 'contain',
          display: 'block',
        }}
      />
      {loading && (
        <div
          style={{
            position: 'absolute',
            inset: 0,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            justifyContent: 'center',
            background: 'rgba(0,0,0,0.85)',
            gap: '12px',
            zIndex: 10,
          }}
        >
          <div
            style={{
              width: 40,
              height: 40,
              border: '3px solid rgba(0,255,255,0.2)',
              borderTop: '3px solid #0ff',
              borderRadius: '50%',
              animation: 'spin 0.8s linear infinite',
            }}
          />
          <span style={{ color: '#0ff', fontFamily: 'monospace', fontSize: '0.85rem' }}>
            Preparing stream…
          </span>
          <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
        </div>
      )}
      {!isHost && !loading && (
        <div
          style={{
            position: 'absolute',
            top: 8,
            right: 12,
            background: 'rgba(0,0,0,0.6)',
            color: '#0ff',
            padding: '4px 10px',
            borderRadius: '6px',
            fontSize: '0.75rem',
            fontFamily: 'monospace',
          }}
        >
          ▶ SYNCED TO HOST
        </div>
      )}
    </motion.div>
  )
}

export default VideoPlayer
