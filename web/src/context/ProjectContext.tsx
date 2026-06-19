import { createContext, useContext } from 'react'

export interface ProjectContextType {
  activeProject: string
  setActiveProject: (project: string) => void
}

export const ProjectContext = createContext<ProjectContextType | undefined>(undefined)

export function useProject() {
  const context = useContext(ProjectContext)
  if (!context) {
    return {
      activeProject: 'default',
      setActiveProject: () => {},
    }
  }
  return context
}
