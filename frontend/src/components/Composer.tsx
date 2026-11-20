import { useLanguage } from '../i18n/LanguageProvider'
import { useCallback, useEffect, useRef, useState, type ClipboardEvent, type FocusEvent, type KeyboardEvent } from 'react'

interface ComposerProps {
  sending: boolean
  maxMB: number
  onSend: (content: string, files: File[]) => Promise<boolean>
  onStop: () => void
}

function formatSize(size: number): string {
  if (size >= 1024 * 1024) return `${(size / (1024 * 1024)).toFixed(1)} MB`
  return `${Math.max(1, Math.round(size / 1024))} KB`
}

function PendingAttachment({ file, onRemove }: { file: File; onRemove: () => void }) {
  const { t } = useLanguage()

  return (
    <div className="smoi-pending-file">
      <i className="bi bi-paperclip" aria-hidden="true" />
      <span>{file.name}</span>
      <small>{formatSize(file.size)}</small>
      <button
        className="smoi-remove"
        type="button"
        aria-label={t('removeAttachment', { name: file.name })}
        onClick={onRemove}
      >
        <i className="bi bi-x-lg" aria-hidden="true" />
      </button>
    </div>
  )
}

export default function Composer({ sending, maxMB, onSend, onStop }: ComposerProps) {
  const { t } = useLanguage()
  const [text, setText] = useState('')
  const [files, setFiles] = useState<File[]>([])
  const [error, setError] = useState('')
  const [isDraggingFiles, setIsDraggingFiles] = useState(false)
  const [isFocused, setIsFocused] = useState(false)
  const fileInput = useRef<HTMLInputElement>(null)
  const dragDepth = useRef(0)
  const textarea = useRef<HTMLTextAreaElement>(null)
  const expanded = isFocused || Boolean(text.trim()) || files.length > 0 || sending

  const addFiles = useCallback((incoming: File[]) => {
    const oversized = incoming.find(file => file.size > maxMB * 1024 * 1024)

    if (oversized) {
      setError(t('fileLimit', { name: oversized.name, limit: maxMB }))
      return
    }

    setError('')
    setFiles(previous => [...previous, ...incoming].slice(0, 20))
  }, [maxMB, t])

  useEffect(() => {
    function hasFiles(event: DragEvent) {
      return Array.from(event.dataTransfer?.types || []).includes('Files')
    }

    function handleDragEnter(event: DragEvent) {
      if (!hasFiles(event)) return
      event.preventDefault()
      dragDepth.current += 1
      setIsDraggingFiles(true)
    }

    function handleDragOver(event: DragEvent) {
      if (!hasFiles(event)) return
      event.preventDefault()
      if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
    }

    function handleDragLeave(event: DragEvent) {
      if (!event.relatedTarget) {
        dragDepth.current = 0
      } else {
        dragDepth.current = Math.max(0, dragDepth.current - 1)
      }
      if (dragDepth.current === 0) setIsDraggingFiles(false)
    }

    function handleDrop(event: DragEvent) {
      if (!hasFiles(event)) return
      event.preventDefault()
      dragDepth.current = 0
      setIsDraggingFiles(false)
      addFiles(Array.from(event.dataTransfer?.files || []))
    }

    window.addEventListener('dragenter', handleDragEnter)
    window.addEventListener('dragover', handleDragOver)
    window.addEventListener('dragleave', handleDragLeave)
    window.addEventListener('drop', handleDrop)

    return () => {
      window.removeEventListener('dragenter', handleDragEnter)
      window.removeEventListener('dragover', handleDragOver)
      window.removeEventListener('dragleave', handleDragLeave)
      window.removeEventListener('drop', handleDrop)
    }
  }, [addFiles])

  useEffect(() => {
    const element = textarea.current
    if (!element) return
    element.style.height = 'auto'
    element.style.height = `${Math.min(element.scrollHeight, 192)}px`
  }, [text])

  async function sendMessage() {
    if (!text.trim() && files.length === 0) return

    const draft = text.trim()
    const attachments = files
    setText('')
    setFiles([])
    setError('')

    const sent = await onSend(draft, attachments)
    if (!sent) {
      setText(previous => previous || draft)
      setFiles(previous => previous.length > 0 ? previous : attachments)
    }
  }

  function handleKeyDown(event: KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key !== 'Enter' || event.shiftKey || event.nativeEvent.isComposing) return

    event.preventDefault()
    void sendMessage()
  }

  function handlePaste(event: ClipboardEvent<HTMLTextAreaElement>) {
    const images = Array.from(event.clipboardData.files).filter(file => file.type.startsWith('image/'))
    if (images.length === 0) return
    event.preventDefault()
    addFiles(images)
  }

  function handleBlur(event: FocusEvent<HTMLDivElement>) {
    if (!event.currentTarget.contains(event.relatedTarget as Node | null)) setIsFocused(false)
  }

  return (
    <>
      {isDraggingFiles && (
        <div className="smoi-file-drop-overlay" role="status" aria-live="polite">
          <div className="smoi-file-drop-prompt">
            <i className="bi bi-cloud-arrow-up" aria-hidden="true" />
            <span>{t('dropFiles')}</span>
          </div>
        </div>
      )}

      <div className="smoi-composer-area">
        <span className="visually-hidden" role="status" aria-live="polite">
          {files.length === 0
            ? t('pendingFilesNone')
            : files.length === 1
              ? t('pendingFilesOne')
              : t('pendingFilesMany', { count: files.length })}
        </span>
        <div
          className={`smoi-composer ${expanded ? 'is-active' : 'is-idle'}`}
          onFocusCapture={() => setIsFocused(true)}
          onBlurCapture={handleBlur}
        >
          {expanded && (
            <>
              {files.length > 0 && (
                <div className="smoi-pending-files">
                  {files.map((file, index) => (
                    <PendingAttachment
                      key={`${file.name}-${index}`}
                      file={file}
                      onRemove={() => setFiles(previous => previous.filter((_, current) => current !== index))}
                    />
                  ))}
                </div>
              )}

              <div className="smoi-composer-toolbar">
                <input
                  aria-label={t('chooseFiles')}
                  type="file"
                  hidden
                  ref={fileInput}
                  multiple
                  onChange={event => {
                    addFiles(Array.from(event.target.files || []))
                    event.target.value = ''
                  }}
                />
                <button
                  className="smoi-icon-button"
                  type="button"
                  title={t('attachFiles')}
                  aria-label={t('attachFiles')}
                  onClick={() => fileInput.current?.click()}
                >
                  <i className="bi bi-plus-lg" aria-hidden="true" />
                </button>
                <div className="smoi-send-actions">
                  {sending && (
                    <button
                      className="smoi-stop-button"
                      type="button"
                      aria-label={t('stopResponse')}
                      title={t('stopResponse')}
                      onClick={onStop}
                    >
                      <i className="bi bi-stop-fill" aria-hidden="true" />
                    </button>
                  )}
                  <button
                    className="smoi-send-button"
                    type="button"
                    aria-label={t(sending ? 'interruptAndSend' : 'sendMessage')}
                    title={t(sending ? 'interruptAndSend' : 'sendMessage')}
                    disabled={!text.trim() && files.length === 0}
                    onClick={() => void sendMessage()}
                  >
                    <i className="bi bi-arrow-up" aria-hidden="true" />
                  </button>
                </div>
              </div>
            </>
          )}

          <label className="visually-hidden" htmlFor="smoi-message-input">
            {t('messageToSmoi')}
          </label>
          <textarea
            ref={textarea}
            id="smoi-message-input"
            aria-describedby={error ? 'smoi-composer-error' : undefined}
            aria-invalid={Boolean(error)}
            className="form-control smoi-textarea"
            rows={1}
            placeholder={t('messagePlaceholder')}
            value={text}
            onChange={event => setText(event.target.value)}
            onKeyDown={handleKeyDown}
            onPaste={handlePaste}
          />
        </div>

        {error && (
          <div id="smoi-composer-error" className="smoi-composer-error" role="alert">
            {error}
          </div>
        )}
      </div>
    </>
  )
}
