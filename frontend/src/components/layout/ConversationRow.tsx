import { useLanguage } from '../../i18n/LanguageProvider'
import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import type { Conversation } from '../../types'
import type { ConversationProject } from '../../hooks/useConversationOrganization'
import { conversationPath } from '../../services/navigation'

interface ConversationRowProps {
  conversation: Conversation
  active: boolean
  pinned: boolean
  projects: ConversationProject[]
  projectId: string
  onSelect: (id: string) => void
  onRename: (id: string, title: string) => Promise<void>
  onDelete: (conversation: Conversation) => void
  onTogglePin: (id: string) => void
  onAssignProject: (conversationId: string, projectId: string) => void
}

export default function ConversationRow({
  conversation,
  active,
  pinned,
  projects,
  projectId,
  onSelect,
  onRename,
  onDelete,
  onTogglePin,
  onAssignProject
}: ConversationRowProps) {
  const { t } = useLanguage()
  const [isEditing, setIsEditing] = useState(false)
  const [title, setTitle] = useState(conversation.title)
  const [isMenuOpen, setIsMenuOpen] = useState(false)
  const [projectFeedback, setProjectFeedback] = useState('')
  const dialogRef = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    const dialog = dialogRef.current
    if (!dialog) return

    if (isMenuOpen && !dialog.open) dialog.showModal()
    if (!isMenuOpen && dialog.open) dialog.close()
  }, [isMenuOpen])

  function resetMenu() {
    setIsMenuOpen(false)
    setProjectFeedback('')
  }

  function closeMenu() {
    if (dialogRef.current?.open) dialogRef.current.close()
    resetMenu()
  }

  async function saveTitle() {
    setIsEditing(false)
    const updatedTitle = title.trim()

    if (updatedTitle && updatedTitle !== conversation.title) {
      await onRename(conversation.id, updatedTitle)
    }
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Escape') {
      setTitle(conversation.title)
      setIsEditing(false)
    }

    if (event.key === 'Enter') event.currentTarget.blur()
  }

  return (
    <div className={`smoi-chat-row ${active ? 'active' : ''}`}>
      {isEditing ? (
        <input
          className="form-control form-control-sm smoi-rename-input"
          autoFocus
          aria-label={t('renameConversation', { title: conversation.title })}
          value={title}
          onChange={event => setTitle(event.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={() => void saveTitle()}
        />
      ) : (
        <a
          className="smoi-chat-name"
          title={conversation.title}
          href={conversationPath(conversation.id)}
          aria-current={active ? 'page' : undefined}
          onClick={event => {
            if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return
            event.preventDefault()
            onSelect(conversation.id)
          }}
        >
          <i className="bi bi-chat-left" aria-hidden="true" />
          <span>{conversation.title}</span>
        </a>
      )}

      <button
        className="smoi-row-action"
        type="button"
        aria-label={t('moreActions')}
        aria-haspopup="dialog"
        aria-expanded={isMenuOpen}
        title={t('moreActions')}
        onClick={() => setIsMenuOpen(true)}
      >
        <i className="bi bi-three-dots" aria-hidden="true" />
      </button>

      <dialog
        ref={dialogRef}
        className="smoi-conversation-dialog"
        aria-labelledby={`conversation-actions-${conversation.id}`}
        onCancel={resetMenu}
        onClose={resetMenu}
        onClick={event => {
          if (event.target === event.currentTarget) closeMenu()
        }}
      >
        <div className="smoi-conversation-dialog-header">
          <h2 id={`conversation-actions-${conversation.id}`}>{t('conversationActions')}</h2>
          <button
            className="smoi-dialog-close"
            type="button"
            aria-label={t('cancel')}
            onClick={closeMenu}
          >
            <i className="bi bi-x-lg" aria-hidden="true" />
          </button>
        </div>

        <div className="smoi-conversation-dialog-actions">
          <button type="button" onClick={() => {
            onTogglePin(conversation.id)
            closeMenu()
          }}>
            <i className={`bi bi-pin${pinned ? '-fill' : ''}`} aria-hidden="true" />
            {t(pinned ? 'unpinConversation' : 'pinConversation')}
          </button>

          <label className="smoi-row-project-select">
            <span>{t('moveToProject')}</span>
            <select
              className="form-select form-select-sm"
              aria-label={t('moveToProject')}
              value={projectId}
              onChange={event => {
                const nextProjectId = event.target.value
                const nextProject = projects.find(project => project.id === nextProjectId)
                onAssignProject(conversation.id, nextProjectId)
                setProjectFeedback(nextProject
                  ? t('conversationMovedToProject', { project: nextProject.name })
                  : t('conversationRemovedFromProject'))
              }}
            >
              <option value="">{t('noProject')}</option>
              {projects.map(project => (
                <option key={project.id} value={project.id}>{project.name}</option>
              ))}
            </select>
          </label>

          {projectFeedback && (
            <p className="smoi-conversation-dialog-feedback" role="status" aria-live="polite">
              <i className="bi bi-check-circle" aria-hidden="true" />
              {projectFeedback}
            </p>
          )}

          <button type="button" onClick={() => {
            setTitle(conversation.title)
            setIsEditing(true)
            closeMenu()
          }}>
            <i className="bi bi-pencil" aria-hidden="true" />
            {t('rename')}
          </button>

          <button type="button" className="is-danger" onClick={() => {
            closeMenu()
            onDelete(conversation)
          }}>
            <i className="bi bi-trash" aria-hidden="true" />
            {t('delete')}
          </button>
        </div>

        <div className="smoi-conversation-dialog-footer">
          <button type="button" onClick={closeMenu}>{t('done')}</button>
        </div>
      </dialog>
    </div>
  )
}
