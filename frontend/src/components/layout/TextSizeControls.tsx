import { useLanguage } from '../../i18n/LanguageProvider'

interface TextSizeControlsProps {
  textSize: number
  canDecrease: boolean
  canIncrease: boolean
  onDecrease: () => void
  onReset: () => void
  onIncrease: () => void
}

export default function TextSizeControls({
  textSize,
  canDecrease,
  canIncrease,
  onDecrease,
  onReset,
  onIncrease
}: TextSizeControlsProps) {
  const { t } = useLanguage()

  return (
    <>
      <div className="smoi-text-controls" role="group" aria-label={t('textSize')}>
        <button
          className="smoi-text-button"
          type="button"
          aria-label={t('decreaseTextSize')}
          title={t('decreaseTextSize')}
          disabled={!canDecrease}
          onClick={onDecrease}
        >
          A<span aria-hidden="true">-</span>
        </button>
        <button
          className="smoi-text-button smoi-text-reset"
          type="button"
          aria-label={t('resetTextSize')}
          title={t('resetTextSize')}
          aria-pressed={textSize === 16}
          onClick={onReset}
        >
          A
        </button>
        <button
          className="smoi-text-button"
          type="button"
          aria-label={t('increaseTextSize')}
          title={t('increaseTextSize')}
          disabled={!canIncrease}
          onClick={onIncrease}
        >
          A<span aria-hidden="true">+</span>
        </button>
      </div>
      <span className="visually-hidden" role="status" aria-live="polite" aria-atomic="true">
        {t('textSizeAnnouncement', { size: `${textSize}px` })}
      </span>
    </>
  )
}
