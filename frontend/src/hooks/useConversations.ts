import { translate } from '../i18n/LanguageProvider'
import { useCallback, useEffect, useState } from 'react'
import { request } from '../services/api'
import type { Conversation, Page, User } from '../types'
import { errorMessage } from './useSession'

type ConversationUpdate = Partial<Pick<Conversation, 'title' | 'model'>>

export function useConversations(user: User | null, onError: (message: string) => void) {
  const [conversations, setConversations] = useState<Conversation[]>([])
  const [search, setSearch] = useState('')
  const [hasMore, setHasMore] = useState(false)

  const load = useCallback(async (query: string, offset = 0) => {
    if (!user) return

    try {
      const params = new URLSearchParams({ q: query, offset: String(offset) })
      const response = await request<Page<Conversation>>(`/conversations?${params}`)

      setConversations(previous =>
        offset > 0 ? [...previous, ...response.data] : response.data
      )
      setHasMore(Boolean(response.has_more))
    } catch (cause) {
      onError(errorMessage(cause, translate('couldNotLoadConversations')))
    }
  }, [user, onError])

  useEffect(() => {
    if (!user) {
      setConversations([])
      setHasMore(false)
      return
    }

    void load(search)
  }, [user, search, load])

  async function create(model: string): Promise<Conversation> {
    const conversation = await request<Conversation>('/conversations', {
      method: 'POST',
      body: JSON.stringify({ model })
    })

    setConversations(previous => [conversation, ...previous])
    return conversation
  }

  async function update(id: string, changes: ConversationUpdate) {
    const updated = await request<Conversation>(`/conversations/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(changes)
    })

    setConversations(previous =>
      previous.map(item => item.id === updated.id ? updated : item)
    )
  }

  async function remove(id: string) {
    await request(`/conversations/${id}`, { method: 'DELETE' })
    setConversations(previous => previous.filter(item => item.id !== id))
  }

  return {
    conversations,
    hasMore,
    search,
    setSearch,
    loadMore: () => load(search, conversations.length),
    reload: () => load(search),
    create,
    update,
    remove
  }
}
