import { ref, onUnmounted } from 'vue'

const socket = ref<WebSocket | null>(null)
const messageHandlers = ref<((data: any) => void)[]>([])
const presenceHandlers = ref<((data: any) => void)[]>([])
const identityHandlers = ref<((data: any) => void)[]>([])
const errorHandlers = ref<((data: any) => void)[]>([])

export const useStream = () => {
  const { token, restoreSession } = useAuth()
  const config = useRuntimeConfig()

  const addMessageListener = (handler: (data: any) => void) => {
    messageHandlers.value.push(handler)
    return () => {
      messageHandlers.value = messageHandlers.value.filter(h => h !== handler)
    }
  }

  const addPresenceListener = (handler: (data: any) => void) => {
    presenceHandlers.value.push(handler)
    return () => {
      presenceHandlers.value = presenceHandlers.value.filter(h => h !== handler)
    }
  }

  const addIdentityListener = (handler: (data: any) => void) => {
    identityHandlers.value.push(handler)
    return () => {
      identityHandlers.value = identityHandlers.value.filter(h => h !== handler)
    }
  }

  const connect = () => {
    if (!token.value) return
    
    // Use configured WebSocket base. Token is now passed automatically via HTTP-only cookie.
    const wsUrl = `${config.public.wsBase}/chat.v1.ChatService/BidiStreamChat`
    
    socket.value = new WebSocket(wsUrl)

    socket.value.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.message_received) {
          messageHandlers.value.forEach(h => h(data.message_received))
        } else if (data.presence_event) {
          presenceHandlers.value.forEach(h => h(data.presence_event))
        } else if (data.identity_event) {
          identityHandlers.value.forEach(h => h(data.identity_event))
        } else if (data.error) {
          errorHandlers.value.forEach(h => h(data.error))
        }
      } catch (e) {
        console.error('Failed to parse WS message:', e)
      }
    }

    socket.value.onerror = (err) => {
      console.error('WebSocket error:', err)
      errorHandlers.value.forEach(h => h(err))
    }

    socket.value.onclose = async () => {
      console.log('WebSocket closed, attempting background re-auth and reconnect in 5s...')
      stopHeartbeat()
      
      // Attempt silent re-authentication in background
      await restoreSession()
      
      setTimeout(connect, 5000)
    }

    startHeartbeat()
  }

  let heartbeatTimer: any = null
  const startHeartbeat = () => {
    stopHeartbeat()
    heartbeatTimer = setInterval(() => {
      if (socket.value?.readyState === WebSocket.OPEN) {
        socket.value.send(JSON.stringify({ heartbeat: {} }))
      }
    }, 30000) // 30s
  }

  const stopHeartbeat = () => {
    if (heartbeatTimer) {
      clearInterval(heartbeatTimer)
      heartbeatTimer = null
    }
  }

  const sendMessage = (payload: any) => {
    if (socket.value?.readyState === WebSocket.OPEN) {
      socket.value.send(JSON.stringify({ send_message: payload }))
    }
  }

  const disconnect = () => {
    stopHeartbeat()
    if (socket.value) {
      socket.value.onclose = null // Prevent auto-reconnect
      socket.value.close()
      socket.value = null
    }
  }

  return {
    connect,
    disconnect,
    sendMessage,
    addMessageListener,
    addPresenceListener,
    addIdentityListener
  }
}
