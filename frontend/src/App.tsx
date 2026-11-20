import { useLanguage } from './i18n/LanguageProvider'
import { useEffect, useMemo, useState } from 'react'
import Admin from './pages/Admin'
import ChatPanel from './components/chat/ChatPanel'
import ChatHeader from './components/layout/ChatHeader'
import LoginPage from './components/layout/LoginPage'
import PreferencesSidebar from './components/layout/PreferencesSidebar'
import Sidebar from './components/layout/Sidebar'
import { useChat } from './hooks/useChat'
import { useConversations } from './hooks/useConversations'
import { useConversationOrganization } from './hooks/useConversationOrganization'
import { useModels } from './hooks/useModels'
import { useSession, errorMessage } from './hooks/useSession'
import { useTheme } from './hooks/useTheme'
import { useTextSize } from './hooks/useTextSize'
import { request } from './services/api'
import { conversationPath, navigate } from './services/navigation'
import type { Conversation } from './types'

interface SidebarVisibility {
  left: boolean
  right: boolean
}

const sidebarStorageKey = 'smoi-sidebar-visibility'

function readSidebarVisibility(isCompact: boolean): SidebarVisibility {
  if (isCompact) return { left: false, right: false }

  try {
    const saved = localStorage.getItem(sidebarStorageKey)
    if (saved) {
      const parsed = JSON.parse(saved) as Partial<SidebarVisibility>
      return {
        left: typeof parsed.left === 'boolean' ? parsed.left : true,
        right: typeof parsed.right === 'boolean' ? parsed.right : true
      }
    }
  } catch {
    return { left: true, right: true }
  }

  return { left: true, right: true }
}

