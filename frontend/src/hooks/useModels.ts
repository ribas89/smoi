import { translate } from '../i18n/LanguageProvider'
import { useEffect, useState } from 'react'
import { request } from '../services/api'
import type { Conversation, User } from '../types'
import { errorMessage } from './useSession'

interface ModelResponse {
  data: { id: string }[]
  default_model: string
}

interface ConfigResponse {
  max_upload_mb: number
  default_model: string
}

function savedModel(): string {
  try {
    return localStorage.getItem('smoi-model') || ''
  } catch {
    return ''
  }
}

export function useModels(
  user: User | null,
  conversation: Conversation | undefined,
  onError: (message: string) => void
) {
  const [models, setModels] = useState<string[]>([])
  const [model, setModel] = useState(savedModel)
  const [maxUploadMB, setMaxUploadMB] = useState(25)

  useEffect(() => {
    if (!user) return

    async function loadModels() {
      try {
        const response = await request<ModelResponse>('/models')
        const available = response.data.map(item => item.id).filter(Boolean)

        setModels(available)
        setModel(previous => previous || response.default_model || available[0] || '')
      } catch (cause) {
        onError(errorMessage(cause, translate('couldNotLoadModels')))
      }
    }

    async function loadConfig() {
      try {
        const config = await request<ConfigResponse>('/config')
        setMaxUploadMB(config.max_upload_mb)
        setModel(previous => previous || config.default_model)
      } catch {
        return
      }
    }

    void loadModels()
    void loadConfig()
  }, [user, onError])

  useEffect(() => {
    if (conversation) {
      setModel(conversation.model)
      try {
        localStorage.setItem('smoi-model', conversation.model)
      } catch {
        return
      }
    }
  }, [conversation?.id, conversation?.model])

  const availableModels = Array.from(new Set([model, ...models].filter(Boolean)))

  return { model, setModel, availableModels, maxUploadMB }
}
