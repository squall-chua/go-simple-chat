<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const { isAuthenticated, token } = useAuth()
const { connect, disconnect, addIdentityListener } = useStream()
const { fetchChannels } = useChannels()
const { fetchMessages } = useMessages()

// Identity state
const identityWarning = ref<string | null>(null)
const isExpired = ref(false)

let removeIdentityListener: (() => void) | null = null

// Modal states
const showCreateChannel = ref(false)
const showNewDM = ref(false)
const showAddParticipant = ref(false)

onMounted(async () => {
  if (!isAuthenticated.value) {
    navigateTo('/')
    return
  }
  
  // Initialize chat
  await fetchChannels()
  connect()

  // Handle identity signals
  removeIdentityListener = addIdentityListener((event) => {
    if (event.type === 'TYPE_EXPIRING_SOON') {
      const date = new Date(event.expires_at.seconds * 1000)
      identityWarning.value = `Your identity certificate is expiring soon (${date.toLocaleString()}). Please renew now to avoid disconnection.`
    } else if (event.type === 'TYPE_EXPIRED') {
      isExpired.value = true
      // Forced disconnection happens on server side, we just show UI
    }
  })
})

onUnmounted(() => {
  disconnect()
  if (removeIdentityListener) removeIdentityListener()
})
</script>

<template>
  <div class="h-screen flex overflow-hidden bg-slate-950">
    <!-- Sidebar -->
    <ChatChannelSidebar 
      @create-channel="showCreateChannel = true"
      @new-dm="showNewDM = true"
    />

    <!-- Main Chat Area -->
    <main class="flex-1 flex flex-col min-w-0 min-h-0 h-full relative">
      <!-- Expiration Banner -->
      <div v-if="identityWarning && !isExpired" class="bg-amber-500/10 border-b border-amber-500/20 px-4 py-2 flex items-center justify-between">
        <div class="flex items-center gap-2 text-amber-400 text-sm font-medium">
          <Icon name="ph:warning-circle-bold" class="w-5 h-5" />
          {{ identityWarning }}
        </div>
        <NuxtLink to="/renew" class="bg-amber-500 hover:bg-amber-600 text-slate-950 text-xs font-bold px-3 py-1 rounded transition-colors">
          RENEW NOW
        </NuxtLink>
      </div>

      <ChatMessageList />
      <ChatMessageInput />

      <!-- Expired Overlay -->
      <div v-if="isExpired" class="absolute inset-0 z-50 bg-slate-950/90 backdrop-blur-md flex flex-col items-center justify-center p-8 text-center">
        <div class="w-20 h-20 bg-rose-500/10 rounded-full flex items-center justify-center mb-6">
          <Icon name="ph:lock-keyhole-bold" class="text-rose-500 w-10 h-10" />
        </div>
        <h2 class="text-2xl font-bold text-white mb-2">Identity Expired</h2>
        <p class="text-slate-400 max-w-md mb-8">
          Your cryptographic certificate has expired. For security reasons, you have been disconnected from the secure chat core.
        </p>
        <NuxtLink to="/renew" class="bg-slate-100 hover:bg-white text-slate-950 font-bold px-8 py-3 rounded-xl transition-all shadow-xl shadow-white/5">
          Renew Certificate
        </NuxtLink>
      </div>
    </main>

    <!-- Participant Panel -->
    <ChatParticipantPanel 
      @add-participant="showAddParticipant = true"
    />

    <!-- Modals -->
    <ModalsCreateChannelModal 
      :show="showCreateChannel" 
      @close="showCreateChannel = false" 
    />
    
    <ModalsAddParticipantModal 
      :show="showAddParticipant" 
      @close="showAddParticipant = false" 
    />

    <ModalsNewDMModal 
      :show="showNewDM"
      @close="showNewDM = false"
    />
  </div>
</template>

<style>
/* Global page transitions or styles if needed */
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 10px;
}
</style>
