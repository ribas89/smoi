import { useEffect, useRef, useState, type KeyboardEvent } from 'react'
import { useLanguage } from '../../i18n/LanguageProvider'
import type { User } from '../../types'
import UserAvatar from './UserAvatar'
import LanguageSwitcher from './LanguageSwitcher'
import TextSizeControls from './TextSizeControls'

interface PreferencesSidebarProps {
  user: User
  theme: 'light' | 'dark'
  isOpen: boolean
  isAdminOpen: boolean
  model: string
  availableModels: string[]
  textSize: number
  canDecreaseTextSize: boolean
  canIncreaseTextSize: boolean
  onClose: () => void
  onSignOut: () => void
  onToggleTheme: () => void
  onOpenAdmin: () => void
  onChooseModel: (model: string) => void
  onDecreaseTextSize: () => void
  onResetTextSize: () => void
  onIncreaseTextSize: () => void
}

export default function PreferencesSidebar({
  user,
  theme,
  isOpen,
  isAdminOpen,
  model,
  availableModels,
  textSize,
  canDecreaseTextSize,
  canIncreaseTextSize,
  onClose,
  onSignOut,
  onToggleTheme,
  onOpenAdmin,
  onChooseModel,
  onDecreaseTextSize,
  onResetTextSize,
  onIncreaseTextSize
}: PreferencesSidebarProps) {
  const { t } = useLanguage()
  const [isCompact, setIsCompact] = useState(() => window.matchMedia('(max-width: 48rem)').matches)
  const sidebarRef = useRef<HTMLElement>(null)
  const closeButtonRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    const query = window.matchMedia('(max-width: 48rem)')
    const handleChange = () => setIsCompact(query.matches)
    query.addEventListener('change', handleChange)
    return () => query.removeEventListener('change', handleChange)
  }, [])

  useEffect(() => {
    if (!isCompact || !isOpen) return

    const previousFocus = document.activeElement as HTMLElement | null
    closeButtonRef.current?.focus()

    return () => previousFocus?.focus()
  }, [isCompact, isOpen])

  function handleKeyDown(event: KeyboardEvent<HTMLElement>) {
    if (!isCompact || !isOpen) return

    if (event.key === 'Escape') {
      event.preventDefault()
      onClose()
      return
    }

    if (event.key !== 'Tab') return

    const items = Array.from(sidebarRef.current?.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled])'
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

  return (
    <>
      {isCompact && isOpen && (
        <button
          type="button"
          className="smoi-mobile-backdrop smoi-preferences-backdrop"
          aria-label={t('closePreferences')}
          tabIndex={-1}
          onClick={onClose}
        />
      )}

      <aside
        id="smoi-preferences-sidebar"
        ref={sidebarRef}
        className={`smoi-preferences-sidebar ${isOpen ? 'is-open' : ''}`}
        aria-label={t('preferences')}
        aria-modal={isCompact && isOpen ? true : undefined}
        role={isCompact && isOpen ? 'dialog' : undefined}
        inert={isCompact && !isOpen}
        onKeyDown={handleKeyDown}
      >
        <div className="smoi-preferences-header">
          <div className="smoi-account-header">
            <UserAvatar user={user} />
            <button
              type="button"
              className="smoi-logout"
              aria-label={t('signOut')}
              title={t('signOut')}
              onClick={onSignOut}
            >
              <span>{t('signOut')}</span>
              <i className="bi bi-box-arrow-right" aria-hidden="true" />
            </button>
          </div>
          <button
            ref={closeButtonRef}
            className="smoi-sidebar-mobile-close"
            type="button"
            aria-label={t('closePreferences')}
            onClick={onClose}
          >
            <i className="bi bi-x-lg" aria-hidden="true" />
          </button>
        </div>

        <div className="smoi-preferences-content">
          <section className="smoi-preferences-section" aria-labelledby="smoi-language-heading">
            <h2 className="smoi-preferences-heading" id="smoi-language-heading">{t('language')}</h2>
            <LanguageSwitcher />
          </section>

          <section className="smoi-preferences-section" aria-labelledby="smoi-font-heading">
            <h2 className="smoi-preferences-heading" id="smoi-font-heading">{t('textSize')}</h2>
            <TextSizeControls
              textSize={textSize}
              canDecrease={canDecreaseTextSize}
              canIncrease={canIncreaseTextSize}
              onDecrease={onDecreaseTextSize}
              onReset={onResetTextSize}
              onIncrease={onIncreaseTextSize}
            />
          </section>

          <section className="smoi-preferences-section" aria-labelledby="smoi-appearance-heading">
            <h2 className="smoi-preferences-heading" id="smoi-appearance-heading">{t('appearance')}</h2>
            <div className="smoi-appearance-options" role="group" aria-label={t('appearance')}>
              <button
                type="button"
                className={`smoi-appearance-option ${theme === 'light' ? 'active' : ''}`}
                aria-pressed={theme === 'light'}
                onClick={() => { if (theme !== 'light') onToggleTheme() }}
              >
                <i className="bi bi-sun" aria-hidden="true" />
                {t('lightMode')}
              </button>
              <button
                type="button"
                className={`smoi-appearance-option ${theme === 'dark' ? 'active' : ''}`}
                aria-pressed={theme === 'dark'}
                onClick={() => { if (theme !== 'dark') onToggleTheme() }}
              >
                <i className="bi bi-moon" aria-hidden="true" />
                {t('darkMode')}
              </button>
            </div>
          </section>

          {user.role === 'admin' && (
            <section className="smoi-preferences-section smoi-administration-section" aria-labelledby="smoi-administration-heading">
              <h2 className="smoi-preferences-heading" id="smoi-administration-heading">{t('administration')}</h2>
              <button
                type="button"
                className={`smoi-preference-link ${isAdminOpen ? 'active' : ''}`}
                aria-current={isAdminOpen ? 'page' : undefined}
                onClick={onOpenAdmin}
              >
                <i className="bi bi-sliders2" aria-hidden="true" />
                {t('storageAdministration')}
                <i className="bi bi-chevron-right ms-auto" aria-hidden="true" />
              </button>
            </section>
          )}

          <section className="smoi-preferences-section smoi-model-section" aria-labelledby="smoi-model-heading">
            <h2 className="smoi-preferences-heading" id="smoi-model-heading">{t('model')}</h2>
            <select
              id="smoi-model-select"
              className="smoi-model-select"
              aria-labelledby="smoi-model-heading"
              value={model || ''}
              disabled={availableModels.length === 0}
              onChange={event => onChooseModel(event.currentTarget.value)}
            >
              {availableModels.length === 0 ? (
                <option value="">{t('noModels')}</option>
              ) : availableModels.map(value => (
                <option key={value} value={value}>{value}</option>
              ))}
            </select>
          </section>
        </div>
      </aside>
    </>
  )
}
