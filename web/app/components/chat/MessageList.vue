<script setup lang="ts">
import { ref, onUpdated, onMounted, watch } from 'vue'
import { Hash, MessageSquare, Clock } from 'lucide-vue-next'

const { messages, isLoading } = useMessages()
const { userId, username } = useAuth()
const { activeChannel, markAsRead } = useChannels()

const getDisplayName = (channel: any) => {
  if (!channel) return ''
  const isDirect = channel.type === 'TYPE_DIRECT' || channel.type === 'direct'
  if (isDirect) {
    const other = channel.participant_usernames.find((u: string) => u !== username.value)
    return other || channel.name || 'Private Chat'
  }
  return channel.name
}

const scrollContainer = ref<HTMLElement | null>(null)
const initialReadId = ref<string | null>(null)

// Capture initial read state when channel changes
watch(() => activeChannel.value?.id, (newId) => {
  if (newId) {
    initialReadId.value = activeChannel.value?.last_read_id || null
  } else {
    initialReadId.value = null
  }
}, { immediate: true })

const isFirstUnread = (index: number) => {
  if (!initialReadId.value || !messages.value[index]) return false
  
  const lastReadIndex = messages.value.findIndex(m => m.message_id === initialReadId.value)
  
  // Case 1: Marker is within the current visible list
  if (lastReadIndex !== -1) {
    return index === lastReadIndex + 1
  }
  
  // Case 2: Marker is NOT in the visible list (history is further back)
  // If we couldn't find the last read ID in the current batch, but we have messages,
  // it means all these current messages are likely new.
  return index === 0 && messages.value[0] && messages.value[0].sender_id !== userId.value
}

// Watch for own messages to clear the "New Messages" marker
watch(messages, (newMsgs) => {
  const lastMsg = newMsgs[newMsgs.length - 1]
  if (lastMsg && lastMsg.sender_id === userId.value) {
    initialReadId.value = null
  }
}, { deep: true })

const smartScroll = () => {
  if (!scrollContainer.value) return

  // 1. Try to find the unread marker
  const marker = scrollContainer.value.querySelector('#unread-marker')
  if (marker && initialReadId.value) {
    marker.scrollIntoView({ block: 'center' })
    return
  }

  // 2. Default: scroll to bottom
  scrollContainer.value.scrollTop = scrollContainer.value.scrollHeight
}

watch(messages, () => {
  setTimeout(smartScroll, 50)
}, { deep: true })

onMounted(smartScroll)

const isOnlyEmoji = (content: string) => {
  if (!content) return false
  // Improved regex for emojis including newer ones and variations
  const emojiOnlyRegex = /^(\s*[\u{1F300}-\u{1F9FF}\u{2600}-\u{26FF}\u{2700}-\u{27BF}\u{1F1E6}-\u{1F1FF}\u{1F191}-\u{1F251}\u{1F004}\u{1F0CF}\u{1F170}-\u{1F171}\u{1F17E}-\u{1F17F}\u{1F18E}\u{3030}\u{2B50}\u{2B55}\u{2934}-\u{2935}\u{2B05}-\u{2B07}\u{2B1B}-\u{2B1C}\u{3297}\u{3299}\u{303D}\u{00A9}\u{00AE}\u{2122}\u{23F3}-\u{23F9}\u{24C2}\u{23E9}-\u{23EF}\u{25B6}\u{23F8}-\u{23FA}\u{1F600}-\u{1F64F}\u{1F680}-\u{1F6FF}\u{2B50}-\u{2B55}\u{1F004}-\u{1F0CF}]\s*)+$/u
  return emojiOnlyRegex.test(content)
}

const formatTime = (timestamp: string) => {
  if (!timestamp) return ''
  const date = new Date(timestamp)
  const now = new Date()
  
  const isToday = date.toDateString() === now.toDateString()
  
  const timeStr = date.toLocaleTimeString([], { 
    hour: '2-digit', 
    minute: '2-digit',
    hour12: true 
  })
  
  if (isToday) return timeStr
  
  const dateStr = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
  
  return `${dateStr} ${timeStr}`
}

const isSameSender = (index: number) => {
  if (index === 0 || !messages.value[index] || !messages.value[index - 1]) return false
  return messages.value[index]?.sender_id === messages.value[index - 1]?.sender_id
}
</script>

