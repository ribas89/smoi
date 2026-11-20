import { useLanguage } from '../../i18n/LanguageProvider'
import { useEffect, useRef, useState } from 'react'
import type { GenerationJob, Message } from '../../types'
import Composer from '../Composer'
import MessageView from '../MessageView'

interface ChatPanelProps {
  selectedId: string
  messages: Message[]
  jobs: GenerationJob[]
  isSending: boolean
  maxUploadMB: number
  error: string
  onDismissError: () => void
  onSend: (content: string, files: File[]) => Promise<boolean>
  onStop: () => void
}

export default function ChatPanel({
  selectedId, messages, jobs, isSending, maxUploadMB,
  error, onDismissError, onSend, onStop
}: ChatPanelProps) {
  const { t } = useLanguage()
  const bottom = useRef<HTMLDivElement>(null)
  const scrollArea = useRef<HTMLDivElement>(null)
  const shouldAutoScroll = useRef(true)
  const wasSending = useRef(false)
  const [announcement, setAnnouncement] = useState('')

  useEffect(() => {
    shouldAutoScroll.current = true
  }, [selectedId])

  useEffect(() => {
    if (!shouldAutoScroll.current) return

    bottom.current?.scrollIntoView({
      behavior: isSending || window.matchMedia('(prefers-reduced-motion: reduce)').matches
        ? 'auto'
        : 'smooth',
      block: 'end'
    })
  }, [messages, isSending])

  useEffect(() => {
    if (isSending) {
      setAnnouncement(t('isResponding'))
    } else if (wasSending.current) {
      setAnnouncement(t('finishedResponding'))
    }

    wasSending.current = isSending
  }, [isSending, t])

  function handleScroll() {
    const area = scrollArea.current
    if (!area) return

    shouldAutoScroll.current = area.scrollHeight - area.scrollTop - area.clientHeight < area.clientHeight / 8
  }

  const jobsByMessage = new Map(jobs.map(job => [job.assistant_message_id, job]))

  return (
    <>
      <div
        ref={scrollArea}
        className="smoi-message-scroll"
        role="region"
        aria-label={t('conversationContent')}
        onScroll={handleScroll}
      >
        {error && (
          <div className="alert alert-danger alert-dismissible smoi-error-banner" role="alert">
            {error}
            <button
              className="btn-close"
              type="button"
              aria-label={t('dismissError')}
              onClick={onDismissError}
            />
          </div>
        )}

        {messages.length === 0 ? (
          <div className="smoi-empty-state">
            <img className="smoi-welcome-icon" src="/assets/smoi.svg" alt={t('smoiLogo')} />
            <h2>{t('greeting')}</h2>
            <p>{t('greetingDescription')}</p>
          </div>
        ) : (
          <div
            className="smoi-message-list"
            role="log"
            aria-label={t('conversationMessages')}
            aria-live="off"
            aria-relevant="additions"
          >
            {messages.map(message => (
              <MessageView
                key={message.id}
                message={message}
                job={jobsByMessage.get(message.id)}
              />
            ))}
          </div>
        )}

        <div ref={bottom} />
      </div>

      <span className="visually-hidden" role="status" aria-live="polite" aria-atomic="true">
        {announcement}
      </span>

      <Composer
        sending={isSending}
        maxMB={maxUploadMB}
        onSend={onSend}
        onStop={onStop}
      />
    </>
  )
}
