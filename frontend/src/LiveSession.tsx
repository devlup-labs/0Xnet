import React, { useState, useEffect, useRef } from 'react'
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

interface Message {
  type: 'chat' | 'system'
  sender?: string
  message: string
  timestamp: string
}

interface LiveSessionProps {
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

const LiveSession: React.FC<LiveSessionProps> = ({ sessionData, onLeave }) => {
  const [participantsOpen, setParticipantsOpen] = useState(false)
  const [chatOpen, setChatOpen] = useState(false)
  const [activeTab, setActiveTab] = useState('video')
  const [isMuted, setIsMuted] = useState(false)
  const [isVideoOn, setIsVideoOn] = useState(true)
  const [messages, setMessages] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState('')
  
  const ws = useRef<WebSocket | null>(null)
  const chatEndRef = useRef<HTMLDivElement>(null)

  // Get current user's name
  const myName = sessionData.members.find(m => m.isMe)?.name || 'Me'

  useEffect(() => {
    // Determine backend port based on current frontend port (simulation)
    const backendPort = '8080'
    const targetHost = sessionData.hostIp || window.location.hostname
    const targetPort = sessionData.hostPort || backendPort
    
    // Create WebSocket connection
    const socket = new WebSocket(`ws://${targetHost}:${targetPort}/ws`)
    ws.current = socket

    socket.onopen = () => {
      console.log('WS Connected')
      // Join session
      socket.send(JSON.stringify({
        type: 'join-session',
        sessionId: sessionData.id,
        username: myName
      }))
    }

    socket.onmessage = (event) => {
      const data = JSON.parse(event.data)
      setMessages(prev => [...prev, data])
    }

    socket.onclose = () => {
      console.log('WS Disconnected')
    }

    return () => {
      socket.close()
    }
  }, [sessionData.id, myName])

  useEffect(() => {
    // Scroll to bottom when messages change
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const controlItems = [
    { id: 'video', label: isVideoOn ? 'Video' : 'Video Off', icon: isVideoOn ? '📹' : '🚫' },
    { id: 'audio', label: isMuted ? 'Muted' : 'Unmuted', icon: isMuted ? '🔇' : '🎙️' },
    { id: 'screenshare', label: 'Screen', icon: '🖥️' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ]

  const handleControlChange = (id: string) => {
    if (id === 'video') setIsVideoOn(!isVideoOn)
    if (id === 'audio') setIsMuted(!isMuted)
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
               👥 <span>{sessionData.members.length}</span>
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
                      <div key={i} className={`chat-message ${msg.sender === myName ? 'me' : ''}`}>
                        <div className="message-info">
                          <span className="m-sender">{msg.sender}</span>
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