<template>
  <div class="flex-1 flex flex-col min-w-0 min-h-0 bg-black">
    <!-- Channel Header -->
    <header class="h-16 border-b border-white/10 flex items-center px-6 justify-between bg-black backdrop-blur-md sticky top-0 z-10">
      <div class="flex items-center gap-3 overflow-hidden" v-if="activeChannel">
        <component 
          :is="(activeChannel.type === 'TYPE_DIRECT' || activeChannel.type === 'direct') ? MessageSquare : Hash" 
          class="w-6 h-6 text-sky-400 shrink-0"
        />
        <h2 class="text-lg font-bold text-white truncate tracking-tight">{{ getDisplayName(activeChannel) }}</h2>
        <div class="hidden sm:flex items-center gap-2 ml-4 px-2 py-0.5 rounded-full bg-white/10 border border-white/20">
          <div class="w-1.5 h-1.5 rounded-full bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.8)]"></div>
          <span class="text-[10px] text-slate-200 font-bold uppercase tracking-tight">{{ activeChannel.participant_usernames.length }} Members</span>
        </div>
      </div>
      <div v-else class="text-slate-500 italic text-sm font-medium">No channel selected</div>
      
      <Clock class="w-5 h-5 text-slate-400 hover:text-white cursor-help transition-colors" />
    </header>

    <!-- Messages Area -->
    <div 
      ref="scrollContainer"
      class="flex-1 overflow-y-auto p-4 space-y-3 custom-scrollbar"
    >
      <div v-if="isLoading" class="flex flex-col items-center justify-center h-full gap-4 text-slate-500 animate-pulse">
        <div class="w-10 h-10 border-2 border-slate-700 border-t-sky-500 rounded-full animate-spin"></div>
        <p class="text-xs uppercase tracking-widest font-bold">Synchronizing History</p>
      </div>

      <template v-else-if="messages.length > 0">
        <template v-for="(msg, index) in messages" :key="msg.message_id">
          <!-- Unread Separator -->
          <div v-if="isFirstUnread(index)" id="unread-marker" class="flex items-center gap-4 py-6">
            <div class="flex-1 h-px bg-red-500/50"></div>
            <span class="text-[10px] font-black text-red-500 uppercase tracking-[0.3em] px-3 py-1 rounded-full bg-red-500/10 border border-red-500/30">New Messages</span>
            <div class="flex-1 h-px bg-red-500/50"></div>
          </div>

          <div 
            :class="[
            'group flex flex-col gap-1 px-4 py-2 rounded-xl transition-all hover:bg-white/5',
            msg.sender_id === userId ? 'items-end' : 'items-start'
          ]"
        >
          <!-- Sender Username (only if first message or different sender) -->
          <div 
            v-if="!isSameSender(index)"
            class="flex items-center gap-2 mb-1"
          >
            <span :class="[
              'text-[10px] font-black uppercase tracking-[0.2em]',
              msg.sender_id === userId ? 'text-sky-400' : 'text-slate-400'
            ]">
              {{ msg.sender_id === userId ? 'You' : msg.sender_username }}
            </span>
          </div>

          <!-- Message Content -->
          <div :class="[
            'relative max-w-[85%] break-words flex flex-col gap-2',
            isOnlyEmoji(msg.content) && !msg.medias?.length
              ? 'text-5xl !bg-transparent !shadow-none !px-0 !py-3'
              : [
                  'px-4 py-2.5 pb-1.5 rounded-2xl text-sm leading-relaxed shadow-xl',
                  msg.sender_id === userId 
                    ? 'bg-sky-600 text-white rounded-tr-none border border-sky-400/30' 
                    : 'bg-[#1E293B] text-white rounded-tl-none border border-white/10 shadow-black/50'
                ]
          ]">
            <!-- Media Attachments -->
            <template v-if="msg.medias && msg.medias.length > 0">
              <div v-for="(media, mIdx) in msg.medias" :key="mIdx" class="mb-1">
                <img 
                  v-if="media.type === 'image'" 
                  :src="media.url" 
                  class="rounded-lg max-w-full max-h-80 object-contain shadow-2xl border border-white/5 cursor-zoom-in"
                  loading="lazy"
                />
                <a 
                  v-else 
                  :href="media.url" 
                  target="_blank" 
                  class="flex items-center gap-2 p-3 rounded-lg bg-black/40 hover:bg-black/60 transition-colors text-xs font-bold border border-white/10"
                  :title="media.name"
                >
                  <Paperclip class="w-3.5 h-3.5 shrink-0 text-sky-400" />
                  <span class="truncate max-w-[200px] text-slate-200">{{ media.name || 'Attachment' }}</span>
                </a>
              </div>
            </template>

            <span v-if="msg.content" class="font-medium tracking-wide">{{ msg.content }}</span>
            <div :class="[
              'text-[9px] mt-1 opacity-70 font-bold whitespace-nowrap tracking-wider uppercase',
              msg.sender_id === userId ? 'text-right text-sky-100' : 'text-left text-slate-400',
              isOnlyEmoji(msg.content) ? '!text-slate-500 !text-[10px]' : ''
            ]">
              {{ formatTime(msg.created_at) }}
            </div>
          </div>
        </div>
      </template>
    </template>

      <div v-else-if="activeChannel" class="h-full flex flex-col items-center justify-center text-center p-12 space-y-6">
        <div class="w-20 h-20 rounded-[2.5rem] bg-white/5 border border-white/10 flex items-center justify-center shadow-inner">
          <MessageSquare class="w-10 h-10 text-slate-800" />
        </div>
        <div class="space-y-2">
          <p class="text-slate-200 text-lg font-bold tracking-tight">Beginning of history</p>
          <p class="text-[11px] text-slate-500 uppercase tracking-[0.2em] font-bold">Send a message to start the conversation</p>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.custom-scrollbar::-webkit-scrollbar {
  width: 6px;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 10px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.1);
}
</style>
