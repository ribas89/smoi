import { useLanguage } from '../../i18n/LanguageProvider'

interface ChatHeaderProps {
  title: string
  isAdminOpen: boolean
  isSidebarOpen: boolean
  isPreferencesOpen: boolean
  onOpenSidebar: () => void
  onOpenPreferences: () => void
}

export default function ChatHeader({
  title,
  isAdminOpen,
  isSidebarOpen,
  isPreferencesOpen,
  onOpenSidebar,
  onOpenPreferences
}: ChatHeaderProps) {
  const { t } = useLanguage()

  return (
    <header className="smoi-topbar">
      <button
        id="smoi-mobile-menu"
        className="smoi-menu-button"
        type="button"
        aria-label={t(isSidebarOpen ? 'closeNavigation' : 'openNavigation')}
        aria-controls="smoi-sidebar"
        aria-expanded={isSidebarOpen}
        onClick={onOpenSidebar}
      >
        <i className="bi bi-layout-sidebar" aria-hidden="true" />
      </button>

      <h1 className="smoi-topbar-title">
        {isAdminOpen ? t('administration') : title}
      </h1>

      <button
        className="smoi-preferences-trigger"
        type="button"
        aria-label={t(isPreferencesOpen ? 'closePreferences' : 'openPreferences')}
        aria-controls="smoi-preferences-sidebar"
        aria-expanded={isPreferencesOpen}
        onClick={onOpenPreferences}
      >
        <i className="bi bi-layout-sidebar-reverse" aria-hidden="true" />
      </button>
    </header>
  )
}
