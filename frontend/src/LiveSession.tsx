import React, { useState, useEffect, useRef, useMemo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import ScrambledText from './ScrambledText'
import PillNav from './PillNav'
import useWebRTC from './useWebRTC'
import './LiveSession.css'

interface LiveSessionProps {
  sessionData: {
    id: string
    name: string
    activeSince: string
  }
  wsUrl: string
  peerId: string
  onLeave: () => void
}

// Component to render a video stream into a <video> element
function VideoRenderer({ stream, muted = false }: { stream: MediaStream; muted?: boolean }) {
  const videoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream
    }
  }, [stream])

  return (
    <video
      ref={videoRef}
      autoPlay
      playsInline
      muted={muted}
      className="tile-video"
    />
  )
}

const LiveSession: React.FC<LiveSessionProps> = ({ sessionData, wsUrl, peerId, onLeave }) => {
  const [participantsOpen, setParticipantsOpen] = useState(false)
  const [chatOpen, setChatOpen] = useState(false)
  const [elapsedTime, setElapsedTime] = useState('00h 00m 00s')

  const {
    localStream,
    remoteStreams,
    peers,
    isMuted,
    isVideoOn,
    toggleAudio,
    toggleVideo,
    error
  } = useWebRTC(wsUrl, sessionData.id, peerId)

  // Elapsed time counter
  useEffect(() => {
    const start = Date.now()
    const interval = setInterval(() => {
      const diff = Date.now() - start
      const h = String(Math.floor(diff / 3600000)).padStart(2, '0')
      const m = String(Math.floor((diff % 3600000) / 60000)).padStart(2, '0')
      const s = String(Math.floor((diff % 60000) / 1000)).padStart(2, '0')
      setElapsedTime(`${h}h ${m}m ${s}s`)
    }, 1000)
    return () => clearInterval(interval)
  }, [])

  const controlItems = [
    { id: 'video', label: isVideoOn ? 'Video' : 'Video Off', icon: isVideoOn ? '📹' : '🚫' },
    { id: 'audio', label: isMuted ? 'Muted' : 'Unmuted', icon: isMuted ? '🔇' : '🎙️' },
    { id: 'screenshare', label: 'Screen', icon: '🖥️' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ]

  const handleControlChange = (id: string) => {
    if (id === 'video') toggleVideo()
    if (id === 'audio') toggleAudio()
  }

  // Build the list of all tiles: self + remote peers
  const remotePeerIds = Array.from(remoteStreams.keys())
  const totalCount = 1 + remotePeerIds.length // self + remotes
  const allParticipants = ['You', ...remotePeerIds]

  return (
    <motion.div
      className="live-session-overlay meet-theme"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      exit={{ opacity: 0 }}
    >
      <div className="meet-container">
        {/* Error Banner */}
        {error && (
          <div className="webrtc-error-banner">
            ⚠️ {error}
          </div>
        )}

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
               👥 <span>{totalCount}</span>
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
               💬
             </motion.button>
          </div>
        </header>

        {/* Video Grid */}
        <main className="meet-main">
          <div className={`video-grid count-${Math.min(totalCount, 4)}`}>
            {/* Self tile */}
            <motion.div
              className="video-tile"
              initial={{ opacity: 0, scale: 0.9 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: 0 }}
              layout
            >
              {localStream && isVideoOn ? (
                <VideoRenderer stream={localStream} muted={true} />
              ) : (
                <div className="video-avatar">
                  <div className="avatar-placeholder">Y</div>
                </div>
              )}
              <div className="tile-label">You</div>
              {!isMuted && (
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
              )}
              {isMuted && (
                <div className="muted-indicator">🔇</div>
              )}
            </motion.div>

            {/* Remote peer tiles */}
            {remotePeerIds.map((remotePeerId, i) => {
              const stream = remoteStreams.get(remotePeerId)
              return (
                <motion.div
                  key={remotePeerId}
                  className="video-tile"
                  initial={{ opacity: 0, scale: 0.9 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ delay: (i + 1) * 0.1 }}
                  layout
                >
                  {stream ? (
                    <VideoRenderer stream={stream} />
                  ) : (
                    <div className="video-avatar">
                      <div className="avatar-placeholder">
                        {remotePeerId.charAt(0).toUpperCase()}
                      </div>
                    </div>
                  )}
                  <div className="tile-label">
                    Peer {remotePeerId.substring(0, 6)}
                  </div>
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
              )
            })}
          </div>

          {/* Participants Sidebar */}
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
                  <h3>Participants ({totalCount})</h3>
                  <button onClick={() => setParticipantsOpen(false)}>✕</button>
                </div>
                <div className="participants-list">
                  {/* Self */}
                  <div className="participant-row">
                    <div className="p-avatar">Y</div>
                    <span className="p-name">You</span>
                    <div className="p-controls">
                      {isMuted ? '🔇' : '🎙️'} {isVideoOn ? '📹' : '🚫'}
                    </div>
                  </div>
                  {/* Remote peers */}
                  {remotePeerIds.map(rpId => (
                    <div key={rpId} className="participant-row">
                      <div className="p-avatar">{rpId.charAt(0).toUpperCase()}</div>
                      <span className="p-name">Peer {rpId.substring(0, 6)}</span>
                      <div className="p-controls">🎙️ 📹</div>
                    </div>
                  ))}
                </div>
              </motion.aside>
            )}
          </AnimatePresence>
        </main>

        {/* Control Bar */}
        <footer className="meet-footer">
           <div className="footer-left-info">
              {elapsedTime}
           </div>

           <div className="footer-center">
              <PillNav
                items={controlItems}
                activeId={isVideoOn ? 'video' : (isMuted ? 'audio' : 'video')}
                onChange={handleControlChange}
                className="control-pills"
              />
              <motion.button
                className="end-call-btn"
                whileHover={{ scale: 1.1, backgroundColor: '#ea4335' }}
                whileTap={{ scale: 0.9 }}
                onClick={onLeave}
                title="Leave Call"
              >
                📞
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
