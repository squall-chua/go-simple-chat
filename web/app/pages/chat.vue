<script setup lang="ts">
import { ref, onMounted } from 'vue'

const { isAuthenticated, token } = useAuth()
const { connect } = useStream()
const { fetchChannels } = useChannels()
const { fetchMessages } = useMessages()

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
      <ChatMessageList />
      <ChatMessageInput />
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
