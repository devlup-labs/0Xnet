import React, { Suspense, useEffect, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import Particles from './Particles'
import BlurText from './BlurText'
import LiveSession from './LiveSession'
import ScrambledText from './ScrambledText'
import PillNav from './PillNav'
class ErrorBoundary extends React.Component<{ children: React.ReactNode }, { hasError: boolean }> {
  state = { hasError: false }

  static getDerivedStateFromError() {
    return { hasError: true }
  }

  componentDidCatch(error: unknown) {
    console.error('ErrorBoundary caught:', error)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="errorBanner">
          <p>
            Interactive effects are unavailable. Your app is still usable, but the
            animated background has been disabled.
          </p>
        </div>
      )
    }

    return this.props.children
  }
}

interface SessionData {
  id: string
  name: string
  devices: number
  status: 'active' | 'idle'
  members?: number
}

function MainContent({ onJoin, onCreateClicked }: { onJoin: (session: SessionData) => void, onCreateClicked: () => void }) {
  const [sessions, setSessions] = useState<SessionData[]>([
    { id: '01', name: 'Session 01', devices: 2, status: 'active', members: 3 },
    { id: '02', name: 'Session 02', devices: 1, status: 'active', members: 1 },
    { id: '03', name: 'Session 03', devices: 0, status: 'idle', members: 0 },
    { id: '04', name: 'Session 04', devices: 5, status: 'active', members: 5 },
  ])

  return (
    <main className="main-content">
      <div className="intro-section">
        <div className="status-badge">
          <span className="status-dot">•</span>
          OFFLINE READY
        </div>

        <BlurText
          text="OXNET"
          delay={150}
          animateBy="letters"
          direction="top"
          className="main-title"
        />

        <p className="tagline">No internet, just connect</p>
        <p className="subtitle">Connect devices instantly</p>

        <button className="create-session-btn" onClick={onCreateClicked}>
          <span>✚</span> Create Session
        </button>
      </div>

      <section className="sessions-section">
        <div className="sessions-section-header">
          <h2 className="section-title">AVAILABLE SESSIONS</h2>
          <button className="search-session-btn" onClick={() => console.log('Search sessions')}>
            Search Session 🔍
          </button>
        </div>
        <div className="sessions-container-scroll">
          <div className="sessions-grid">
            {sessions.map((session) => (
              <div key={session.id} className="session-card">
                <div className="session-id">#{session.id}</div>
                <div className="session-info">
                  <div className="session-header">
                    <span className="session-label">{session.name}</span>
                  </div>
                  <div className="session-status">
                    <span>{session.devices} Connected</span>
                  </div>
                </div>
                <button className="join-btn" onClick={() => onJoin(session as SessionData)}>Join ▸</button>
              </div>
            ))}
          </div>
        </div>
      </section>
    </main>
  )
}

