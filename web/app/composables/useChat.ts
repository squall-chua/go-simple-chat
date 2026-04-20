export const useChat = () => {
  const config = useRuntimeConfig()
  const messages = ref<{ id: string|number, sender: string, content: string, self: boolean }[]>([])
  const onlineUsers = ref<{ name: string, status: string }[]>([])
  const socket = ref<WebSocket | null>(null)

  const connect = () => {
    const url = `${config.public.wsBase}/chat.v1.ChatService/BidiStreamChat`
    console.log('Connecting to WS:', url)
    
    // In mTLS mode, browser will prompt/include the cert automatically
    socket.value = new WebSocket(url)

    socket.value.onopen = () => {
      console.log('Connected to Chat Stream')
    }

    socket.value.onmessage = (event) => {
      try {
        const res = JSON.parse(event.data)
        // gRPC-gateway mapping for streaming response
        if (res.result?.messageReceived) {
          const m = res.result.messageReceived
          if (m.senderId !== 'test_user') {
            messages.value.push({
              id: m.messageId || Date.now(),
              sender: m.senderId,
              content: m.content,
              self: false
            })
          }
        }
      } catch (e) {
        console.error('Failed to parse WS message:', e)
      }
    }

    socket.value.onclose = () => {
      console.log('Disconnected. Reconnecting...')
      setTimeout(connect, 3000)
    }
  }

  const sendMessage = async (content: string) => {
    try {
      const response = await fetch(`${config.public.apiBase}/v1/messages`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          channelId: 'global',
          content: content,
          mediaType: 'text'
        })
      })
      
      if (!response.ok) {
        throw new Error('Failed to send message')
      }
      
      return await response.json()
    } catch (err) {
      console.error('Send error:', err)
      throw err
    }
  }

  return {
    messages,
    onlineUsers,
    connect,
    sendMessage
  }
}
