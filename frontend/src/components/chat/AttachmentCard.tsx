import { useLanguage } from '../../i18n/LanguageProvider'
import { useEffect, useState } from 'react'
import { attachmentBlob, downloadAttachment } from '../../services/api'
import type { Attachment } from '../../types'

export default function AttachmentCard({ attachment }: { attachment: Attachment }) {
  const { t } = useLanguage()
  const [preview, setPreview] = useState('')
  const isImage = attachment.mime_type.startsWith('image/')

  useEffect(() => {
    if (!isImage) return

    let disposed = false
    let previewURL = ''

    attachmentBlob(attachment.id)
      .then(blob => {
        if (disposed) return
        previewURL = URL.createObjectURL(blob)
        setPreview(previewURL)
      })
      .catch(() => setPreview(''))

    return () => {
      disposed = true
      if (previewURL) URL.revokeObjectURL(previewURL)
    }
  }, [attachment.id, isImage])

  return (
    <button
      className="smoi-attachment"
      type="button"
      title={t('downloadFile', { name: attachment.filename })}
      aria-label={t('downloadAttachment', { name: attachment.filename })}
      onClick={() => void downloadAttachment(attachment.id, attachment.filename)}
    >
      {preview ? (
        <img src={preview} alt={attachment.filename} className="smoi-image-preview" />
      ) : (
        <i className={`bi ${isImage ? 'bi-image' : 'bi-file-earmark-text'}`} aria-hidden="true" />
      )}
      <span className="smoi-attachment-name">{attachment.filename}</span>
      <i className="bi bi-download smoi-download-icon" aria-hidden="true" />
    </button>
  )
}
