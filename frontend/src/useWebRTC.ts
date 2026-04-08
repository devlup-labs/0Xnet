import { useEffect, useRef, useState, useCallback } from 'react'

interface PeerConnection {
  pc: RTCPeerConnection
  stream: MediaStream | null
}

interface UseWebRTCReturn {
  localStream: MediaStream | null
  remoteStreams: Map<string, MediaStream>
  peers: string[]
  isMuted: boolean
  isVideoOn: boolean
  toggleAudio: () => void
  toggleVideo: () => void
  error: string | null
}

export default function useWebRTC(
  wsUrl: string,
  sessionId: string,
  peerId: string
): UseWebRTCReturn {
  const [localStream, setLocalStream] = useState<MediaStream | null>(null)
  const [remoteStreams, setRemoteStreams] = useState<Map<string, MediaStream>>(new Map())
  const [peers, setPeers] = useState<string[]>([])
  const [isMuted, setIsMuted] = useState(false)
  const [isVideoOn, setIsVideoOn] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const wsRef = useRef<WebSocket | null>(null)
  const peerConnections = useRef<Map<string, PeerConnection>>(new Map())
  const localStreamRef = useRef<MediaStream | null>(null)

  // Create a peer connection for a specific remote peer
  const createPeerConnection = useCallback((remotePeerId: string): RTCPeerConnection => {
    // LAN only — no STUN/TURN servers needed
    const pc = new RTCPeerConnection({ iceServers: [] })

    // Add local tracks to the connection
    if (localStreamRef.current) {
      localStreamRef.current.getTracks().forEach(track => {
        pc.addTrack(track, localStreamRef.current!)
      })
    }

    // Handle incoming remote tracks
    pc.ontrack = (event) => {
      const remoteStream = event.streams[0]
      if (remoteStream) {
        setRemoteStreams(prev => {
          const next = new Map(prev)
          next.set(remotePeerId, remoteStream)
          return next
        })
      }
    }

    // Send ICE candidates to remote peer via WS
    pc.onicecandidate = (event) => {
      if (event.candidate && wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({
          type: 'ice-candidate',
          targetPeerId: remotePeerId,
          fromPeerId: peerId,
          candidate: event.candidate.toJSON()
        }))
      }
    }

    pc.onconnectionstatechange = () => {
      if (pc.connectionState === 'failed' || pc.connectionState === 'disconnected') {
        console.warn(`[WebRTC] Connection to ${remotePeerId} ${pc.connectionState}`)
      }
    }

    peerConnections.current.set(remotePeerId, { pc, stream: null })
    return pc
  }, [peerId])

  // Create an SDP offer and send it to a remote peer
  const createAndSendOffer = useCallback(async (remotePeerId: string) => {
    const pc = createPeerConnection(remotePeerId)
    try {
      const offer = await pc.createOffer()
      await pc.setLocalDescription(offer)

      wsRef.current?.send(JSON.stringify({
        type: 'rtc-offer',
        targetPeerId: remotePeerId,
        fromPeerId: peerId,
        sdp: pc.localDescription
      }))
    } catch (err) {
      console.error(`[WebRTC] Failed to create offer for ${remotePeerId}:`, err)
    }
  }, [createPeerConnection, peerId])

  // Handle incoming SDP offer — create answer
  const handleOffer = useCallback(async (fromPeerId: string, sdp: RTCSessionDescriptionInit) => {
    const pc = createPeerConnection(fromPeerId)
    try {
      await pc.setRemoteDescription(new RTCSessionDescription(sdp))
      const answer = await pc.createAnswer()
      await pc.setLocalDescription(answer)

      wsRef.current?.send(JSON.stringify({
        type: 'rtc-answer',
        targetPeerId: fromPeerId,
        fromPeerId: peerId,
        sdp: pc.localDescription
      }))
    } catch (err) {
      console.error(`[WebRTC] Failed to handle offer from ${fromPeerId}:`, err)
    }
  }, [createPeerConnection, peerId])

  // Handle incoming SDP answer
  const handleAnswer = useCallback(async (fromPeerId: string, sdp: RTCSessionDescriptionInit) => {
    const peerConn = peerConnections.current.get(fromPeerId)
    if (peerConn) {
      try {
        await peerConn.pc.setRemoteDescription(new RTCSessionDescription(sdp))
      } catch (err) {
        console.error(`[WebRTC] Failed to set answer from ${fromPeerId}:`, err)
      }
    }
  }, [])

  // Handle incoming ICE candidate
  const handleIceCandidate = useCallback(async (fromPeerId: string, candidate: RTCIceCandidateInit) => {
    const peerConn = peerConnections.current.get(fromPeerId)
    if (peerConn) {
      try {
        await peerConn.pc.addIceCandidate(new RTCIceCandidate(candidate))
      } catch (err) {
        console.error(`[WebRTC] Failed to add ICE candidate from ${fromPeerId}:`, err)
      }
    }
  }, [])

  // Remove a peer connection
  const removePeer = useCallback((remotePeerId: string) => {
    const peerConn = peerConnections.current.get(remotePeerId)
    if (peerConn) {
      peerConn.pc.close()
      peerConnections.current.delete(remotePeerId)
    }
    setRemoteStreams(prev => {
      const next = new Map(prev)
      next.delete(remotePeerId)
      return next
    })
    setPeers(prev => prev.filter(p => p !== remotePeerId))
  }, [])

  // Main effect — setup local media, WebSocket, and message handling
  useEffect(() => {
    let mounted = true

    const init = async () => {
      // 1. Get local camera + mic
      try {
        const stream = await navigator.mediaDevices.getUserMedia({
          video: true,
          audio: true
        })
        if (!mounted) {
          stream.getTracks().forEach(t => t.stop())
          return
        }
        localStreamRef.current = stream
        setLocalStream(stream)
      } catch (err) {
        console.error('[WebRTC] getUserMedia failed:', err)
        setError('Camera/mic access denied. Please allow permissions and try again.')
        return
      }

      // 2. Connect WebSocket
      const ws = new WebSocket(`${wsUrl}?session=${sessionId}&peerId=${peerId}`)
      wsRef.current = ws

      ws.onopen = () => {
        console.log('[WS] Connected to signaling server')
      }

      ws.onmessage = (event) => {
        let data: any
        try {
          data = JSON.parse(event.data)
        } catch {
          return
        }

        switch (data.type) {
          case 'existing-peers':
            // We just joined — create offers to all existing peers
            const existingPeers = data.peers as string[]
            setPeers(existingPeers)
            existingPeers.forEach(remotePeerId => {
              createAndSendOffer(remotePeerId)
            })
            break

          case 'peer-joined':
            // A new peer joined — they will send us an offer, we just track them
            setPeers(prev => [...prev.filter(p => p !== data.peerId), data.peerId])
            break

          case 'rtc-offer':
            handleOffer(data.fromPeerId, data.sdp)
            break

          case 'rtc-answer':
            handleAnswer(data.fromPeerId, data.sdp)
            break

          case 'ice-candidate':
            handleIceCandidate(data.fromPeerId, data.candidate)
            break

          case 'peer-left':
            removePeer(data.peerId)
            break
        }
      }

      ws.onclose = () => {
        console.log('[WS] Disconnected from signaling server')
      }

      ws.onerror = (err) => {
        console.error('[WS] Error:', err)
      }
    }

    init()

    // Cleanup
    return () => {
      mounted = false

      // Close all peer connections
      peerConnections.current.forEach(({ pc }) => pc.close())
      peerConnections.current.clear()

      // Close WebSocket
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }

      // Stop local media tracks
      if (localStreamRef.current) {
        localStreamRef.current.getTracks().forEach(t => t.stop())
        localStreamRef.current = null
      }
    }
  }, [wsUrl, sessionId, peerId, createAndSendOffer, handleOffer, handleAnswer, handleIceCandidate, removePeer])

  // Toggle audio
  const toggleAudio = useCallback(() => {
    if (localStreamRef.current) {
      const audioTrack = localStreamRef.current.getAudioTracks()[0]
      if (audioTrack) {
        audioTrack.enabled = !audioTrack.enabled
        setIsMuted(!audioTrack.enabled)
      }
    }
  }, [])

  // Toggle video
  const toggleVideo = useCallback(() => {
    if (localStreamRef.current) {
      const videoTrack = localStreamRef.current.getVideoTracks()[0]
      if (videoTrack) {
        videoTrack.enabled = !videoTrack.enabled
        setIsVideoOn(videoTrack.enabled)
      }
    }
  }, [])

  return {
    localStream,
    remoteStreams,
    peers,
    isMuted,
    isVideoOn,
    toggleAudio,
    toggleVideo,
    error
  }
}
