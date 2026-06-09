// src/routes/__root.tsx
import { useState, useEffect } from 'react'
import { Link, Outlet } from '@tanstack/react-router'
import { LayoutDashboard, Settings, Database, BookOpen, Sun, Moon } from 'lucide-react'

export function Root() {
  const [darkMode, setDarkMode] = useState(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem('darkMode')
      if (saved !== null) {
        return saved === 'true'
      }
      return window.matchMedia('(prefers-color-scheme: dark)').matches
    }
    return false
  })

  useEffect(() => {
    if (darkMode) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
    localStorage.setItem('darkMode', String(darkMode))
  }, [darkMode])

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-white text-zinc-900 transition-colors duration-200 dark:bg-zinc-950 dark:text-zinc-100 font-sans">
      {/* Left Sidebar */}
      <aside className="w-64 border-r border-zinc-200 bg-zinc-50 flex flex-col justify-between dark:border-zinc-850 dark:bg-zinc-900/40">
        <div className="flex flex-col">
          {/* Top Brand Header */}
          <div className="flex items-center gap-3 p-5 border-b border-zinc-200 dark:border-zinc-850">
            <div className="h-9 w-9 bg-zinc-900 dark:bg-zinc-100 flex items-center justify-center rounded border border-zinc-300 dark:border-zinc-800 shadow-sm">
              <span className="font-mono font-bold text-lg text-white dark:text-zinc-950">S</span>
            </div>
            <div className="flex flex-col">
              <span className="font-mono font-bold text-sm tracking-tight">Stratum Engine</span>
              <span className="text-[10px] font-mono text-zinc-500 font-semibold uppercase tracking-wider">
                v0.1
              </span>
            </div>
          </div>

          {/* Navigation Links */}
          <nav className="flex flex-col gap-1 p-3">
            <Link
              to="/"
              activeProps={{
                className:
                  'bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 font-semibold',
              }}
              inactiveProps={{
                className:
                  'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-900/60',
              }}
              className="flex items-center gap-3 px-3 py-2.5 rounded text-xs font-mono uppercase tracking-wider transition-all"
            >
              <LayoutDashboard className="h-4 w-4" />
              <span>Dashboard</span>
            </Link>

            <Link
              to="/ingest"
              activeProps={{
                className:
                  'bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 font-semibold',
              }}
              inactiveProps={{
                className:
                  'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-900/60',
              }}
              className="flex items-center gap-3 px-3 py-2.5 rounded text-xs font-mono uppercase tracking-wider transition-all"
            >
              <Settings className="h-4 w-4" />
              <span>Ingest Config</span>
            </Link>

            <Link
              to="/sql"
              activeProps={{
                className:
                  'bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 font-semibold',
              }}
              inactiveProps={{
                className:
                  'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-900/60',
              }}
              className="flex items-center gap-3 px-3 py-2.5 rounded text-xs font-mono uppercase tracking-wider transition-all"
            >
              <Database className="h-4 w-4" />
              <span>SQL Playground</span>
            </Link>

            <Link
              to="/docs"
              activeProps={{
                className:
                  'bg-zinc-200 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 font-semibold',
              }}
              inactiveProps={{
                className:
                  'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-900/60',
              }}
              className="flex items-center gap-3 px-3 py-2.5 rounded text-xs font-mono uppercase tracking-wider transition-all"
            >
              <BookOpen className="h-4 w-4" />
              <span>Developer Docs</span>
            </Link>
          </nav>
        </div>

        {/* Bottom Footer with Dark Toggle */}
        <div className="p-4 border-t border-zinc-200 dark:border-zinc-850 bg-zinc-100/60 dark:bg-zinc-900/20">
          <button
            onClick={() => setDarkMode(!darkMode)}
            className="w-full flex items-center justify-between px-3 py-2 rounded border border-zinc-300 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-850 transition text-xs font-mono select-none"
          >
            <span className="text-zinc-500 dark:text-zinc-400 uppercase tracking-wide">Theme</span>
            <div className="flex items-center gap-1.5">
              {darkMode ? (
                <>
                  <Moon className="h-3.5 w-3.5 text-zinc-400" />
                  <span className="text-zinc-300">Night</span>
                </>
              ) : (
                <>
                  <Sun className="h-3.5 w-3.5 text-amber-500" />
                  <span className="text-zinc-700">Light</span>
                </>
              )}
            </div>
          </button>
        </div>
      </aside>

      {/* Main Component Stage */}
      <main className="flex-1 overflow-y-auto p-10 select-text">
        <Outlet />
      </main>
    </div>
  )
}
