import { useEffect, useState } from 'react'

export interface ConversationProject {
  id: string
  name: string
  conversationIds: string[]
}

interface OrganizationState {
  owner: string
  pinnedIds: string[]
  projects: ConversationProject[]
}

function readOrganization(userId: string): OrganizationState {
  if (!userId) return { owner: '', pinnedIds: [], projects: [] }

  try {
    const raw = localStorage.getItem(`smoi-organization-${userId}`)
    if (raw) {
      const value = JSON.parse(raw) as Partial<OrganizationState>
      return {
        owner: userId,
        pinnedIds: Array.isArray(value.pinnedIds) ? value.pinnedIds.filter((id): id is string => typeof id === 'string') : [],
        projects: Array.isArray(value.projects)
          ? value.projects.filter(project => project && typeof project.id === 'string' && typeof project.name === 'string')
            .map(project => ({
              id: project.id,
              name: project.name,
              conversationIds: Array.isArray(project.conversationIds)
                ? project.conversationIds.filter((id): id is string => typeof id === 'string')
                : []
            }))
          : []
      }
    }
  } catch {
    return { owner: userId, pinnedIds: [], projects: [] }
  }

  return { owner: userId, pinnedIds: [], projects: [] }
}

function createId(): string {
  return typeof crypto !== 'undefined' && 'randomUUID' in crypto
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function useConversationOrganization(userId: string) {
  const [state, setState] = useState<OrganizationState>(() => readOrganization(userId))

  useEffect(() => {
    if (!userId) {
      setState({ owner: '', pinnedIds: [], projects: [] })
      return
    }

    setState(readOrganization(userId))
  }, [userId])

  useEffect(() => {
    if (!userId || state.owner !== userId) return

    try {
      localStorage.setItem(`smoi-organization-${userId}`, JSON.stringify({
        pinnedIds: state.pinnedIds,
        projects: state.projects
      }))
    } catch {
      return
    }
  }, [state, userId])

  function togglePin(conversationId: string) {
    setState(previous => ({
      ...previous,
      pinnedIds: previous.pinnedIds.includes(conversationId)
        ? previous.pinnedIds.filter(id => id !== conversationId)
        : [conversationId, ...previous.pinnedIds]
    }))
  }

  function createProject(name: string) {
    const trimmed = name.trim()
    if (!trimmed) return

    setState(previous => ({
      ...previous,
      projects: [...previous.projects, { id: createId(), name: trimmed, conversationIds: [] }]
    }))
  }

  function deleteProject(projectId: string) {
    setState(previous => ({
      ...previous,
      projects: previous.projects.filter(project => project.id !== projectId)
    }))
  }

  function assignConversation(conversationId: string, projectId: string) {
    setState(previous => ({
      ...previous,
      projects: previous.projects.map(project => ({
        ...project,
        conversationIds: project.id === projectId
          ? [...new Set([...project.conversationIds, conversationId])]
          : project.conversationIds.filter(id => id !== conversationId)
      }))
    }))
  }

  return {
    pinnedIds: state.pinnedIds,
    projects: state.projects,
    togglePin,
    createProject,
    deleteProject,
    assignConversation
  }
}
