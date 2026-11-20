import { useLanguage } from '../../i18n/LanguageProvider'
import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import type { Conversation } from '../../types'
import type { ConversationProject } from '../../hooks/useConversationOrganization'
import ConversationRow from './ConversationRow'

interface SidebarProps {
  conversations: Conversation[]
  hasMore: boolean
  search: string
  selectedId: string
  isOpen: boolean
  isAdminOpen: boolean
  pinnedIds: string[]
  projects: ConversationProject[]
  onClose: () => void
  onNew: () => void
  onSearchChange: (query: string) => void
  onSelect: (id: string) => void
  onRename: (id: string, title: string) => Promise<void>
  onDelete: (conversation: Conversation) => void
  onLoadMore: () => void
  onTogglePin: (id: string) => void
  onCreateProject: (name: string) => void
  onDeleteProject: (id: string) => void
  onAssignProject: (conversationId: string, projectId: string) => void
}

export default function Sidebar({
  conversations,
  hasMore,
  search,
  selectedId,
  isOpen,
  isAdminOpen,
  pinnedIds,
  projects,
  onClose,
  onNew,
  onSearchChange,
  onSelect,
  onRename,
  onDelete,
  onLoadMore,
  onTogglePin,
  onCreateProject,
  onDeleteProject,
  onAssignProject
}: SidebarProps) {
  const { t } = useLanguage()
  const [isCompact, setIsCompact] = useState(() => window.matchMedia('(max-width: 48rem)').matches)
  const [activeProjectId, setActiveProjectId] = useState('')
  const [isCreatingProject, setIsCreatingProject] = useState(false)
  const [newProjectName, setNewProjectName] = useState('')
  const sidebarRef = useRef<HTMLElement>(null)
  const closeButtonRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    const query = window.matchMedia('(max-width: 48rem)')
    const handleChange = () => setIsCompact(query.matches)
    query.addEventListener('change', handleChange)
    return () => query.removeEventListener('change', handleChange)
  }, [])

  useEffect(() => {
    if (!projects.some(project => project.id === activeProjectId)) setActiveProjectId('')
  }, [projects, activeProjectId])

  useEffect(() => {
    if (!isCompact || !isOpen) return

    const previousFocus = document.activeElement as HTMLElement | null
    closeButtonRef.current?.focus()

    return () => previousFocus?.focus()
  }, [isCompact, isOpen])

  function handleNavigationKeys(event: KeyboardEvent<HTMLElement>) {
    if (!isCompact || !isOpen) return

    if (event.key === 'Escape') {
      event.preventDefault()
      onClose()
      return
    }

    if (event.key !== 'Tab') return

    const items = Array.from(sidebarRef.current?.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), summary'
    ) || []).filter(item => item.getClientRects().length > 0)

    if (items.length === 0) return

    const first = items[0]
    const last = items[items.length - 1]

    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault()
      last.focus()
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault()
      first.focus()
    }
  }

  function createProject() {
    if (!newProjectName.trim()) return
    onCreateProject(newProjectName)
    setNewProjectName('')
    setIsCreatingProject(false)
  }

  function deleteProject(project: ConversationProject) {
    if (window.confirm(t('confirmDeleteProject', { name: project.name }))) {
      onDeleteProject(project.id)
      if (activeProjectId === project.id) setActiveProjectId('')
    }
  }

  const pinnedConversations = conversations.filter(conversation => pinnedIds.includes(conversation.id))
  const visibleConversations = activeProjectId
    ? conversations.filter(conversation => projects.find(project => project.id === activeProjectId)?.conversationIds.includes(conversation.id))
    : conversations
  const activeProject = projects.find(project => project.id === activeProjectId)

  function renderConversation(conversation: Conversation) {
    const projectId = projects.find(project => project.conversationIds.includes(conversation.id))?.id || ''

    return (
      <ConversationRow
        key={conversation.id}
        conversation={conversation}
        active={conversation.id === selectedId && !isAdminOpen}
        pinned={pinnedIds.includes(conversation.id)}
        projects={projects}
        projectId={projectId}
        onSelect={onSelect}
        onRename={onRename}
        onDelete={onDelete}
        onTogglePin={onTogglePin}
        onAssignProject={onAssignProject}
      />
    )
  }

  return (
    <>
      {isCompact && isOpen && (
        <button
          type="button"
          className="smoi-mobile-backdrop"
          aria-label={t('closeNavigation')}
          tabIndex={-1}
          onClick={onClose}
        />
      )}

      <aside
        id="smoi-sidebar"
        ref={sidebarRef}
        className={`smoi-sidebar ${isOpen ? 'is-open' : ''}`}
        aria-label={t('conversationNavigation')}
        aria-modal={isCompact && isOpen ? true : undefined}
        role={isCompact && isOpen ? 'dialog' : undefined}
        inert={isCompact && !isOpen}
        onKeyDown={handleNavigationKeys}
      >
        <div className="smoi-sidebar-top">
          <div className="smoi-brand">
            <img className="smoi-brand-symbol" src="/assets/smoi.svg" alt="" />
            <span>SMOI</span>
          </div>
          <button
            ref={closeButtonRef}
            className="smoi-sidebar-mobile-close"
            type="button"
            aria-label={t('closeNavigation')}
            onClick={onClose}
          >
            <i className="bi bi-x-lg" aria-hidden="true" />
          </button>
        </div>

        <button className="smoi-new-chat" type="button" onClick={onNew}>
          <i className="bi bi-plus-lg" aria-hidden="true" />
          <span>{t('newConversation')}</span>
          <i className="bi bi-pencil-square ms-auto" aria-hidden="true" />
        </button>

        <div className="smoi-search">
          <i className="bi bi-search" aria-hidden="true" />
          <label className="visually-hidden" htmlFor="smoi-conversation-search">
            {t('searchConversations')}
          </label>
          <input
            id="smoi-conversation-search"
            type="search"
            placeholder={t('searchConversations')}
            value={search}
            onChange={event => {
              setActiveProjectId('')
              onSearchChange(event.target.value)
            }}
          />
        </div>

        <div className="smoi-sidebar-content">
          {pinnedConversations.length > 0 && (
            <section className="smoi-sidebar-section" aria-labelledby="smoi-pinned-heading">
              <h2 className="smoi-sidebar-heading" id="smoi-pinned-heading">
                <i className="bi bi-pin-angle" aria-hidden="true" />
                {t('pinned')}
              </h2>
              <nav className="smoi-history" aria-label={t('pinned')}>
                {pinnedConversations.map(renderConversation)}
              </nav>
            </section>
          )}

          <section className="smoi-sidebar-section" aria-labelledby="smoi-projects-heading">
            <div className="smoi-sidebar-section-title">
              <h2 className="smoi-sidebar-heading" id="smoi-projects-heading">
                <i className="bi bi-folder2-open" aria-hidden="true" />
                {t('projects')}
              </h2>
              <button
                type="button"
                className="smoi-section-action"
                aria-label={t('createProject')}
                title={t('createProject')}
                onClick={() => setIsCreatingProject(true)}
              >
                <i className="bi bi-plus-lg" aria-hidden="true" />
              </button>
            </div>
            {isCreatingProject && (
              <form
                className="smoi-project-create-form"
                onSubmit={event => {
                  event.preventDefault()
                  createProject()
                }}
              >
                <label className="visually-hidden" htmlFor="smoi-new-project-name">{t('projectNamePrompt')}</label>
                <input
                  id="smoi-new-project-name"
                  className="form-control form-control-sm"
                  autoFocus
                  value={newProjectName}
                  placeholder={t('projectNamePlaceholder')}
                  onChange={event => setNewProjectName(event.target.value)}
                />
                <button className="smoi-section-action" type="submit" aria-label={t('createProject')} disabled={!newProjectName.trim()}>
                  <i className="bi bi-check2" aria-hidden="true" />
                </button>
                <button className="smoi-section-action" type="button" aria-label={t('cancel')} onClick={() => {
                  setIsCreatingProject(false)
                  setNewProjectName('')
                }}>
                  <i className="bi bi-x-lg" aria-hidden="true" />
                </button>
              </form>
            )}
            {projects.length > 0 ? (
              <div className="smoi-project-list">
                {projects.map(project => (
                  <div className={`smoi-project-row ${activeProjectId === project.id ? 'active' : ''}`} key={project.id}>
                    <button
                      type="button"
                      className="smoi-project-select"
                      aria-pressed={activeProjectId === project.id}
                      onClick={() => setActiveProjectId(previous => previous === project.id ? '' : project.id)}
                    >
                      <i className="bi bi-folder" aria-hidden="true" />
                      <span>{project.name}</span>
                      <small>{project.conversationIds.length}</small>
                    </button>
                    <button
                      type="button"
                      className="smoi-section-action smoi-project-delete"
                      aria-label={t('deleteProject', { name: project.name })}
                      title={t('deleteProject', { name: project.name })}
                      onClick={() => deleteProject(project)}
                    >
                      <i className="bi bi-trash" aria-hidden="true" />
                    </button>
                  </div>
                ))}
              </div>
            ) : (
              <p className="smoi-sidebar-empty">{t('noProjects')}</p>
            )}
          </section>

          <section className="smoi-sidebar-section smoi-recents-section" aria-labelledby="smoi-recent-heading">
            <div className="smoi-sidebar-section-title">
              <h2 className="smoi-sidebar-heading" id="smoi-recent-heading">
                <i className="bi bi-clock-history" aria-hidden="true" />
                {activeProject ? activeProject.name : t('recentConversations')}
              </h2>
              {activeProject && (
                <button className="smoi-section-action" type="button" onClick={() => setActiveProjectId('')} aria-label={t('allConversations')}>
                  <i className="bi bi-x-lg" aria-hidden="true" />
                </button>
              )}
            </div>
            <nav className="smoi-history" aria-labelledby="smoi-recent-heading">
              {visibleConversations.map(renderConversation)}
              {visibleConversations.length === 0 && (
                <p className="smoi-empty-history">{activeProject ? t('emptyProject') : t('noConversations')}</p>
              )}
              {!activeProjectId && hasMore && (
                <button className="btn btn-link smoi-load-more" type="button" onClick={onLoadMore}>
                  {t('loadMoreConversations')}
                </button>
              )}
            </nav>
          </section>
        </div>
      </aside>
    </>
  )
}
