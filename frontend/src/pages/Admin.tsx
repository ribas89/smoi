import { useLanguage } from '../i18n/LanguageProvider'
import { useEffect, useState } from 'react'
import { request } from '../services/api'
import type { AdminConversation, AdminStats, AdminUser, Page, User } from '../types'
import { errorMessage } from '../hooks/useSession'
import ConversationTable from '../components/admin/ConversationTable'
import DeleteDialog, { type DeletionTarget } from '../components/admin/DeleteDialog'
import StorageSummary from '../components/admin/StorageSummary'
import UserTable from '../components/admin/UserTable'

interface AdminProps {
  user: User
  onBack: () => void
}

export default function Admin({ user, onBack }: AdminProps) {
  const { t } = useLanguage()
  const [stats, setStats] = useState<AdminStats | null>(null)
  const [users, setUsers] = useState<AdminUser[]>([])
  const [conversations, setConversations] = useState<AdminConversation[]>([])
  const [selectedUser, setSelectedUser] = useState('')
  const [pendingDeletion, setPendingDeletion] = useState<DeletionTarget | null>(null)
  const [isDeleting, setIsDeleting] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    void loadData(selectedUser)
  }, [selectedUser])

  async function loadData(userFilter: string) {
    try {
      const filter = encodeURIComponent(userFilter)
      const [overview, members, chats] = await Promise.all([
        request<AdminStats>('/admin/stats'),
        request<Page<AdminUser>>('/admin/users'),
        request<Page<AdminConversation>>(`/admin/conversations?user_id=${filter}`)
      ])

      setStats(overview)
      setUsers(members.data)
      setConversations(chats.data)
      setError('')
    } catch (cause) {
      setError(errorMessage(cause, t('couldNotLoadAdministration')))
    }
  }

  async function deleteSelected() {
    if (!pendingDeletion) return

    setIsDeleting(true)

    try {
      await request(`/admin/${pendingDeletion.type}/${pendingDeletion.id}`, {
        method: 'DELETE'
      })
      setPendingDeletion(null)
      await loadData(selectedUser)
    } catch (cause) {
      setError(errorMessage(cause, t('deletionFailed')))
    } finally {
      setIsDeleting(false)
    }
  }

  function confirmDeletion(type: DeletionTarget['type'], id: string, label: string) {
    setPendingDeletion({ type, id, label })
  }

  return (
    <section className="smoi-admin-container">
      <header className="d-flex align-items-center gap-3 mb-4">
        <button className="btn btn-outline-secondary" type="button" onClick={onBack}>
          <i className="bi bi-arrow-left me-2" aria-hidden="true" />{t('backToChat')}
        </button>
        <div>
          <h2 className="h4 m-0">{t('storageAdministration')}</h2>
          <p className="text-body-secondary small m-0">
            {t('storageDescription')}
          </p>
        </div>
      </header>

      {error && <div className="alert alert-danger" role="alert">{error}</div>}

      <StorageSummary stats={stats} />
      <UserTable
        users={users}
        currentUserId={user.id}
        onDelete={(id, label) => confirmDeletion('users', id, label)}
      />
      <ConversationTable
        conversations={conversations}
        users={users}
        selectedUser={selectedUser}
        onSelectUser={setSelectedUser}
        onDelete={(id, label) => confirmDeletion('conversations', id, label)}
      />
      <DeleteDialog
        target={pendingDeletion}
        saving={isDeleting}
        onCancel={() => setPendingDeletion(null)}
        onConfirm={() => void deleteSelected()}
      />
    </section>
  )
}
