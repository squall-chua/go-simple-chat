import { ref, onMounted, onUnmounted } from 'vue'

const onlineUsers = ref<Set<string>>(new Set())

let presenceListenerInitialized = false

export const usePresence = () => {
  const { addPresenceListener } = useStream()

  // Handle incoming presence events (Singleton)
  if (!presenceListenerInitialized && process.client) {
    presenceListenerInitialized = true
    addPresenceListener((event: { user_id: string, online: boolean }) => {
      if (event.online) {
        onlineUsers.value.add(event.user_id)
      } else {
        onlineUsers.value.delete(event.user_id)
      }
    })
  }
  
  const config = useRuntimeConfig()
  const { token, userId } = useAuth()

  const fetchPresence = async (userIds: string[]) => {
    if (!token.value || userIds.length === 0) return
    
    // Filter out empty IDs and self
    const targetIds = userIds.filter(id => id && id !== userId.value)
    if (targetIds.length === 0) return

    try {
      // Build query string for repeated user_ids
      const params = new URLSearchParams()
      targetIds.forEach(id => params.append('user_ids', id))

      const response = await fetch(`${config.public.apiBase}/v1/presence?${params.toString()}`, {
        credentials: 'include'
      })

      if (response.ok) {
        const data = await response.json()
        if (data.presences && Array.isArray(data.presences)) {
          data.presences.forEach((p: { user_id: string, online: boolean }) => {
            if (p.online) onlineUsers.value.add(p.user_id)
            else onlineUsers.value.delete(p.user_id)
          })
        }
      }
    } catch (err) {
      console.error('Failed to fetch bulk presence:', err)
    }
  }

  const isOnline = (id: string) => {
    if (!id) return false
    if (id === userId.value) return true
    return onlineUsers.value.has(id)
  }

  return {
    onlineUsers,
    isOnline,
    fetchPresence
  }
}
