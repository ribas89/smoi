export interface Page<T> {
  data: T[]
  has_more?: boolean
}

export interface User {
  id: string
  username: string
  email: string
  display_name: string
  avatar_url: string
  role: 'admin' | 'user'
}

export interface Conversation {
  id: string
  user_id: string
  title: string
  model: string
  created_at: number
  updated_at: number
}

export interface Attachment {
  id: string
  user_id: string
  conversation_id: string
  filename: string
  mime_type: string
  size: number
  created_at: number
}

export interface Message {
  id: string
  conversation_id: string
  role: 'user' | 'assistant' | 'system'
  content: string
  created_at: number
  attachments: Attachment[]
}

export interface AdminStats {
  users: number
  conversations: number
  storage_bytes: number
}

export interface AdminUser {
  id: string
  username: string
  email: string
  display_name: string
  role: string
  conversation_count: number
  storage_bytes: number
}

export interface AdminConversation {
  id: string
  user_id: string
  owner: string
  title: string
  storage_bytes: number
  updated_at: number
}

export interface GenerationJob {
  id: string
  conversation_id: string
  user_message_id: string
  assistant_message_id: string
  status: 'pending' | 'running' | 'completed' | 'stopped' | 'failed'
  error: string
  created_at: number
  updated_at: number
}
