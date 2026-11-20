import { translate } from '../i18n/LanguageProvider'
import { useEffect, useRef, useState } from 'react'
import { request, upload } from '../services/api'
import { conversationFromPath, conversationPath, navigate } from '../services/navigation'
import type { Attachment, Conversation, GenerationJob, Message } from '../types'
import { errorMessage } from './useSession'

interface ChatDependencies {
  createConversation: (model: string) => Promise<Conversation>
  reloadConversations: () => Promise<void> | void
  onError: (message: string) => void
  enabled: boolean
}

interface ConversationState {
  data: Message[]
  jobs: GenerationJob[]
}

export function useChat({ createConversation, reloadConversations, onError, enabled }: ChatDependencies) {
  const [selectedId, setSelectedId] = useState(conversationFromPath)
  const [messages, setMessages] = useState<Message[]>([])
  const [jobs, setJobs] = useState<GenerationJob[]>([])
  const selectedIdRef = useRef(selectedId)
  const submissionChain = useRef<Promise<unknown>>(Promise.resolve())
  const activeJobs = jobs.filter(job => job.status === 'running' || job.status === 'pending')
  const isSending = activeJobs.length > 0

  async function loadMessages(conversationId: string) {
    const state = await request<ConversationState>(`/conversations/${conversationId}/messages`)

    if (selectedIdRef.current === conversationId) {
      setMessages(state.data)
      setJobs(state.jobs)
    }

    return state
  }

  useEffect(() => {
    selectedIdRef.current = selectedId

    if (!selectedId || !enabled) {
      setMessages([])
      setJobs([])
      return
    }

    let active = true
    let timer: ReturnType<typeof setTimeout> | undefined

    async function poll() {
      let nextDelay = 2500

      try {
        const state = await request<ConversationState>(`/conversations/${selectedId}/messages`)
        if (!active || selectedIdRef.current !== selectedId) return

        setMessages(state.data)
        setJobs(state.jobs)

        const busy = state.jobs.some(job => job.status === 'pending' || job.status === 'running')
        nextDelay = busy ? 500 : 2500
      } catch (cause) {
        if (active && selectedIdRef.current === selectedId) {
          onError(errorMessage(cause, translate('unableToLoadConversation')))
        }
        nextDelay = 4000
      } finally {
        if (active) {
          timer = setTimeout(poll, document.hidden ? 3000 : nextDelay)
        }
      }
    }

    function refreshWhenVisible() {
      if (!document.hidden) {
        clearTimeout(timer)
        void poll()
      }
    }

    void poll()
    document.addEventListener('visibilitychange', refreshWhenVisible)

    return () => {
      active = false
      clearTimeout(timer)
      document.removeEventListener('visibilitychange', refreshWhenVisible)
    }
  }, [selectedId, enabled])

  useEffect(() => {
    function handleHistoryChange() {
      if (window.location.pathname === '/admin') return

      const id = conversationFromPath()
      if (selectedIdRef.current !== id) selectConversation(id)
    }

    window.addEventListener('popstate', handleHistoryChange)
    return () => window.removeEventListener('popstate', handleHistoryChange)
  }, [])

  function selectConversation(id: string) {
    if (selectedIdRef.current === id) return

    selectedIdRef.current = id
    setSelectedId(id)
    setMessages([])
    setJobs([])
  }

  function startNewConversation() {
    selectConversation('')
  }

  async function stopGeneration() {
    if (!selectedIdRef.current) return

    try {
      await request(`/conversations/${selectedIdRef.current}/stop`, { method: 'POST' })
      await loadMessages(selectedIdRef.current)
    } catch (cause) {
      onError(errorMessage(cause, translate('couldNotStopGeneration')))
    }
  }

  async function submit(content: string, files: File[], model: string): Promise<boolean> {
    let conversationId = selectedIdRef.current

    try {
      if (!conversationId) {
        const created = await createConversation(model)
        conversationId = created.id
        selectedIdRef.current = conversationId
        setSelectedId(conversationId)
        navigate(conversationPath(conversationId))
      }

      const attachments: Attachment[] = []
      for (const file of files) {
        attachments.push(await upload(conversationId, file))
      }

      await request(`/conversations/${conversationId}/messages`, {
        method: 'POST',
        body: JSON.stringify({
          content,
          attachment_ids: attachments.map(attachment => attachment.id),
          request_id: crypto.randomUUID()
        })
      })

      try {
        await loadMessages(conversationId)
      } catch (cause) {
        onError(errorMessage(cause, translate('messageSavedNoRefresh')))
      }
      void reloadConversations()
      return true
    } catch (cause) {
      onError(errorMessage(cause, translate('unableToSendMessage')))
      return false
    }
  }

  function send(content: string, files: File[], model: string): Promise<boolean> {
    const task = submissionChain.current.then(() => submit(content, files, model))
    submissionChain.current = task.then(() => undefined, () => undefined)
    return task
  }

  return {
    selectedId,
    messages,
    jobs,
    isSending,
    selectConversation,
    startNewConversation,
    stopGeneration,
    send
  }
}
