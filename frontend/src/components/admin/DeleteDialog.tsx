import { useLanguage } from '../../i18n/LanguageProvider'
import { useEffect, useRef } from 'react'

export interface DeletionTarget {
  type: 'users' | 'conversations'
  id: string
  label: string
}

interface DeleteDialogProps {
  target: DeletionTarget | null
  saving: boolean
  onCancel: () => void
  onConfirm: () => void
}

export default function DeleteDialog({
  target, saving, onCancel, onConfirm
}: DeleteDialogProps) {
  const { t } = useLanguage()
  const dialog = useRef<HTMLDialogElement>(null)

  useEffect(() => {
    if (target) dialog.current?.showModal()
    else dialog.current?.close()
  }, [target])

  return (
    <dialog
      ref={dialog}
      className="smoi-dialog"
      aria-labelledby="smoi-delete-title"
      aria-describedby="smoi-delete-description"
      onClose={onCancel}
    >
      <h2 id="smoi-delete-title" className="h5">{t('confirmPermanentDeletion')}</h2>
      <p id="smoi-delete-description" className="text-body-secondary">
        {t('deleteWithFiles', { name: target?.label || '' })}
      </p>
      <div className="d-flex justify-content-end gap-2">
        <button className="btn btn-outline-secondary" type="button" onClick={onCancel}>
          {t('cancel')}
        </button>
        <button
          className="btn btn-danger"
          type="button"
          disabled={saving}
          onClick={onConfirm}
        >
          {t(saving ? 'deleting' : 'deletePermanently')}
        </button>
      </div>
    </dialog>
  )
}
