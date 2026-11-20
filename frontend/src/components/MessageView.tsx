import { useLanguage } from '../i18n/LanguageProvider'
import { useState } from 'react'
import type { GenerationJob, Message } from '../types'
import AttachmentCard from './chat/AttachmentCard'
import MarkdownContent from './chat/MarkdownContent'

interface MessageViewProps {
  message: Message
  job?: GenerationJob
}

export default function MessageView({ message, job }: MessageViewProps) {
  const { t } = useLanguage()
  const streaming = job?.status === 'running'
  const pending = job?.status === 'pending'
  const failed = job?.status === 'failed'
  const stopped = job?.status === 'stopped'
  const empty = job?.status === 'completed' && !message.content
  const isAssistant = message.role === 'assistant'
  const messageClass = isAssistant ? 'smoi-message-assistant' : 'smoi-message-user'
  const [copied, setCopied] = useState(false)

  async function copyMessage() {
    try {
      if (navigator.clipboard?.writeText) {
        await navigator.clipboard.writeText(message.content)
      } else {
        throw new Error('Clipboard API unavailable')
      }
    } catch {
      const temporary = document.createElement('textarea')
      temporary.value = message.content
      temporary.setAttribute('readonly', '')
      temporary.style.position = 'fixed'
      temporary.style.opacity = '0'
      document.body.appendChild(temporary)
      temporary.select()
      const copied = document.execCommand('copy')
      temporary.remove()
      if (!copied) return
    }

    setCopied(true)
    window.setTimeout(() => setCopied(false), 1800)
  }

  return (
    <article
      className={`smoi-message ${messageClass}`}
      aria-label={t(isAssistant ? 'assistantMessage' : 'yourMessage')}
      aria-busy={streaming}
    >
      <div className="smoi-message-body">
        {!isAssistant && <div className="smoi-message-label">{t('you')}</div>}

        {message.content && (
          isAssistant ? (
            <MarkdownContent content={message.content} />
          ) : (
            <div className="smoi-user-text">{message.content}</div>
          )
        )}

        {message.attachments.length > 0 && (
          <div className="smoi-attachments">
            {message.attachments.map(file => (
              <AttachmentCard key={file.id} attachment={file} />
            ))}
          </div>
        )}

        {pending && (
          <span className="smoi-generation-state" role="status">
            <i className="bi bi-hourglass-split" aria-hidden="true" /> {t('starting')}
          </span>
        )}
        {streaming && !message.content && (
          <span className="smoi-generation-state" role="status">
            <span className="spinner-grow spinner-grow-sm" aria-hidden="true" /> {t('responding')}
          </span>
        )}
        {failed && (
          <span className="smoi-generation-state text-danger" role="alert">
            <i className="bi bi-exclamation-circle" aria-hidden="true" /> {job?.error || t('generationFailed')}
          </span>
        )}
        {stopped && (
          <span className="smoi-generation-state" role="status">
            {t(job?.error === 'Interrupted by a newer message' ? 'responseInterrupted' : 'responseStopped')}
          </span>
        )}
        {empty && <span className="smoi-generation-state" role="status">{t('emptyResponse')}</span>}
        {isAssistant && streaming && <span className="smoi-caret" aria-hidden="true" />}

        <button
            className="smoi-copy-message"
            type="button"
            title={t('copyMessage')}
            aria-label={t('copyMessage')}
            onClick={() => void copyMessage()}
          >
            <i className={`bi bi-${copied ? 'check2' : 'copy'}`} aria-hidden="true" />
            <span>{t(copied ? 'copied' : 'copy')}</span>
        </button>

        <span className="visually-hidden" role="status" aria-live="polite" aria-atomic="true">
          {copied ? t('messageCopied') : ''}
        </span>
      </div>
    </article>
  )
}