export default function App() {
  const { t } = useLanguage()
  const { user, isReady, error, setError, signOut } = useSession()
  const { theme, toggleTheme } = useTheme()
  const {
    textSize, canDecrease, canIncrease, decreaseTextSize, increaseTextSize, resetTextSize
  } = useTextSize()
  const [isCompact, setIsCompact] = useState(() => window.matchMedia('(max-width: 48rem)').matches)
  const [sidebarVisibility, setSidebarVisibility] = useState<SidebarVisibility>(() =>
    readSidebarVisibility(window.matchMedia('(max-width: 48rem)').matches)
  )
  const isSidebarOpen = sidebarVisibility.left
  const isPreferencesOpen = sidebarVisibility.right
  const [isAdminOpen, setIsAdminOpen] = useState(() => window.location.pathname === '/admin')
  const [linkedConversation, setLinkedConversation] = useState<Conversation | null>(null)
  const [organizedConversations, setOrganizedConversations] = useState<Conversation[]>([])

  useEffect(() => {
    const query = window.matchMedia('(max-width: 48rem)')
    const handleChange = (event: MediaQueryListEvent) => {
      setIsCompact(event.matches)
      setSidebarVisibility(readSidebarVisibility(event.matches))
    }

    query.addEventListener('change', handleChange)
    return () => query.removeEventListener('change', handleChange)
  }, [])

  useEffect(() => {
    if (isCompact) return

    try {
      localStorage.setItem(sidebarStorageKey, JSON.stringify(sidebarVisibility))
    } catch {
      return
    }
  }, [isCompact, sidebarVisibility])

  useEffect(() => {
    const viewport = window.visualViewport
    if (!viewport) return

    const root = document.documentElement
    const updateViewport = () => {
      const rootFontSize = Number.parseFloat(getComputedStyle(root).fontSize) || 16
      root.style.setProperty('--smoi-viewport-height', `${viewport.height / rootFontSize}rem`)
      root.style.setProperty('--smoi-viewport-offset-top', `${viewport.offsetTop / rootFontSize}rem`)
    }

    updateViewport()
    viewport.addEventListener('resize', updateViewport)
    viewport.addEventListener('scroll', updateViewport)
    window.addEventListener('resize', updateViewport)

    return () => {
      viewport.removeEventListener('resize', updateViewport)
      viewport.removeEventListener('scroll', updateViewport)
      window.removeEventListener('resize', updateViewport)
      root.style.removeProperty('--smoi-viewport-height')
      root.style.removeProperty('--smoi-viewport-offset-top')
    }
  }, [])
  const conversationStore = useConversations(user, setError)
  const organization = useConversationOrganization(user?.id || '')
  const chat = useChat({
    createConversation: conversationStore.create,
    reloadConversations: conversationStore.reload,
    onError: setError,
    enabled: Boolean(user)
  })

  useEffect(() => {
    if (!user) {
      setOrganizedConversations([])
      return
    }

    const organizedIds = [...new Set([
      ...organization.pinnedIds,
      ...organization.projects.flatMap(project => project.conversationIds)
    ])]
    const knownIds = new Set([
      ...conversationStore.conversations.map(conversation => conversation.id),
      ...organizedConversations.map(conversation => conversation.id)
    ])
    const missingIds = organizedIds.filter(id => !knownIds.has(id))
    if (missingIds.length === 0) return

    let cancelled = false
    void Promise.all(missingIds.map(id =>
      request<Conversation>(`/conversations/${id}`).catch(() => null)
    )).then(results => {
      if (cancelled) return
      const found = results.filter((conversation): conversation is Conversation => conversation !== null)
      setOrganizedConversations(previous => {
        const byId = new Map(previous.map(conversation => [conversation.id, conversation]))
        found.forEach(conversation => byId.set(conversation.id, conversation))
        return Array.from(byId.values())
      })
    })

    return () => { cancelled = true }
  }, [user?.id, organization.pinnedIds, organization.projects, conversationStore.conversations])

  const sidebarConversations = useMemo(() => {
    const items = conversationStore.search.trim()
      ? conversationStore.conversations
      : [...conversationStore.conversations, ...organizedConversations]
    return Array.from(new Map(items.map(conversation => [conversation.id, conversation])).values())
      .sort((a, b) => b.updated_at - a.updated_at)
  }, [conversationStore.conversations, conversationStore.search, organizedConversations])

  useEffect(() => {
    function handleHistoryChange() {
      setIsAdminOpen(window.location.pathname === '/admin')
    }

    window.addEventListener('popstate', handleHistoryChange)
    return () => window.removeEventListener('popstate', handleHistoryChange)
  }, [])

  useEffect(() => {
    const id = chat.selectedId
    if (!user || !id || conversationStore.conversations.some(item => item.id === id)) {
      setLinkedConversation(null)
      return
    }

    let cancelled = false
    void request<Conversation>(`/conversations/${id}`)
      .then(conversation => {
        if (!cancelled) setLinkedConversation(conversation)
      })
      .catch(cause => {
        if (!cancelled) setError(errorMessage(cause, t('couldNotLoadConversation')))
      })

    return () => { cancelled = true }
  }, [user, chat.selectedId, conversationStore.conversations, setError])

  const selectedConversation = conversationStore.conversations.find(
    conversation => conversation.id === chat.selectedId
  ) || (linkedConversation?.id === chat.selectedId ? linkedConversation : undefined)

  const { model, setModel, availableModels, maxUploadMB } = useModels(
    user,
    selectedConversation,
    setError
  )

  async function send(content: string, files: File[]) {
    return chat.send(content, files, model)
  }

  function closeSidebars() {
    if (isCompact) setSidebarVisibility({ left: false, right: false })
  }

  function toggleSidebar(side: keyof SidebarVisibility) {
    setSidebarVisibility(previous => {
      const nextOpen = !previous[side]

      if (isCompact && nextOpen) {
        return {
          left: side === 'left',
          right: side === 'right'
        }
      }

      return { ...previous, [side]: nextOpen }
    })
  }

  function openConversation(id: string) {
    chat.selectConversation(id)
    navigate(conversationPath(id))
    setIsAdminOpen(false)
    closeSidebars()
  }

  function newConversation() {
    chat.startNewConversation()
    navigate('/')
    setIsAdminOpen(false)
    closeSidebars()
  }

  async function renameConversation(id: string, title: string) {
    try {
      await conversationStore.update(id, { title })
    } catch (cause) {
      setError(errorMessage(cause, t('couldNotRenameConversation')))
    }
  }

  async function deleteConversation(conversation: Conversation) {
    if (!window.confirm(t('confirmDeleteConversation', { title: conversation.title }))) return

    try {
      if (chat.selectedId === conversation.id) {
        chat.startNewConversation()
        navigate('/', true)
        setIsAdminOpen(false)
      }
      await conversationStore.remove(conversation.id)
      if (organization.pinnedIds.includes(conversation.id)) organization.togglePin(conversation.id)
      organization.assignConversation(conversation.id, '')
      setOrganizedConversations(previous => previous.filter(item => item.id !== conversation.id))
    } catch (cause) {
      setError(errorMessage(cause, t('couldNotDeleteConversation')))
    }
  }

  async function chooseModel(nextModel: string) {
    setModel(nextModel)
    try {
      localStorage.setItem('smoi-model', nextModel)
    } catch {
      setError(t('couldNotSaveModelPreference'))
    }

    if (!chat.selectedId) return

    try {
      await conversationStore.update(chat.selectedId, { model: nextModel })
    } catch (cause) {
      setError(errorMessage(cause, t('couldNotChangeModel')))
    }
  }

  useEffect(() => {
    document.title = `${isAdminOpen ? t('administration') : selectedConversation?.title || t('newConversation')} | SMOI`
  }, [isAdminOpen, selectedConversation?.title, t])

  if (!isReady) {
    return (
      <div className="smoi-loading" role="status">
        <div className="spinner-border" aria-hidden="true" />
        <span>{t('loading')}</span>
      </div>
    )
  }

  if (!user) return (
    <LoginPage
      error={error}
      textSize={textSize}
      canDecrease={canDecrease}
      canIncrease={canIncrease}
      onDecrease={decreaseTextSize}
      onReset={resetTextSize}
      onIncrease={increaseTextSize}
    />
  )

  return (
    <div className={`smoi-app ${isSidebarOpen ? 'has-left-sidebar' : ''} ${isPreferencesOpen ? 'has-right-sidebar' : ''}`}>
      <a className="smoi-skip-link" href="#main-content">{t('skipContent')}</a>
      <Sidebar
        conversations={sidebarConversations}
        hasMore={conversationStore.hasMore}
        search={conversationStore.search}
        selectedId={chat.selectedId}
        isOpen={isSidebarOpen}
        isAdminOpen={isAdminOpen}
        pinnedIds={organization.pinnedIds}
        projects={organization.projects}
        onClose={() => setSidebarVisibility(previous => ({ ...previous, left: false }))}
        onNew={newConversation}
        onSearchChange={conversationStore.setSearch}
        onSelect={openConversation}
        onRename={renameConversation}
        onDelete={conversation => void deleteConversation(conversation)}
        onLoadMore={() => void conversationStore.loadMore()}
        onTogglePin={organization.togglePin}
        onCreateProject={organization.createProject}
        onDeleteProject={organization.deleteProject}
        onAssignProject={organization.assignConversation}
      />

      <main id="main-content" className="smoi-main" tabIndex={-1}>
        <ChatHeader
          title={selectedConversation?.title || t('newConversation')}
          isAdminOpen={isAdminOpen}
          isSidebarOpen={isSidebarOpen}
          isPreferencesOpen={isPreferencesOpen}
          onOpenSidebar={() => toggleSidebar('left')}
          onOpenPreferences={() => toggleSidebar('right')}
        />

        {isAdminOpen ? (
          <Admin user={user} onBack={() => {
            navigate(chat.selectedId ? conversationPath(chat.selectedId) : '/')
            setIsAdminOpen(false)
          }} />
        ) : (
          <ChatPanel
            selectedId={chat.selectedId}
            messages={chat.messages}
            isSending={chat.isSending}
            jobs={chat.jobs}
            maxUploadMB={maxUploadMB}
            error={error}
            onDismissError={() => setError('')}
            onSend={send}
            onStop={chat.stopGeneration}
          />
        )}
      </main>

      <PreferencesSidebar
        user={user}
        theme={theme}
        isOpen={isPreferencesOpen}
        isAdminOpen={isAdminOpen}
        model={model}
        availableModels={availableModels}
        textSize={textSize}
        canDecreaseTextSize={canDecrease}
        canIncreaseTextSize={canIncrease}
        onClose={() => setSidebarVisibility(previous => ({ ...previous, right: false }))}
        onSignOut={() => {
          closeSidebars()
          void signOut()
        }}
        onToggleTheme={toggleTheme}
        onOpenAdmin={() => {
          navigate('/admin')
          setIsAdminOpen(true)
          closeSidebars()
        }}
        onChooseModel={nextModel => void chooseModel(nextModel)}
        onDecreaseTextSize={decreaseTextSize}
        onResetTextSize={resetTextSize}
        onIncreaseTextSize={increaseTextSize}
      />
    </div>
  )
}
