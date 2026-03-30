import React, { useState, useEffect, useRef } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import ScrambledText from './ScrambledText'
import PillNav from './PillNav'
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
  const [activeTab, setActiveTab] = useState('video')
  const [isMuted, setIsMuted] = useState(false)
  const [isVideoOn, setIsVideoOn] = useState(true)
  const [messages, setMessages] = useState<Message[]>([])
  const [inputValue, setInputValue] = useState('')
  const [localStreamLoaded, setLocalStreamLoaded] = useState(false)
  
  const ws = useRef<WebSocket | null>(null)
  const chatEndRef = useRef<HTMLDivElement>(null)
  
  // WebRTC Refs
  const localStream = useRef<MediaStream | null>(null)
  const peerConnections = useRef<{ [peerId: string]: RTCPeerConnection }>({})
  const [remoteStreams, setRemoteStreams] = useState<{ [peerId: string]: MediaStream }>({})

  // Get current user's name
  const myName = sessionData.members.find(m => m.deviceId === myDeviceId)?.name || 'Me'

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
      socket.send(JSON.stringify({
        type: 'join-session',
        sessionId: sessionData.id,
        username: myName
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
          
        default:
          console.log('Unknown message type:', data.type)
      }
    }

    socket.onclose = () => {
      console.log('WS Disconnected')
    }

    return () => {
      socket.close()
      localStream.current?.getTracks().forEach(t => t.stop())
      Object.values(peerConnections.current).forEach(pc => pc.close())
    }
  }, [sessionData.id, myName, localStreamLoaded])

  // WebRTC Signaling Handlers
  const createPeerConnection = (peerId: string) => {
    const pc = new RTCPeerConnection({
      iceServers: [{ urls: 'stun:stun.l.google.com:19302' }]
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

    if (localStream.current) {
      localStream.current.getTracks().forEach(track => {
        pc.addTrack(track, localStream.current!)
      })
    }

    peerConnections.current[peerId] = pc
    return pc
  }

  const handleOffer = async (data: any) => {
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
    // When members list changes, see if there are new people to call
    sessionData.members.forEach(member => {
      if (!member.isMe && !peerConnections.current[member.id] && !initiatedCalls.current.has(member.id)) {
        console.log(`Initiating call to ${member.name} (${member.id})`)
        initiatedCalls.current.add(member.id)
        startCall(member.id)
      }
    })
  }, [sessionData.members])

  const startCall = async (peerId: string) => {
    const pc = createPeerConnection(peerId)
    const offer = await pc.createOffer()
    await pc.setLocalDescription(offer)
    
    ws.current?.send(JSON.stringify({
      type: 'offer',
      offer: offer,
      targetPeerId: peerId
    }))
  }

  useEffect(() => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  const controlItems = [
    { id: 'video', label: isVideoOn ? 'Video' : 'Video Off', icon: isVideoOn ? '📹' : '🚫' },
    { id: 'audio', label: isMuted ? 'Muted' : 'Unmuted', icon: isMuted ? '🔇' : '🎙️' },
    { id: 'screenshare', label: 'Screen', icon: '🖥️' },
    { id: 'settings', label: 'Settings', icon: '⚙️' },
  ]

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
                  ) : remoteStreams[member.id] ? (
                     <video 
                       autoPlay 
                       ref={el => { if (el) el.srcObject = remoteStreams[member.id] }} 
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
                  {sessionData.members.map((member) => (
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
