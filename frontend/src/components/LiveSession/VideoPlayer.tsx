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
  const retryTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadingStatus, setLoadingStatus] = useState('Preparing stream…')

  // ── Attach source to <video> ──────────────────────────
  useEffect(() => {
    const video = videoRef.current
    if (!video || !playlistUrl) return

    setLoading(true)
    setLoadingStatus('Preparing stream…')

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
    let destroyed = false
    let retryCount = 0
    const MAX_RETRIES = 5

    const createHls = () => {
      if (destroyed) return

      // Cleanup previous instance
      if (hlsRef.current) {
        hlsRef.current.destroy()
        hlsRef.current = null
      }

      setLoadingStatus(retryCount > 0 ? `Reconnecting… (attempt ${retryCount})` : 'Connecting to stream…')

      const hls = new Hls({
        enableWorker: true,
        // ── Aggressive manifest retry ──
        manifestLoadingTimeOut: 8000,
        manifestLoadingMaxRetry: 60,
        manifestLoadingRetryDelay: 1000,
        manifestLoadingMaxRetryTimeout: 8000,
        // ── Level / fragment retry ──
        levelLoadingTimeOut: 8000,
        levelLoadingMaxRetry: 15,
        levelLoadingRetryDelay: 1000,
        fragLoadingTimeOut: 15000,
        fragLoadingMaxRetry: 10,
        fragLoadingRetryDelay: 1000,
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
        setLoadingStatus('Buffering…')
        video.play().catch(() => {})
      })

      // Hide spinner once the first frame is decoded
      hls.on(Hls.Events.FRAG_BUFFERED, () => {
        setLoading(false)
        retryCount = 0 // reset on success
      })

      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (data.fatal) {
          console.warn('[HLS] Fatal error:', data.type, data.details)
          if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
            // Try to recover within the same instance first
            console.log('[HLS] Attempting in-place recovery…')
            hls.startLoad()

            // If we're still in fatal state after 5s, destroy and recreate
            if (retryCount < MAX_RETRIES) {
              retryTimerRef.current = setTimeout(() => {
                if (destroyed) return
                retryCount++
                console.log(`[HLS] Recreating HLS instance (retry ${retryCount}/${MAX_RETRIES})`)
                createHls()
              }, 3000)
            }
          } else if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
            hls.recoverMediaError()
          } else {
            // Non-recoverable error — retry the whole instance
            if (retryCount < MAX_RETRIES) {
              retryCount++
              retryTimerRef.current = setTimeout(() => {
                if (!destroyed) createHls()
              }, 2000)
            }
          }
        }
      })

      hlsRef.current = hls
    }

    if (Hls.isSupported()) {
      createHls()
    } else if (video.canPlayType('application/vnd.apple.mpegurl')) {
      // Safari native HLS
      video.src = playlistUrl
      video.addEventListener('loadedmetadata', () => {
        setLoading(false)
        video.play().catch(() => {})
      })
    }

    return () => {
      destroyed = true
      if (retryTimerRef.current) clearTimeout(retryTimerRef.current)
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

    // Periodic heartbeat sync so late-joiners and drifted guests stay aligned
    const heartbeat = setInterval(() => {
      if (!video.paused && ws.current?.readyState === WebSocket.OPEN) {
        ws.current.send(JSON.stringify({
          type: 'sync-playback',
          action: 'play',
          currentTime: video.currentTime,
        }))
      }
    }, 5000)

    return () => {
      video.removeEventListener('play', onPlay)
      video.removeEventListener('pause', onPause)
      video.removeEventListener('seeked', onSeeked)
      clearInterval(heartbeat)
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
          const timeDiff = Math.abs(video.currentTime - data.currentTime)
          if (timeDiff > 1.5) {
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
        controls={isHost}
        playsInline
        disablePictureInPicture={!isHost}
        disableRemotePlayback={!isHost}
        tabIndex={-1}
        style={{
          width: '100%',
          height: '100%',
          objectFit: 'contain',
          display: 'block',
          pointerEvents: isHost ? 'auto' : 'none',
        }}
      />
      {!isHost && (
        <div style={{
          position: 'absolute',
          inset: 0,
          zIndex: 5,
          cursor: 'default',
          // Completely blocks any interaction targeting the video beneath
        }} />
      )}
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
            {loadingStatus}
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
