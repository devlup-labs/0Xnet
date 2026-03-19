import React, { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import ScrambledText from './ScrambledText'
import PillNav from './PillNav'
import './LiveSession.css'

interface Participant {
  id: string
  name: string
  avatar: string
  status: 'online' | 'busy' | 'away'
  role?: 'host' | 'guest'
  isMe?: boolean
}

interface LiveSessionProps {
  sessionData: {
    id: string
    name: string
    activeSince: string
    members: Participant[]
  }
  onLeave: () => void
}

const LiveSession: React.FC<LiveSessionProps> = ({ sessionData, onLeave }) => {
  const [participantsOpen, setParticipantsOpen] = useState(false)
  const [activeTab, setActiveTab] = useState('video')
  const [isMuted, setIsMuted] = useState(false)
  const [isVideoOn, setIsVideoOn] = useState(true)

  const controlItems = [
    { id: 'video', label: isVideoOn ? 'Video' : 'Video Off', icon: isVideoOn ? '📹' : '🚫' },
    { id: 'audio', label: isMuted ? 'Muted' : 'Unmuted', icon: isMuted ? '🔇' : '🎙️' },
    { id: 'screenshare', label: 'Screen', icon: '🖥️' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ]

  // Mock toggle functions
  const handleControlChange = (id: string) => {
    if (id === 'video') setIsVideoOn(!isVideoOn)
    if (id === 'audio') setIsMuted(!isMuted)
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
               onClick={() => setParticipantsOpen(!participantsOpen)}
               whileHover={{ scale: 1.1 }}
               whileTap={{ scale: 0.9 }}
               title="Participants"
             >
               👥 <span>{sessionData.members.length}</span>
             </motion.button>
             <button className="meet-utility-btn" title="Chat">💬</button>
          </div>
        </header>

        {/* Video Call Grid Area */}
        <main className="meet-main">
          <div className={`video-grid count-${sessionData.members.length}`}>
            {sessionData.members.map((member, i) => (
              <motion.div 
                key={member.id} 
                className="video-tile"
                initial={{ opacity: 0, scale: 0.9 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: i * 0.1 }}
                layout
              >
                <div className="video-avatar">
                   {member.avatar ? <img src={member.avatar} alt={member.name} /> : <div className="avatar-placeholder">{member.name[0]}</div>}
                </div>
                <div className="tile-label">
                   {member.isMe ? 'You' : member.name}
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

          {/* Floating Participants Sidebar (Overlay if screen is small) */}
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
                  {sessionData.members.map((member) => (
                    <div key={member.id} className="participant-row">
                      <div className="p-avatar">{member.name[0]}</div>
                      <span className="p-name">{member.name} {member.isMe && '(You)'}</span>
                      <div className="p-controls">
                         🎙️ 📹
                      </div>
                    </div>
                  ))}
                </div>
              </motion.aside>
            )}
          </AnimatePresence>
        </main>

        {/* Floating Control Bar */}
        <footer className="meet-footer">
           <div className="footer-left-info">
              {sessionData.activeSince}
           </div>
           
           <div className="footer-center">
              <PillNav 
                items={controlItems} 
                activeId={activeTab} 
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
