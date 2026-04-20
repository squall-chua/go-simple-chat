import { ref } from 'vue'

export interface Channel {
  id: string
  name: string
  type: 'TYPE_DIRECT' | 'TYPE_GROUP' | 'direct' | 'group'
  participants: string[]
  participant_usernames: string[]
  last_read_id?: string
  last_message_id?: string
  unread_count: number
}

export interface Message {
  message_id: string
  channel_id: string
  sender_id: string
  sender_username: string
  content: string
  medias?: {
    type: string
    url: string
    name: string
  }[]
  created_at: string
}

// Global Singletons
export const channels = ref<Channel[]>([])
export const activeChannelId = ref<string | null>(null)
export const messages = ref<Message[]>([])
export const isLoadingMessages = ref(false)
