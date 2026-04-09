import React, { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import ScrambledText from '../ui/ScrambledText'

import VideoPlayer from './VideoPlayer'
import './LiveSession.css'

interface Participant {
  id: string
  deviceId: string
  name: string
  avatar: string
  status: 'online' | 'busy' | 'away'
  role?: 'host' | 'guest'
  isMe?: boolean
}

interface Message {
  type: 'chat' | 'system'
  sender?: string
  message: string
  timestamp: string
}

interface LiveSessionProps {
  myDeviceId: string
  sessionData: {
    id: string
    name: string
    activeSince: string
    members: Participant[]
    hostIp?: string
    hostPort?: number
  }
  onLeave: () => void
}

const LiveSession: React.FC<LiveSessionProps> = ({ myDeviceId, sessionData, onLeave }) => {
  const [participantsOpen, setParticipantsOpen] = useState(false)
  const [chatOpen, setChatOpen] = useState(false)
  const [isMuted, setIsMuted] = useState(false)
  const [isVideoOn, setIsVideoOn] = useState(true)
  const [messages, setMessages] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState('')
  const [localStreamLoaded, setLocalStreamLoaded] = useState(false)
  const [wsReady, setWsReady] = useState(false)
  const [participants, setParticipants] = useState<Participant[]>(sessionData.members)
  const [sessionDuration, setSessionDuration] = useState('00h 00m 00s')

  useEffect(() => {
    const startTime = Date.now();
    const interval = setInterval(() => {
      const diff = Math.floor((Date.now() - startTime) / 1000);
      const hrs = Math.floor(diff / 3600);
      const mins = Math.floor((diff % 3600) / 60);
      const secs = diff % 60;
      
      const fmt = (n: number) => n.toString().padStart(2, '0');
      setSessionDuration(`${fmt(hrs)}h ${fmt(mins)}m ${fmt(secs)}s`);
    }, 1000);
    return () => clearInterval(interval);
  }, []);

  // ── Host departure state ─────────────────────────────────
  const [hostLeft, setHostLeft] = useState(false)
  const [countdown, setCountdown] = useState(10)

  // ── HLS Streaming State ─────────────────────────────────
  const [hlsPlaylistUrl, setHlsPlaylistUrl] = useState<string | null>(null)
  const [isStreaming, setIsStreaming] = useState(false)
  const [streamLoading, setStreamLoading] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const streamContainerRef = useRef<HTMLDivElement>(null)
  const hostPlayingLocally = useRef(false) // true when host is playing via blob URL
  const isHost = participants.find(p => p.isMe)?.role === 'host'

  const ws = useRef<WebSocket | null>(null)
  const chatEndRef = useRef<HTMLDivElement>(null)

  // WebRTC Refs
  const localStream = useRef<MediaStream | null>(null)
  const peerConnections = useRef<{ [peerId: string]: RTCPeerConnection }>({})
  const [remoteStreams, setRemoteStreams] = useState<{ [peerId: string]: MediaStream }>({})

  useEffect(() => {
    setParticipants(sessionData.members)
  }, [sessionData.members])

  // Sync fullscreen state with browser (e.g. user presses Escape)
  useEffect(() => {
    const handler = () => setIsFullscreen(!!document.fullscreenElement)
    document.addEventListener('fullscreenchange', handler)
    return () => document.removeEventListener('fullscreenchange', handler)
  }, [])

  const shouldInitiateConnection = (localId: string, remoteId: string) => {
    return localId.localeCompare(remoteId) < 0
  }

  const removePeerConnection = (peerId: string) => {
    const existing = peerConnections.current[peerId]
    if (existing) {
      existing.onicecandidate = null
      existing.ontrack = null
      existing.onconnectionstatechange = null
      existing.close()
      delete peerConnections.current[peerId]
    }

    setRemoteStreams(prev => {
      const next = { ...prev }
      delete next[peerId]
      return next
    })
  }

  useEffect(() => {
    const backendPort = '8080'
    const targetHost = sessionData.hostIp || window.location.hostname
    const targetPort = sessionData.hostPort || backendPort

    const fetchMembers = async () => {
      try {
        const resp = await fetch(
          `http://${targetHost}:${targetPort}/session/members?sessionId=${encodeURIComponent(sessionData.id)}`
        )
        if (!resp.ok) return

        const members = await resp.json()
        const mapped: Participant[] = Array.isArray(members)
          ? members.map((m: any) => ({
            id: m.id || m.deviceId || Math.random().toString(),
            deviceId: m.deviceId || 'unknown',
            name: m.deviceName || m.deviceId || 'Unknown',
            avatar: '',
            status: 'online',
            role: m.deviceName === 'Host' ? 'host' : 'guest',
            isMe: m.deviceId === myDeviceId
          }))
          : []

        if (!mapped.some((m) => m.deviceId === myDeviceId)) {
          mapped.unshift({
            id: myDeviceId || 'local',
            deviceId: myDeviceId,
            name: 'You',
            avatar: '',
            status: 'online',
            role: 'guest',
            isMe: true
          })
        }

        setParticipants(mapped)
      } catch (err) {
        console.error('Failed to refresh session members', err)
      }
    }

    fetchMembers()
    const interval = setInterval(fetchMembers, 2000)
    return () => clearInterval(interval)
  }, [sessionData.id, sessionData.hostIp, sessionData.hostPort, myDeviceId])

  useEffect(() => {
    // Initialize local media
    const initMedia = async () => {
      try {
        console.log('Requesting local media...')
        const stream = await navigator.mediaDevices.getUserMedia({
          video: true,
          audio: true
        })
        console.log('Local stream obtained:', stream.id)
        localStream.current = stream
        setLocalStreamLoaded(true)
      } catch (err) {
        console.error('Error accessing media devices:', err)
      }
    }

    if (!localStream.current) {
      initMedia()
    }

    return () => {
      // Don't stop tracks here, let the main session effect handle it
    }
  }, [])

  useEffect(() => {
    const backendPort = '8080'
    const targetHost = sessionData.hostIp || window.location.hostname
    const targetPort = sessionData.hostPort || backendPort

    // 2. Create WebSocket connection
    const socket = new WebSocket(`ws://${targetHost}:${targetPort}/ws`)
    ws.current = socket

    socket.onopen = () => {
      console.log('WS Connected')
      setWsReady(true)
      socket.send(JSON.stringify({
        type: 'join-session',
        sessionId: sessionData.id,
        username: myDeviceId
      }))
    }

    socket.onmessage = async (event) => {
      const data = JSON.parse(event.data)

      switch (data.type) {
        case 'chat':
        case 'system':
          setMessages(prev => [...prev, data])
          break

        case 'offer':
          handleOffer(data)
          break

        case 'answer':
          handleAnswer(data)
          break

        case 'ice-candidate':
          handleICECandidate(data)
          break

        case 'stream-started': {
          // Host is already playing locally via blob URL — don't overwrite with HLS URL
          if (hostPlayingLocally.current) {
            console.log('[HLS] Host already playing locally, ignoring stream-started')
            break
          }
          const base = `http://${sessionData.hostIp || window.location.hostname}:${sessionData.hostPort || '8080'}`
          const fullUrl = data.playlistUrl.startsWith('http') ? data.playlistUrl : `${base}${data.playlistUrl}`
          console.log('[HLS] Stream started:', fullUrl)
          setHlsPlaylistUrl(fullUrl)
          setIsStreaming(true)
          break
        }

        case 'stream-stopped':
          console.log('[HLS] Stream stopped')
          setHlsPlaylistUrl(null)
          setIsStreaming(false)
          break

        case 'host-left':
          console.log('[Session] Host left — countdown started')
          setHostLeft(true)
          setCountdown(data.countdown || 10)
          break

        case 'session-ended':
          console.log('[Session] Session ended by host')
          onLeave()
          break

        default:
          console.log('Unknown message type:', data.type)
      }
    }

    socket.onclose = () => {
      console.log('WS Disconnected')
      setWsReady(false)
    }

    return () => {
      setWsReady(false)
      socket.close()
      localStream.current?.getTracks().forEach(t => t.stop())
      Object.values(peerConnections.current).forEach(pc => pc.close())
    }
  }, [sessionData.id, myDeviceId, sessionData.hostIp, sessionData.hostPort])

  // Countdown timer effect for when host leaves
  useEffect(() => {
    if (hostLeft && countdown > 0) {
      const timer = setTimeout(() => {
        setCountdown(countdown - 1)
      }, 1000)
      return () => clearTimeout(timer)
    }
  }, [hostLeft, countdown])

  // WebRTC Signaling Handlers
  const createPeerConnection = (peerId: string) => {
    const pc = new RTCPeerConnection({
      // No STUN/TURN servers needed — all peers are on the same LAN.
      // Empty iceServers forces ICE to use only host candidates (direct LAN IPs).
      iceServers: []
    })

    pc.onicecandidate = (event) => {
      if (event.candidate && ws.current) {
        ws.current.send(JSON.stringify({
          type: 'ice-candidate',
          candidate: event.candidate,
          targetPeerId: peerId // Logic handles relaying
        }))
      }
    }

    pc.ontrack = (event) => {
      setRemoteStreams(prev => ({
        ...prev,
        [peerId]: event.streams[0]
      }))
    }

    pc.onconnectionstatechange = () => {
      const state = pc.connectionState
      if (state === 'failed' || state === 'disconnected' || state === 'closed') {
        removePeerConnection(peerId)
        initiatedCalls.current.delete(peerId)
      }
    }

    if (localStream.current) {
      localStream.current.getTracks().forEach(track => {
        pc.addTrack(track, localStream.current!)
      })
    }

    peerConnections.current[peerId] = pc
    return pc
  }

  const handleOffer = async (data: any) => {
    removePeerConnection(data.sender)
    const pc = createPeerConnection(data.sender)
    await pc.setRemoteDescription(new RTCSessionDescription(data.offer))
    const answer = await pc.createAnswer()
    await pc.setLocalDescription(answer)

    ws.current?.send(JSON.stringify({
      type: 'answer',
      answer: answer,
      targetPeerId: data.sender
    }))
  }

  const handleAnswer = async (data: any) => {
    const pc = peerConnections.current[data.sender]
    if (pc) {
      await pc.setRemoteDescription(new RTCSessionDescription(data.answer))
    }
  }

  const handleICECandidate = async (data: any) => {
    const pc = peerConnections.current[data.sender]
    if (pc) {
      await pc.addIceCandidate(new RTCIceCandidate(data.candidate))
    }
  }

  // Track which peers we've already initiated calls with to avoid double calling
  const initiatedCalls = useRef<Set<string>>(new Set())

  useEffect(() => {
    if (!wsReady || !localStreamLoaded) {
      return
    }

    // Drop stale initiation records so a leaving/rejoining peer can be called again.
    const activePeerIDs = new Set(participants.filter(p => !p.isMe).map(p => p.deviceId))
    initiatedCalls.current.forEach(peerId => {
      if (!activePeerIDs.has(peerId)) {
        initiatedCalls.current.delete(peerId)
      }
    })

    // When members list changes, start negotiation with peers not yet called.
    participants.forEach(member => {
      if (
        !member.isMe &&
        shouldInitiateConnection(myDeviceId, member.deviceId) &&
        !peerConnections.current[member.deviceId] &&
        !initiatedCalls.current.has(member.deviceId)
      ) {
        console.log(`Initiating call to ${member.name} (${member.deviceId})`)
        startCall(member.deviceId)
      }
    })
  }, [participants, wsReady, localStreamLoaded, myDeviceId])

  const startCall = async (peerId: string) => {
    if (!ws.current || ws.current.readyState !== WebSocket.OPEN) {
      return
    }

    if (!localStream.current) {
      console.warn('Local stream not ready, delaying call start for peer', peerId)
      return
    }

    initiatedCalls.current.add(peerId)

    try {
      const pc = createPeerConnection(peerId)
      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)

      ws.current.send(JSON.stringify({
        type: 'offer',
        offer: offer,
        targetPeerId: peerId
      }))
    } catch (err) {
      initiatedCalls.current.delete(peerId)
      removePeerConnection(peerId)
      console.error('Failed to start WebRTC call for peer', peerId, err)
    }
  }

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  // ── HLS Streaming Controls ──────────────────────────────
  const backendBase = `http://${sessionData.hostIp || window.location.hostname}:${sessionData.hostPort || '8080'}`

  /** Upload the selected file to the backend and start streaming */
  const handleFileSelected = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = '' // reset so same file can be re-selected

    // ── HOST: Play locally IMMEDIATELY via blob URL ──────
    const blobUrl = URL.createObjectURL(file)
    hostPlayingLocally.current = true
    setHlsPlaylistUrl(blobUrl)
    setIsStreaming(true)

    // ── Upload in background so guests can watch via HLS ─
    setStreamLoading(true) // shows a small "sharing…" indicator
    try {
      const formData = new FormData()
      formData.append('sessionId', sessionData.id)
      formData.append('file', file)

      const resp = await fetch(`${backendBase}/stream/upload`, {
        method: 'POST',
        body: formData,
      })
      const result = await resp.json().catch(() => ({ error: 'Invalid response' }))
      if (!resp.ok) {
        console.error('[HLS] Upload/start failed:', result.error || resp.status)
        // Host is still playing locally — don't interrupt, just log
      }
      // Don't change the host's playlistUrl (keep blob).
      // The backend's WebSocket broadcast already notifies guests with the HLS URL.
    } catch (err: any) {
      console.error('[HLS] Background upload error:', err)
    } finally {
      setStreamLoading(false)
    }
  }

  const handleStopStream = async () => {
    try {
      await fetch(`${backendBase}/stream/stop`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ sessionId: sessionData.id }),
      })
      // Revoke blob URL if the host was playing locally
      if (hlsPlaylistUrl && hlsPlaylistUrl.startsWith('blob:')) {
        URL.revokeObjectURL(hlsPlaylistUrl)
      }
      hostPlayingLocally.current = false
      setHlsPlaylistUrl(null)
      setIsStreaming(false)
    } catch (err) {
      console.error('[HLS] Stop stream error:', err)
    }
  }



  const handleControlChange = (id: string) => {
    if (id === 'video') {
      const newVideoState = !isVideoOn
      setIsVideoOn(newVideoState)
      localStream.current?.getVideoTracks().forEach(t => t.enabled = newVideoState)
    }
    if (id === 'audio') {
      const newMuteState = !isMuted
      setIsMuted(newMuteState)
      localStream.current?.getAudioTracks().forEach(t => t.enabled = !newMuteState)
    }
    if (id === 'media') {
      if (isStreaming) {
        handleStopStream()
      } else if (isHost) {
        fileInputRef.current?.click()
      } else {
        // Guests can't start streams
        console.log('Only the host can share media')
      }
    }
  }

  const handleSendMessage = (e: React.FormEvent) => {
    e.preventDefault()
    if (!inputValue.trim() || !ws.current) return

    const msg = {
      type: 'chat',
      message: inputValue,
      timestamp: new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    }

    ws.current.send(JSON.stringify(msg))
    setInputValue('')
  }

  return (
    <motion.div
      className="live-session-overlay meet-theme"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      <div className="meet-container">
        {/* Top Header */}
        <header className="meet-header">
          <div className="header-left">
            <ScrambledText
              text={sessionData.name}
              className="meet-session-name"
              duration={1000}
            />
            <span className="meet-session-id">0X-{sessionData.id}</span>
          </div>

          <div className="header-right">
            <motion.button
              className={`meet-utility-btn ${participantsOpen ? 'active' : ''}`}
              onClick={() => {
                setParticipantsOpen(!participantsOpen)
                setChatOpen(false)
              }}
              whileHover={{ scale: 1.1 }}
              whileTap={{ scale: 0.9 }}
              title="Participants"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
              <span>{participants.length}</span>
            </motion.button>
            <motion.button
              className={`meet-utility-btn ${chatOpen ? 'active' : ''}`}
              onClick={() => {
                setChatOpen(!chatOpen)
                setParticipantsOpen(false)
              }}
              whileHover={{ scale: 1.1 }}
              whileTap={{ scale: 0.9 }}
              title="Chat"
            >
              <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"/></svg>
            </motion.button>
          </div>
        </header>

        {/* Hidden file input for host media sharing */}
        <input
          ref={fileInputRef}
          type="file"
          accept="video/*,audio/*"
          style={{ display: 'none' }}
          onChange={handleFileSelected}
        />

        {/* Host Left Overlay */}
        <AnimatePresence>
          {hostLeft && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              style={{
                position: 'fixed',
                inset: 0,
                backgroundColor: 'rgba(0, 0, 0, 0.85)',
                backdropFilter: 'blur(8px)',
                zIndex: 9999,
                display: 'flex',
                flexDirection: 'column',
                justifyContent: 'center',
                alignItems: 'center',
                color: '#fff',
              }}
            >
              <motion.div
                initial={{ scale: 0.8 }}
                animate={{ scale: 1 }}
                style={{
                  background: 'rgba(234, 67, 53, 0.2)',
                  border: '2px solid rgba(234, 67, 53, 0.5)',
                  padding: '2rem 3rem',
                  borderRadius: '16px',
                  textAlign: 'center',
                  boxShadow: '0 10px 40px rgba(234, 67, 53, 0.2)'
                }}
              >
                <div style={{ fontSize: '3rem', marginBottom: '1rem' }}>⚠️</div>
                <h2 style={{ fontSize: '1.5rem', marginBottom: '0.5rem' }}>The host has left</h2>
                <p style={{ color: 'rgba(255,255,255,0.7)', marginBottom: '1.5rem' }}>
                  This session will close automatically in
                </p>
                <div style={{
                  fontSize: '3.5rem',
                  fontWeight: 'bold',
                  fontFamily: 'monospace',
                  color: '#ea4335'
                }}>
                  {countdown}s
                </div>
              </motion.div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Video Call Grid Area */}
        <main className="meet-main">
          {/* HLS Video Player — takes over the full area when streaming */}
          <AnimatePresence>
            {isStreaming && hlsPlaylistUrl && (
              <motion.div
                ref={streamContainerRef}
                key="hls-player"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
                className={`stream-overlay${isFullscreen ? ' stream-fullscreen' : ''}`}
              >
                {/* Top bar with live indicator + controls */}
                <div className="stream-top-bar">
                  <div className="stream-live-badge">
                    <span className="stream-live-dot" />
                    LIVE STREAM
                  </div>
                  <div className="stream-top-actions">
                    {isHost && (
                      <motion.button
                        onClick={handleStopStream}
                        whileHover={{ scale: 1.05 }}
                        whileTap={{ scale: 0.95 }}
                        className="stream-stop-btn"
                      >
                        ⏹ STOP
                      </motion.button>
                    )}
                    <motion.button
                      onClick={() => {
                        if (!document.fullscreenElement) {
                          streamContainerRef.current?.requestFullscreen().then(() => setIsFullscreen(true)).catch(() => {})
                        } else {
                          document.exitFullscreen().then(() => setIsFullscreen(false)).catch(() => {})
                        }
                      }}
                      whileHover={{ scale: 1.1 }}
                      whileTap={{ scale: 0.95 }}
                      className="stream-fullscreen-btn"
                      title={isFullscreen ? 'Exit Fullscreen' : 'Fullscreen'}
                    >
                      {isFullscreen ? '⊡' : '⛶'}
                    </motion.button>
                  </div>
                </div>
                <VideoPlayer
                  playlistUrl={hlsPlaylistUrl}
                  isHost={!!isHost}
                  ws={ws}
                  onStreamEnd={() => {
                    setHlsPlaylistUrl(null)
                    setIsStreaming(false)
                    setIsFullscreen(false)
                  }}
                />
              </motion.div>
            )}
          </AnimatePresence>

          {streamLoading && (
            <div style={{
              color: '#0ff', fontFamily: 'monospace', fontSize: '0.75rem',
              padding: '6px 14px', background: 'rgba(0,0,0,0.7)',
              borderRadius: '0 0 8px 0',
              position: 'absolute', top: 0, left: 0, zIndex: 20,
            }}>
              ↑ Sharing with participants…
            </div>
          )}

          {/* Hide the video call grid when streaming media */}
          <div className={`video-grid count-${participants.length}`} style={isStreaming ? { display: 'none' } : undefined}>
            {participants.map((member, i) => (
              <motion.div
                key={member.id}
                className="video-tile"
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: i * 0.1 }}
                layout
              >
                {member.deviceId === myDeviceId ? (
                  <>
                    <video
                      autoPlay
                      muted
                      playsInline
                      ref={el => {
                        if (el && localStream.current) {
                          if (el.srcObject !== localStream.current) {
                            console.log('Setting local video srcObject')
                            el.srcObject = localStream.current
                          }
                          // Extra nudge for Chrome/Edge
                          if (el.paused) el.play().catch(e => console.error("Play error:", e))
                        }
                      }}
                      className={`video-feed local ${!isVideoOn ? 'hidden' : ''}`}
                    />
                    {!isVideoOn && (
                      <div className="video-avatar absolute">
                        <div className="avatar-placeholder">{member.name[0]}</div>
                      </div>
                    )}
                  </>
                ) : remoteStreams[member.deviceId] ? (
                  <video
                    autoPlay
                    playsInline
                    ref={el => {
                      if (el && remoteStreams[member.deviceId]) {
                        if (el.srcObject !== remoteStreams[member.deviceId]) {
                          el.srcObject = remoteStreams[member.deviceId]
                        }
                        if (el.paused) el.play().catch(() => {})
                      }
                    }}
                    className="video-feed"
                  />
                ) : (
                  <div className="video-avatar">
                    {member.avatar ? <img src={member.avatar} alt={member.name} /> : <div className="avatar-placeholder">{member.name[0]}</div>}
                  </div>
                )}

                <div className="tile-label">
                  {member.deviceId === myDeviceId ? 'You' : member.name}
                </div>
                {/* Simulated Audio Indicator */}
                <div className="audio-indicator">
                  <div className="audio-bars">
                    {[1, 2, 3].map(barIdx => (
                      <motion.div
                        key={barIdx}
                        className="bar"
                        animate={{ height: [2, Math.random() * 8 + 4, 2] }}
                        transition={{ repeat: Infinity, duration: 0.4 + Math.random() * 0.4 }}
                      />
                    ))}
                  </div>
                </div>
              </motion.div>
            ))}
          </div>

          {/* Floating Participants Sidebar */}
          <AnimatePresence>
            {participantsOpen && (
              <motion.aside
                className="meet-participants-sidebar"
                initial={{ x: 400, opacity: 0 }}
                animate={{ x: 0, opacity: 1 }}
                exit={{ x: 400, opacity: 0 }}
                transition={{ type: 'spring', damping: 20 }}
              >
                <div className="sidebar-header">
                  <h3>Participants</h3>
                  <button onClick={() => setParticipantsOpen(false)}>✕</button>
                </div>
                <div className="participants-list">
                  {participants.map((member) => (
                    <div key={member.id} className="participant-row">
                      <div className="p-avatar">{member.name[0]}</div>
                      <span className="p-name">{member.name} {member.deviceId === myDeviceId && '(You)'}</span>
                      <div className="p-controls">
                        🎙️ 📹
                      </div>
                    </div>
                  ))}
                </div>
              </motion.aside>
            )}
          </AnimatePresence>

          {/* Chat Sidebar */}
          <AnimatePresence>
            {chatOpen && (
              <motion.aside
                className="meet-chat-sidebar"
                initial={{ x: 400, opacity: 0 }}
                animate={{ x: 0, opacity: 1 }}
                exit={{ x: 400, opacity: 0 }}
                transition={{ type: 'spring', damping: 20 }}
              >
                <div className="sidebar-header">
                  <h3>Chat</h3>
                  <button onClick={() => setChatOpen(false)}>✕</button>
                </div>

                <div className="chat-messages">
                  {messages.map((msg, i) => (
                    msg.type === 'system' ? (
                      <div key={i} className="message-type-system">
                        {msg.message}
                      </div>
                    ) : (
                      <div key={i} className={`chat-message ${msg.sender === myDeviceId ? 'me' : ''}`}>
                        <div className="message-info">
                          <span className="m-sender">{participants.find(p => p.deviceId === msg.sender)?.name || msg.sender}</span>
                          <span className="m-time">{msg.timestamp}</span>
                        </div>
                        <div className="message-text">
                          {msg.message}
                        </div>
                      </div>
                    )
                  ))}
                  <div ref={chatEndRef} />
                </div>

                <div className="chat-input-area">
                  <form className="chat-form" onSubmit={handleSendMessage}>
                    <input
                      type="text"
                      placeholder="Send a message..."
                      value={inputValue}
                      onChange={(e) => setInputValue(e.target.value)}
                    />
                    <button type="submit" className="chat-send-btn" disabled={!inputValue.trim()}>
                      ➤
                    </button>
                  </form>
                </div>
              </motion.aside>
            )}
          </AnimatePresence>
        </main>

        {/* Floating Control Bar */}
        <footer className="meet-footer">
          <div className="footer-left-info">
            {sessionDuration}
          </div>

          <div className="footer-center">
            <div className="control-bar" style={{
              display: 'flex',
              gap: '8px',
              padding: '6px 12px',
              background: 'rgba(255, 255, 255, 0.05)',
              borderRadius: '999px',
              backdropFilter: 'blur(10px)',
              border: '1px solid rgba(255,255,255,0.08)',
              alignItems: 'center',
            }}>
              {/* Video Toggle */}
              <motion.button
                onClick={() => handleControlChange('video')}
                whileHover={{ scale: 1.1 }}
                whileTap={{ scale: 0.9 }}
                title={isVideoOn ? 'Turn off camera' : 'Turn on camera'}
                style={{
                  padding: '8px 16px',
                  fontSize: '0.85rem',
                  fontWeight: 700,
                  color: 'white',
                  background: isVideoOn ? 'rgba(168, 85, 247, 0.3)' : 'rgba(234, 67, 53, 0.8)',
                  border: 'none',
                  borderRadius: '999px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  boxShadow: isVideoOn ? 'none' : '0 4px 15px rgba(234, 67, 53, 0.4)',
                  transition: 'background 0.2s, box-shadow 0.2s',
                }}
              >
                {isVideoOn ? 'Video' : 'Video Off'}
              </motion.button>

              {/* Audio Toggle */}
              <motion.button
                onClick={() => handleControlChange('audio')}
                whileHover={{ scale: 1.1 }}
                whileTap={{ scale: 0.9 }}
                title={isMuted ? 'Unmute' : 'Mute'}
                style={{
                  padding: '8px 16px',
                  fontSize: '0.85rem',
                  fontWeight: 700,
                  color: 'white',
                  background: isMuted ? 'rgba(234, 67, 53, 0.8)' : 'rgba(168, 85, 247, 0.3)',
                  border: 'none',
                  borderRadius: '999px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                  boxShadow: isMuted ? '0 4px 15px rgba(234, 67, 53, 0.4)' : 'none',
                  transition: 'background 0.2s, box-shadow 0.2s',
                }}
              >
                {isMuted ? 'Muted' : 'Unmuted'}
              </motion.button>

              {/* Screen Share - placeholder */}
              <motion.button
                whileHover={{ scale: 1.1 }}
                whileTap={{ scale: 0.9 }}
                title="Screen share"
                style={{
                  padding: '8px 16px',
                  fontSize: '0.85rem',
                  fontWeight: 700,
                  color: 'rgba(255,255,255,0.6)',
                  background: 'none',
                  border: 'none',
                  borderRadius: '999px',
                  cursor: 'pointer',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '6px',
                }}
              >
                Screen
              </motion.button>

              {/* Media Share (host only) */}
              {isHost && (
                <motion.button
                  onClick={() => handleControlChange('media')}
                  whileHover={{ scale: 1.1 }}
                  whileTap={{ scale: 0.9 }}
                  title={isStreaming ? 'Stop sharing media' : 'Share media'}
                  style={{
                    padding: '8px 16px',
                    fontSize: '0.85rem',
                    fontWeight: 700,
                    color: 'white',
                    background: isStreaming ? 'rgba(234, 67, 53, 0.8)' : 'rgba(168, 85, 247, 0.3)',
                    border: 'none',
                    borderRadius: '999px',
                    cursor: 'pointer',
                    display: 'flex',
                    alignItems: 'center',
                    gap: '6px',
                    boxShadow: isStreaming ? '0 4px 15px rgba(234, 67, 53, 0.4)' : 'none',
                    transition: 'background 0.2s, box-shadow 0.2s',
                  }}
                >
                  {isStreaming ? 'Stop Media' : 'Share Media'}
                </motion.button>
              )}
            </div>

            <motion.button
              className="end-call-btn"
              whileHover={{ scale: 1.1, backgroundColor: '#ea4335' }}
              whileTap={{ scale: 0.9 }}
              onClick={onLeave}
              title="Leave Call"
            >
              End Call
            </motion.button>
          </div>

          <div className="footer-right-info">
            0XNET SECURE • LOCAL
          </div>
        </footer>
      </div>
    </motion.div>
  )
}

export default LiveSession