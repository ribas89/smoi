import { useLanguage } from '../../i18n/LanguageProvider'
import type { AdminStats } from '../../types'
import { formatStorageSize } from '../../services/format'

export default function StorageSummary({ stats }: { stats: AdminStats | null }) {
  const { t } = useLanguage()
  const values = [
    { label: t('users'), value: stats?.users ?? '…' },
    { label: t('conversations'), value: stats?.conversations ?? '…' },
    {
      label: t('messagesAndAttachments'),
      value: stats ? formatStorageSize(stats.storage_bytes) : '…'
    }
  ]

  return (
    <div className="smoi-stats-grid mb-4">
      {values.map(item => (
        <div className="card" key={item.label}>
          <div className="card-body">
            <small className="text-body-secondary">{item.label}</small>
            <div className="h3 mb-0">{item.value}</div>
          </div>
        </div>
      ))}
    </div>
  )
}
