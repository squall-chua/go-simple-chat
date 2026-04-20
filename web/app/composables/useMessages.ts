import { ref, watch, onMounted, onUnmounted } from 'vue'
import { activeChannelId, messages, isLoadingMessages as isLoading } from './useChatState'
import type { Message } from './useChatState'

export const useMessages = () => {
  const config = useRuntimeConfig()
  const { token, userId } = useAuth()
  const { activeChannelId } = useChannels()
  const { addMessageListener } = useStream()
  const { showError } = useToast()

  const fetchMessages = async (channelId: string) => {
    if (!token.value) return
    
    isLoading.value = true
    try {
      const response = await fetch(`${config.public.apiBase}/v1/channels/${channelId}/messages?limit=50`, {
        headers: { 'x-session-token': token.value || '' }
      })
      if (!response.ok) throw new Error('Failed to fetch messages')
      
      const data = await response.json()
      // Sort: history usually comes newest first (desc), we want oldest first (asc) for display
      const fetchedMessages = data.messages || []
      messages.value = [...fetchedMessages].reverse()
    } catch (err: any) {
      showError(err.message)
    } finally {
      isLoading.value = false
    }
  }

  const sendMessage = async (content: string, medias: any[] = []) => {
    if (!activeChannelId.value || !token.value) return
    
    try {
      const response = await fetch(`${config.public.apiBase}/v1/channels/${activeChannelId.value}/messages`, {
        method: 'POST',
        headers: { 
          'x-session-token': token.value || '',
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ content, medias })
      })
      if (!response.ok) throw new Error('Failed to send message')
    } catch (err: any) {
      showError(err.message)
    }
  }

  // Handle incoming messages from stream
  onMounted(() => {
    const cleanup = addMessageListener((msg: Message) => {
      if (msg.channel_id === activeChannelId.value) {
        // Append if not already there (prevent double echo if any)
        if (!messages.value.find(m => m.message_id === msg.message_id)) {
          messages.value.push(msg)
        }
      }
    })
    onUnmounted(() => cleanup())
  })

  return {
    messages,
    isLoading,
    fetchMessages,
    sendMessage
  }
}

// Global watcher for channel changes (singleton)
let watcherInitialized = false
watch(activeChannelId, (newId) => {
  if (newId) {
    const { fetchMessages } = useMessages()
    fetchMessages(newId)
  } else {
    messages.value = []
  }
}, { immediate: true })
