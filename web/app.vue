<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Send, User, Hash, Settings, LogOut } from 'lucide-vue-next'

const { messages, onlineUsers, sendMessage: apiSendMessage, connect } = useChat()

const newMessage = ref('')
const currentUser = ref('test_user')

const sendMessage = async () => {
  if (!newMessage.value.trim()) return
  
  const content = newMessage.value
  newMessage.value = ''
  
  // Optimistic update
  messages.value.push({
    id: Date.now(),
    sender: currentUser.value,
    content: content,
    self: true
  })

  try {
    await apiSendMessage(content)
  } catch (err) {
    console.error('Failed to send:', err)
  }
}

onMounted(() => {
  connect()
})
</script>

<template>
  <div class="h-screen w-screen bg-background font-sans text-slate-200 flex overflow-hidden">
    <!-- Sidebar -->
    <aside class="w-64 border-r border-white/5 bg-black/20 flex flex-col">
      <div class="p-6 border-b border-white/5">
        <h1 class="font-heading font-bold text-xl text-primary tracking-tight flex items-center gap-2">
          <div class="w-2 h-6 bg-primary rounded-full"></div>
          Antigravity
        </h1>
      </div>
      
      <nav class="flex-1 p-4 space-y-8 overflow-y-auto">
        <div>
          <h3 class="text-xs font-bold uppercase tracking-widest text-slate-500 mb-4 px-2">Channels</h3>
          <div class="space-y-1">
            <a href="#" class="flex items-center gap-2 px-2 py-1.5 rounded-lg bg-white/5 text-primary">
              <Hash class="w-4 h-4" /> global
            </a>
            <a href="#" class="flex items-center gap-2 px-2 py-1.5 rounded-lg hover:bg-white/5 transition-colors">
              <Hash class="w-4 h-4 text-slate-500" /> development
            </a>
          </div>
        </div>

        <div>
          <h3 class="text-xs font-bold uppercase tracking-widest text-slate-500 mb-4 px-2">Presence</h3>
          <div class="space-y-3">
            <div v-for="user in onlineUsers" :key="user.name" class="flex items-center gap-3 px-2">
              <div class="relative">
                <div class="w-8 h-8 rounded-full bg-primary/20 flex items-center justify-center text-primary font-bold">
                  {{ user.name[0].toUpperCase() }}
                </div>
                <div :class="user.status === 'online' ? 'bg-green-500' : 'bg-slate-500'" 
                     class="absolute bottom-0 right-0 w-2.5 h-2.5 rounded-full border-2 border-background"></div>
              </div>
              <span class="text-sm font-medium">{{ user.name }}</span>
            </div>
          </div>
        </div>
      </nav>

      <div class="p-4 border-t border-white/5 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <User class="w-8 h-8 p-1.5 rounded-full bg-white/10" />
          <div class="text-sm">
            <p class="font-bold leading-none">{{ currentUser }}</p>
            <p class="text-xs text-slate-500 mt-1">mTLS Secured</p>
          </div>
        </div>
        <Settings class="w-4 h-4 text-slate-500 hover:text-white cursor-pointer" />
      </div>
    </aside>

    <!-- Main Chat Area -->
    <main class="flex-1 flex flex-col relative">
      <!-- Glow background effect -->
      <div class="absolute top-0 right-0 w-96 h-96 bg-primary/5 rounded-full blur-3xl -z-10 translate-x-1/2 -translate-y-1/2"></div>
      
      <header class="p-4 border-b border-white/5 flex justify-between items-center bg-background/50 backdrop-blur-sm sticky top-0 z-10">
        <div class="flex items-center gap-3">
          <Hash class="w-5 h-5 text-slate-500" />
          <h2 class="font-bold">global</h2>
        </div>
        <div class="flex items-center gap-4 text-xs font-heading">
          <span class="text-slate-500">Node: cluster-01</span>
          <span class="bg-green-500/10 text-green-400 px-2 py-1 rounded border border-green-500/20">mTLS Verified</span>
        </div>
      </header>

      <div class="flex-1 overflow-y-auto p-6 flex flex-col gap-4">
        <div v-for="msg in messages" :key="msg.id" 
             :class="msg.self ? 'chat-bubble-self' : 'chat-bubble-peer'">
          <p class="text-xs font-bold mb-1 opacity-50" v-if="!msg.self">{{ msg.sender }}</p>
          <p class="text-sm leading-relaxed">{{ msg.content }}</p>
        </div>
      </div>

      <footer class="p-6 bg-gradient-to-t from-background to-transparent">
        <div class="glass-card flex items-center gap-4 focus-within:ring-2 ring-primary/50 transition-all">
          <input 
            v-model="newMessage"
            @keyup.enter="sendMessage"
            type="text" 
            placeholder="Write a message..." 
            class="flex-1 bg-transparent border-none outline-none text-sm py-2"
          />
          <button @click="sendMessage" class="btn-primary flex items-center gap-2 !py-1.5 text-sm">
            Send <Send class="w-4 h-4" />
          </button>
        </div>
      </footer>
    </main>
  </div>
</template>

<style>
/* Fira Fonts are imported by UnoCSS webFonts */
body {
  margin: 0;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

::-webkit-scrollbar {
  width: 6px;
}
::-webkit-scrollbar-track {
  background: transparent;
}
::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 10px;
}
::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.1);
}
</style>