function SidePanel({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  const [username, setUsername] = useState('User')

  const getInitials = (name: string) => {
    return name.split(' ').map(n => n[0]).join('').toUpperCase()
  }

  return (
    <>
      <div className={`panel-overlay ${isOpen ? 'active' : ''}`} onClick={onClose} />
      <aside className={`side-panel ${isOpen ? 'open' : ''}`}>
        <div className="panel-header">
          <h2 className="panel-title">Menu</h2>
          <button className="close-btn" onClick={onClose}>✕</button>
        </div>

        <div className="panel-content">
          <div className="panel-section profile-section">
            <div className="profile-dp-container">
              <div className="profile-dp">
                {getInitials(username)}
              </div>
            </div>

            <h3 className="panel-section-title">USER PROFILE</h3>
            <div className="profile-item">
              <label className="profile-label">USER NAME</label>
              <input
                type="text"
                className="profile-input"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="Enter username"
              />
            </div>
            <div className="profile-item">
              <label className="profile-label">DEVICE ID</label>
              <input type="text" className="profile-input" value="0X-416V-D3AJ" readOnly />
            </div>
            <div className="profile-item">
              <label className="profile-label">LAN ADDRESS</label>
              <input type="text" className="profile-input" value="192.168.4.130" readOnly />
            </div>
            <button className="save-btn">Save Profile</button>
          </div>

          <div className="panel-section">
            <h3 className="panel-section-title">NAVIGATION</h3>
            <div className="nav-buttons">
              <button className="nav-btn" onClick={() => console.log('Navigate to Settings')}>
                <span className="nav-icon">⚙</span> Settings
              </button>
              <button className="nav-btn" onClick={() => console.log('Navigate to About')}>
                <span className="nav-icon">ℹ</span> About 0xnet
              </button>
            </div>
          </div>

          <div className="panel-section footer-section">
            <div className="version-tag">v0.1.0-alpha</div>
          </div>
        </div>
      </aside>
    </>
  )
}



export default function App() {
  const [panelOpen, setPanelOpen] = useState(false)
  const [activeSession, setActiveSession] = useState<SessionData | null>(null)
  const [isCreating, setIsCreating] = useState(false)
  const [newSessionName, setNewSessionName] = useState('')

  const handleLogoClick = () => {
    setPanelOpen(false)
    setActiveSession(null)
    const mainContent = document.querySelector('.main-content')
    if (mainContent) {
      mainContent.scrollTo({ top: 0, behavior: 'smooth' })
    }
  }

  const handleCreateSession = () => {
    if (!newSessionName.trim()) return
    const session: SessionData = {
      id: Math.floor(Math.random() * 100).toString().padStart(2, '0'),
      name: newSessionName,
      devices: 1,
      status: 'active'
    }
    setActiveSession(session)
    setIsCreating(false)
    setNewSessionName('')
  }

  return (
    <div className="app">
      <ErrorBoundary>
        <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, zIndex: 0 }}>
          <Particles
            particleColors={['#a855f7', '#B19EEF', '#9C90AA', '#ffffff']}
            particleCount={240}
            particleSpread={15}
            speed={0.06}
            particleBaseSize={160}
            moveParticlesOnHover={true}
            alphaParticles={true}
            disableRotation={false}
          />
        </div>

        <header className="top-header">
          <button className="header-logo" onClick={handleLogoClick}>0xnet</button>
          <button className="menu-btn" onClick={() => setPanelOpen(!panelOpen)}>
            ☰
          </button>
        </header>

        <AnimatePresence mode="wait">
          {!activeSession ? (
            <motion.div
              key="home"
              initial={{ opacity: 0, scale: 0.95 }}
              animate={{ opacity: 1, scale: 1 }}
              exit={{ opacity: 0, scale: 1.05, filter: 'blur(10px)' }}
              transition={{ duration: 0.6, ease: [0.19, 1, 0.22, 1] }}
            >
              <MainContent onJoin={(s) => setActiveSession(s)} onCreateClicked={() => setIsCreating(true)} />
            </motion.div>
          ) : (
            <LiveSession 
              key="live-session"
              sessionData={{
                id: activeSession.id,
                name: activeSession.name,
                activeSince: '00h 00m 00s',
                members: [
                  { id: '1', name: 'You', avatar: '', status: 'online', role: 'host', isMe: true },
                  { id: '2', name: 'Alice', avatar: '', status: 'online', role: 'guest' },
                ]
              }}
              onLeave={() => setActiveSession(null)}
            />
          )}
        </AnimatePresence>

        <AnimatePresence>
          {isCreating && (
            <div className="create-session-overlay">
               <motion.div 
                 className="create-session-panel"
                 initial={{ opacity: 0, y: 50, scale: 0.9 }}
                 animate={{ opacity: 1, y: 0, scale: 1 }}
                 exit={{ opacity: 0, y: 20, scale: 0.9 }}
                 transition={{ type: 'spring', damping: 25 }}
               >
                 <h2>Create New Session</h2>
                 <p>Give your shared 0XNET workspace a name</p>
                 <input 
                   autoFocus
                   type="text" 
                   className="session-name-input" 
                   placeholder="e.g. Design Sync" 
                   value={newSessionName}
                   onChange={(e) => setNewSessionName(e.target.value)}
                   onKeyDown={(e) => e.key === 'Enter' && handleCreateSession()}
                 />
                 <div className="panel-actions">
                   <button className="cancel-btn" onClick={() => setIsCreating(false)}>Cancel</button>
                   <button className="submit-create-btn" onClick={handleCreateSession}>Launch Session</button>
                 </div>
               </motion.div>
            </div>
          )}
        </AnimatePresence>
        
        <SidePanel isOpen={panelOpen} onClose={() => setPanelOpen(false)} />
      </ErrorBoundary>
    </div>
  )
}
