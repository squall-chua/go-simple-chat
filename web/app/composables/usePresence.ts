import { ref, onMounted, onUnmounted } from 'vue'

const onlineUsers = ref<Set<string>>(new Set())

export const usePresence = () => {
  const { addPresenceListener } = useStream()

  // Handle incoming presence events
  onMounted(() => {
    const cleanup = addPresenceListener((event: { user_id: string, online: boolean }) => {
      if (event.online) {
        onlineUsers.value.add(event.user_id)
      } else {
        onlineUsers.value.delete(event.user_id)
      }
    })
    onUnmounted(() => cleanup())
  })
  
  const config = useRuntimeConfig()
  const { token, userId } = useAuth()

  const fetchPresence = async (userIds: string[]) => {
    if (!token.value || userIds.length === 0) return
    
    for (const id of userIds) {
      if (!id || id === userId.value) continue
      try {
        const response = await fetch(`${config.public.apiBase}/v1/users/${id}/presence`, {
          headers: { 'x-session-token': token.value || '' }
        })
        if (response.ok) {
          const data = await response.json()
          if (data.online) onlineUsers.value.add(id)
          else onlineUsers.value.delete(id)
        }
      } catch (err) {
        console.error('Failed to fetch presence for', id, err)
      }
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
