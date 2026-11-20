import { useLanguage, type Language } from '../../i18n/LanguageProvider'

const languages: { code: Language; label: string }[] = [
  { code: 'en', label: '🇺🇸 English' },
  { code: 'es', label: '🇪🇸 Español' },
  { code: 'pt-BR', label: '🇧🇷 Português' }
]

export default function LanguageSwitcher() {
  const { language, setLanguage, t } = useLanguage()

  return (
    <label className="smoi-language-select-wrap">
      <span className="visually-hidden">{t('languageSwitcher')}</span>
      <select
        className="form-select smoi-language-select"
        aria-label={t('languageSwitcher')}
        value={language}
        onChange={event => setLanguage(event.target.value as Language)}
      >
        {languages.map(option => (
          <option key={option.code} value={option.code}>{option.label}</option>
        ))}
      </select>
    </label>
  )
}
