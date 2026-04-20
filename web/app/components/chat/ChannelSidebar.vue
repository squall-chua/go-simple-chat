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
  <aside class="w-64 bg-slate-900/50 border-r border-white/5 flex flex-col">
    <!-- Header -->
    <div class="p-4 border-b border-white/5 flex items-center justify-between">
      <div class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-lg bg-primary/20 flex items-center justify-center">
          <Hash class="w-5 h-5 text-primary" />
        </div>
        <span class="font-bold text-white tracking-tight">Channels</span>
      </div>
      <div class="flex items-center gap-1">
        <button 
          @click="$emit('new-dm')"
          class="w-8 h-8 rounded-lg bg-white/5 hover:bg-white/10 flex items-center justify-center transition-colors text-slate-400 hover:text-white"
          title="New Direct Message"
        >
          <MessageSquare class="w-4 h-4" />
        </button>
        <button 
          @click="$emit('create-channel')"
          class="w-8 h-8 rounded-lg bg-white/5 hover:bg-white/10 flex items-center justify-center transition-colors text-slate-400 hover:text-white"
          title="New Channel"
        >
          <Plus class="w-5 h-5" />
        </button>
      </div>
    </div>

    <!-- Channel List -->
    <nav class="flex-1 overflow-y-auto p-3 space-y-1 custom-scrollbar">
      <button 
        v-for="channel in channels" 
        :key="channel.id"
        @click="activeChannelId = channel.id"
        :class="[
          'w-full flex items-center justify-between px-3 py-2.5 rounded-xl transition-all group',
          activeChannelId === channel.id 
            ? 'bg-primary/10 text-primary' 
            : 'text-slate-400 hover:bg-white/5 hover:text-white'
        ]"
      >
        <div class="flex items-center gap-3 overflow-hidden">
          <component 
            :is="(channel.type === 'TYPE_DIRECT' || channel.type === 'direct') ? MessageSquare : Hash" 
            :class="['w-5 h-5 shrink-0', activeChannelId === channel.id ? 'text-primary' : 'text-slate-500 group-hover:text-slate-300']"
          />
          <span class="truncate font-medium">{{ getDisplayName(channel) }}</span>
          
          <span v-if="channel.unread_count > 0" class="bg-red-500 text-white text-[10px] font-bold px-2 py-0.5 rounded-full shrink-0 shadow-lg shadow-red-500/20">
            {{ channel.unread_count > 99 ? '99+' : channel.unread_count }}
          </span>
        </div>
      </button>

      <div v-if="channels.length === 0" class="p-8 text-center space-y-2">
        <p class="text-xs text-slate-500 italic">No channels yet</p>
      </div>
    </nav>

    <!-- User Footer -->
    <div class="h-[132px] p-5 bg-slate-900/80 border-t border-white/5 flex flex-col justify-center gap-2">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-xl bg-slate-800 border border-white/10 flex items-center justify-center text-primary font-bold shadow-lg shrink-0">
          {{ username?.charAt(0).toUpperCase() || '?' }}
        </div>
        <div class="flex-1 min-w-0">
          <p class="text-sm font-bold text-white truncate leading-tight">{{ username }}</p>
          <div class="flex items-center gap-1.5 mt-0">
            <div class="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.5)]"></div>
            <p class="text-[10px] text-slate-500 font-bold uppercase tracking-wider">Online</p>
          </div>
        </div>
      </div>
      
      <div class="flex gap-2">
        <button 
          @click="showSuccess('Settings coming soon!')"
          class="flex-1 h-9 rounded-lg bg-white/5 hover:bg-white/10 flex items-center justify-center transition-all border border-white/5 text-slate-500 group" 
          title="Settings"
        >
          <Settings class="w-4 h-4" />
        </button>
        <button 
          @click="logout"
          class="flex-1 h-9 rounded-lg bg-white/5 hover:bg-red-500/10 hover:text-red-500 flex items-center justify-center transition-all border border-white/5 hover:border-red-500/20 text-slate-500 group" 
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
