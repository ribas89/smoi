import { useLanguage } from '../../i18n/LanguageProvider'
import LanguageSwitcher from './LanguageSwitcher'
import TextSizeControls from './TextSizeControls'
import { login } from '../../services/api'

interface LoginPageProps {
  error: string
  textSize: number
  canDecrease: boolean
  canIncrease: boolean
  onDecrease: () => void
  onReset: () => void
  onIncrease: () => void
}

export default function LoginPage({
  error,
  textSize,
  canDecrease,
  canIncrease,
  onDecrease,
  onReset,
  onIncrease
}: LoginPageProps) {
  const { t } = useLanguage()

  return (
    <main className="smoi-login">
      <section className="smoi-login-card" aria-labelledby="smoi-login-title">
        <div className="smoi-login-brand">
          <img className="smoi-login-logo" src="/assets/smoi.svg" alt={t('smoiLogo')} />
          <h1 id="smoi-login-title">SMOI</h1>
        </div>

        <div className="smoi-login-tools">
          <LanguageSwitcher />
          <TextSizeControls
            textSize={textSize}
            canDecrease={canDecrease}
            canIncrease={canIncrease}
            onDecrease={onDecrease}
            onReset={onReset}
            onIncrease={onIncrease}
          />
        </div>

        {error && (
          <div className="alert alert-danger smoi-login-alert" role="alert">
            {error}
          </div>
        )}

        <button className="btn smoi-login-button w-100" type="button" onClick={login}>
          <i className="bi bi-box-arrow-in-right" aria-hidden="true" />
          <span>{t('loginButton')}</span>
        </button>
      </section>
    </main>
  )
}
