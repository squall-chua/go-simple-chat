<script setup lang="ts">
import { Hash, MessageSquare, Plus, LogOut, Settings } from 'lucide-vue-next'

const { logout, username } = useAuth()
const { activeChannelId, channels } = useChannels()
const { showSuccess } = useToast()

defineEmits(['create-channel', 'new-dm'])

const getDisplayName = (channel: any) => {
  if (!channel) return ''
  const isDirect = channel.type === 'TYPE_DIRECT' || channel.type === 'direct'
  if (isDirect) {
    const other = channel.participant_usernames.find((u: string) => u !== username.value)
    return other || channel.name || 'Private Chat'
  }
  return channel.name
}
</script>

<template>
  <aside class="w-64 bg-black border-r border-white/10 flex flex-col">
    <!-- Header -->
    <div class="p-4 border-b border-white/10 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-lg bg-sky-500/20 flex items-center justify-center">
          <Hash class="w-5 h-5 text-sky-400" />
        </div>
        <span class="font-bold text-white tracking-tight">Channels</span>
      </div>
      <div class="flex items-center gap-1">
        <button 
          @click="$emit('new-dm')"
          class="w-8 h-8 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors text-slate-300 hover:text-white border border-white/5"
          title="New Direct Message"
        >
          <MessageSquare class="w-4 h-4" />
        </button>
        <button 
          @click="$emit('create-channel')"
          class="w-8 h-8 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors text-slate-300 hover:text-white border border-white/5"
          title="New Channel"
        >
          <Plus class="w-5 h-5" />
        </button>
      </div>
    </div>

    <!-- Channel List -->
    <nav class="flex-1 overflow-y-auto p-2 space-y-0.5 custom-scrollbar">
      <button 
        v-for="channel in channels" 
        :key="channel.id"
        @click="activeChannelId = channel.id"
        :class="[
          'w-full flex items-center justify-between px-3 py-2 rounded-lg transition-all group font-medium relative',
          activeChannelId === channel.id 
            ? 'bg-white/10 text-white' 
            : 'text-slate-400 hover:bg-white/5 hover:text-slate-200'
        ]"
      >
        <!-- Active indicator line -->
        <div v-if="activeChannelId === channel.id" class="absolute left-[-8px] top-2 bottom-2 w-1 bg-sky-400 rounded-r-full"></div>

        <div class="flex items-center gap-3 overflow-hidden">
          <component 
            :is="(channel.type === 'TYPE_DIRECT' || channel.type === 'direct') ? MessageSquare : Hash" 
            :class="['w-5 h-5 shrink-0', activeChannelId === channel.id ? 'text-white' : 'text-slate-500 group-hover:text-slate-300']"
          />
          <span class="truncate">{{ getDisplayName(channel) }}</span>
          
          <span v-if="channel.unread_count > 0" class="bg-sky-500 text-white text-[10px] font-bold px-2 py-0.5 rounded-full shrink-0 shadow-lg shadow-sky-500/20">
            {{ channel.unread_count > 99 ? '99+' : channel.unread_count }}
          </span>
        </div>
      </button>

      <div v-if="channels.length === 0" class="p-8 text-center space-y-2">
        <p class="text-xs text-slate-500 italic">No channels yet</p>
      </div>
    </nav>

    <!-- User Footer -->
    <div class="h-[132px] p-5 bg-slate-900/40 border-t border-white/10 flex flex-col justify-center gap-2">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-xl bg-slate-800 border border-white/20 flex items-center justify-center text-sky-400 font-bold shadow-lg shrink-0">
          {{ username?.charAt(0).toUpperCase() || '?' }}
        </div>
        <div class="flex-1 min-w-0">
          <p class="text-sm font-bold text-white truncate leading-tight">{{ username }}</p>
          <div class="flex items-center gap-1.5 mt-0">
            <div class="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.8)]"></div>
            <p class="text-[10px] text-slate-400 font-bold uppercase tracking-wider">Online</p>
          </div>
        </div>
      </div>
      
      <div class="flex gap-2">
        <button 
          @click="showSuccess('Settings coming soon!')"
          class="flex-1 h-9 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-all border border-white/10 text-slate-200 group" 
          title="Settings"
        >
          <Settings class="w-4 h-4" />
        </button>
        <button 
          @click="logout"
          class="flex-1 h-9 rounded-lg bg-white/10 hover:bg-red-500/20 hover:text-red-400 flex items-center justify-center transition-all border border-white/10 hover:border-red-500/30 text-slate-200 group" 
          title="Logout"
        >
          <LogOut class="w-4 h-4" />
        </button>
      </div>
    </div>
  </aside>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 4px;
}
</style>
