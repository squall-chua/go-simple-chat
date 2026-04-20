<script setup lang="ts">
import { watch } from 'vue'
import { Users, UserPlus, Info } from 'lucide-vue-next'

const { activeChannel } = useChannels()
const { onlineUsers, isOnline, fetchPresence } = usePresence()

watch(() => activeChannel.value?.id, (id) => {
  if (id && activeChannel.value) {
    fetchPresence(activeChannel.value.participants)
  }
}, { immediate: true })

defineEmits(['add-participant'])
</script>

<template>
  <aside class="w-64 bg-black border-l border-white/10 flex flex-col hidden lg:flex">
    <!-- Header -->
    <div class="p-4 border-b border-white/10 flex items-center justify-between bg-black">
      <div class="flex items-center gap-2">
        <Users class="w-5 h-5 text-sky-400" />
        <span class="font-bold text-white tracking-tight">Participants</span>
      </div>
      <button 
        v-if="activeChannel?.type !== 'TYPE_DIRECT' && activeChannel?.type !== 'direct'"
        @click="$emit('add-participant')"
        class="w-7 h-7 rounded-lg bg-white/10 hover:bg-white/20 flex items-center justify-center transition-colors text-slate-300 hover:text-white border border-white/5"
        title="Add Participant"
      >
        <UserPlus class="w-4 h-4" />
      </button>
    </div>

    <!-- Participant List -->
    <div class="flex-1 overflow-y-auto p-4 space-y-4 custom-scrollbar">
      <div v-if="activeChannel" class="space-y-4">
        <!-- Direct Message Info -->
        <div v-if="activeChannel.type === 'TYPE_DIRECT'" class="p-3 rounded-xl bg-sky-500/10 border border-sky-500/20 space-y-2">
          <div class="flex items-center gap-2 text-sky-400">
            <Info class="w-4 h-4" />
            <span class="text-[10px] font-black uppercase tracking-widest">About DM</span>
          </div>
          <p class="text-[11px] text-slate-300 leading-relaxed font-medium">
            Direct messages are private conversations between exactly two users.
          </p>
        </div>

        <!-- Participants list -->
        <div class="space-y-3">
          <label class="text-[10px] font-black uppercase tracking-[0.2em] text-slate-400 px-1">Members — {{ activeChannel.participant_usernames.length }}</label>
          <div class="space-y-1">
            <div 
              v-for="(username, index) in activeChannel.participant_usernames" 
              :key="index"
              class="flex items-center gap-3 px-2 py-2 rounded-xl hover:bg-white/10 transition-all group border border-transparent hover:border-white/5"
            >
              <div class="relative">
                <div class="w-8 h-8 rounded-lg bg-slate-900 border border-white/20 flex items-center justify-center text-sky-400 text-xs font-black shadow-lg">
                  {{ username.charAt(0).toUpperCase() }}
                </div>
                <div 
                  v-if="activeChannel.participants[index] && isOnline(activeChannel.participants[index])"
                  class="absolute -bottom-0.5 -right-0.5 w-2.5 h-2.5 rounded-full bg-green-500 border-2 border-black shadow-[0_0_8px_rgba(34,197,94,0.8)]"
                ></div>
              </div>
              <div class="flex-1 overflow-hidden">
                <p class="text-sm font-bold text-slate-200 truncate tracking-tight group-hover:text-white transition-colors">{{ username }}</p>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="h-full flex flex-col items-center justify-center text-center p-8 space-y-4">
        <div class="w-16 h-16 rounded-2xl bg-white/5 border border-white/10 flex items-center justify-center">
          <Users class="w-8 h-8 text-slate-700" />
        </div>
        <p class="text-sm text-slate-500 italic font-medium">Select a channel to view members</p>
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
