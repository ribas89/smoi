import { useLanguage } from '../../i18n/LanguageProvider'
import type { AdminUser } from '../../types'
import { formatStorageSize } from '../../services/format'

interface UserTableProps {
  users: AdminUser[]
  currentUserId: string
  onDelete: (id: string, label: string) => void
}

export default function UserTable({ users, currentUserId, onDelete }: UserTableProps) {
  const { t } = useLanguage()

  return (
    <div className="card mb-4">
      <div className="card-header d-flex align-items-center justify-content-between">
        <strong>{t('users')}</strong>
        <span className="text-body-secondary small">{t('sortedByStorage')}</span>
      </div>

      <div className="table-responsive">
        <table className="table align-middle mb-0">
          <caption className="visually-hidden">{t('userStorageCaption')}</caption>
          <thead>
            <tr>
              <th scope="col">{t('user')}</th>
              <th scope="col">{t('role')}</th>
              <th scope="col">{t('chats')}</th>
              <th scope="col">{t('storage')}</th>
              <th scope="col" className="text-end">{t('actions')}</th>
            </tr>
          </thead>
          <tbody>
            {users.map(member => (
              <tr key={member.id}>
                <td>
                  <div className="fw-medium">{member.display_name || member.username}</div>
                  <small className="text-body-secondary">{member.email}</small>
                </td>
                <td>{t(member.role === 'admin' ? 'roleAdministrator' : 'roleRegular')}</td>
                <td>{member.conversation_count}</td>
                <td>{formatStorageSize(member.storage_bytes)}</td>
                <td className="text-end">
                  <button
                    className="btn btn-sm btn-outline-danger"
                    type="button"
                    disabled={member.id === currentUserId}
                    aria-label={t('deleteUser', { name: member.display_name || member.username })}
                    onClick={() => onDelete(member.id, member.display_name || member.username)}
                  >
                    <i className="bi bi-trash me-1" aria-hidden="true" />{t('delete')}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
