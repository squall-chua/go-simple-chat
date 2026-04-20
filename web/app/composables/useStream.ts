import { ref, onUnmounted } from 'vue'

const socket = ref<WebSocket | null>(null)
const messageHandlers = ref<((data: any) => void)[]>([])
const presenceHandlers = ref<((data: any) => void)[]>([])
const errorHandlers = ref<((data: any) => void)[]>([])

export const useStream = () => {
  const { token } = useAuth()
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

  const connect = () => {
    if (!token.value) return
    
    // In a real prod environment, we'd use a cookie or query param
    // For this bridge, we'll try the query param approach if the proxy supports it,
    // or we assume the browser will send the cookie if we set it in useAuth.
    // Derive WS URL from API base - Point to the explicit gRPC bidirectional stream endpoint
    const apiBase = (config.public.apiBase as string) || 'https://localhost:8080'
    const wsUrl = apiBase.replace(/^http/, 'ws') + `/chat.v1.ChatService/BidiStreamChat?token=${token.value}`
    
    // We set the auth_token cookie in useAuth.login, which the browser will send automatically
    socket.value = new WebSocket(wsUrl)

    socket.value.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data)
        if (data.message_received) {
          messageHandlers.value.forEach(h => h(data.message_received))
        } else if (data.presence_event) {
          presenceHandlers.value.forEach(h => h(data.presence_event))
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

    socket.value.onclose = () => {
      console.log('WebSocket closed, reconnecting in 5s...')
      stopHeartbeat()
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

  onUnmounted(() => disconnect())

  return {
    connect,
    disconnect,
    sendMessage,
    addMessageListener,
    addPresenceListener
  }
}
