import type { User } from '../../types'

interface UserAvatarProps {
  user: User
}

export default function UserAvatar({ user }: UserAvatarProps) {
  if (user.avatar_url) {
    return <img className="smoi-avatar" src={user.avatar_url} alt="" />
  }

  const name = user.display_name || user.username || 'User'
  const initials = name
    .split(/\s+/)
    .slice(0, 2)
    .map(word => word[0])
    .join('')
    .toUpperCase()

  return <span className="smoi-avatar smoi-initials">{initials}</span>
}
