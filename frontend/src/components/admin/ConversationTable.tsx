import { useLanguage } from '../../i18n/LanguageProvider'
import type { AdminConversation, AdminUser } from '../../types'
import { formatStorageSize } from '../../services/format'

interface ConversationTableProps {
  conversations: AdminConversation[]
  users: AdminUser[]
  selectedUser: string
  onSelectUser: (id: string) => void
  onDelete: (id: string, label: string) => void
}

export default function ConversationTable({
  conversations, users, selectedUser, onSelectUser, onDelete
}: ConversationTableProps) {
  const { language, t } = useLanguage()

  return (
    <div className="card">
      <div className="card-header d-flex align-items-center justify-content-between gap-3">
        <strong>{t('conversations')}</strong>
        <select
          className="form-select form-select-sm smoi-user-filter"
          aria-label={t('filterConversations')}
          value={selectedUser}
          onChange={event => onSelectUser(event.target.value)}
        >
          <option value="">{t('allUsers')}</option>
          {users.map(member => (
            <option key={member.id} value={member.id}>
              {member.display_name || member.username}
            </option>
          ))}
        </select>
      </div>

      <div className="table-responsive">
        <table className="table align-middle mb-0">
          <caption className="visually-hidden">{t('conversationStorageCaption')}</caption>
          <thead>
            <tr>
              <th scope="col">{t('conversation')}</th>
              <th scope="col">{t('owner')}</th>
              <th scope="col">{t('lastActivity')}</th>
              <th scope="col">{t('storage')}</th>
              <th scope="col" className="text-end">{t('actions')}</th>
            </tr>
          </thead>
          <tbody>
            {conversations.map(conversation => (
              <tr key={conversation.id}>
                <td>{conversation.title}</td>
                <td>{conversation.owner}</td>
                <td>{new Date(conversation.updated_at * 1000).toLocaleDateString(language)}</td>
                <td>{formatStorageSize(conversation.storage_bytes)}</td>
                <td className="text-end">
                  <button
                    className="btn btn-sm btn-outline-danger"
                    type="button"
                    aria-label={t('deleteConversationNamed', { title: conversation.title })}
                    onClick={() => onDelete(conversation.id, conversation.title)}
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
