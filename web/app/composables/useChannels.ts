import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { channels, activeChannelId, messages } from './useChatState'
import type { Channel } from './useChatState'

const processedMessageIds = new Set<string>()

const lastMarkedIds = new Map<string, string>()

const markAsReadInternal = async (channelId: string, messageId: string) => {
  const { token } = useAuth()
  const config = useRuntimeConfig()
  
  if (!token.value) return
  if (lastMarkedIds.get(channelId) === messageId) return
  lastMarkedIds.set(channelId, messageId)

  try {
    const response = await fetch(`${config.public.apiBase}/v1/channels/${channelId}/read`, {
      method: 'POST',
      headers: { 
        'x-session-token': token.value || '',
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ channel_id: channelId, message_id: messageId })
    })
    
    if (response.ok) {
      const channel = channels.value.find(c => c.id === channelId)
      if (channel) {
        channel.unread_count = 0
        channel.last_read_id = messageId
      }
    } else {
      lastMarkedIds.delete(channelId)
    }
  } catch (err: any) {
    lastMarkedIds.delete(channelId)
  }
}

let initialized = false
const initGlobalListener = () => {
  if (initialized) return
  initialized = true
  const { addMessageListener } = useStream()
  const { userId } = useAuth()
  
  addMessageListener((msg) => {
    if (processedMessageIds.has(msg.message_id)) return
    processedMessageIds.add(msg.message_id)

    if (msg.sender_id !== userId.value && msg.channel_id !== activeChannelId.value) {
      const channel = channels.value.find(c => c.id === msg.channel_id)
      if (channel) {
        channel.unread_count = (channel.unread_count || 0) + 1
      }
    }
  })
}

// Global watcher for marking as read (singleton)
watch(
  [() => activeChannelId.value, () => messages.value.length], 
  ([channelId, msgCount]) => {
    if (channelId && msgCount > 0) {
      const lastMsg = messages.value[msgCount - 1]
      // Critical check: ensure the message actually belongs to this channel
      if (!lastMsg || lastMsg.channel_id !== channelId) return

      const channel = channels.value.find(c => c.id === channelId)
      if (channel && lastMsg && (channel.unread_count > 0 || channel.last_read_id !== lastMsg.message_id)) {
        markAsReadInternal(channelId, lastMsg.message_id)
      }
    }
  }, 
  { immediate: true }
)

export const useChannels = () => {
  const config = useRuntimeConfig()
  const { token } = useAuth()
  const { showError } = useToast()

  if (process.client) {
    initGlobalListener()
  }

  const fetchChannels = async () => {
    if (!token.value) return
    
    try {
      const response = await fetch(`${config.public.apiBase}/v1/channels`, {
        headers: { 'x-session-token': token.value || '' }
      })
      if (!response.ok) throw new Error('Failed to fetch channels')
      
      const data = await response.json()
      channels.value = data.channels || []
      
      // If no active channel, pick the first one
      if (!activeChannelId.value && channels.value && channels.value.length > 0) {
        activeChannelId.value = channels.value[0]?.id || null
      }
    } catch (err: any) {
      showError(err.message)
    }
  }

  const createChannel = async (name: string, type: string, participants: string[], usernames: string[] = []) => {
    try {
      const response = await fetch(`${config.public.apiBase}/v1/channels`, {
        method: 'POST',
        headers: { 
          'x-session-token': token.value || '',
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ name, type, participants, participant_usernames: usernames })
      })
      if (!response.ok) throw new Error('Failed to create channel')
      
      const data = await response.json()
      await fetchChannels()
      activeChannelId.value = data.channel_id
      return data.channel_id
    } catch (err: any) {
      showError(err.message)
      return null
    }
  }

  const markAsRead = async (channelId: string, messageId: string) => {
    return markAsReadInternal(channelId, messageId)
  }

  const activeChannel = computed(() => 
    channels.value.find(c => c.id === activeChannelId.value) || null
  )

  watch(activeChannelId, (newId) => {
    if (newId) {
      const channel = channels.value.find(c => c.id === newId)
      if (channel) channel.unread_count = 0
    }
  })

  return {
    channels,
    activeChannelId,
    activeChannel,
    fetchChannels,
    createChannel,
    markAsRead
  }
}
