// src/routes/index.tsx
import { useState, useEffect, useRef } from 'react'
import {
  FileText,
  Users,
  Globe,
  Building2,
  Terminal,
  Play,
  Loader2,
  CheckCircle,
  Database,
} from 'lucide-react'
import { mockMetrics } from '../lib/mock-stratum'

interface TargetRowProps {
  name: string
  description: string
  metric: string
  status: 'ACTIVE_SYNC' | 'STABLE' | 'PENDING'
  tags: string[]
  icon: React.ReactNode
}

function TargetRow({ name, description, metric, status, tags, icon }: TargetRowProps) {
  const statusStyles = {
    ACTIVE_SYNC:
      'bg-green-50 text-green-700 border-green-200 dark:bg-green-950/20 dark:text-green-400 dark:border-green-800/40',
    STABLE:
      'bg-zinc-100 text-zinc-700 border-zinc-200 dark:bg-zinc-800 dark:text-zinc-300 dark:border-zinc-700/60',
    PENDING:
      'bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-950/20 dark:text-amber-400 dark:border-amber-800/40',
  }

  return (
    <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between p-4 border border-zinc-200 dark:border-zinc-850 hover:bg-zinc-50 dark:hover:bg-zinc-900/20 transition-all font-mono text-xs">
      <div className="flex items-center gap-4 w-full sm:w-auto">
        <div className="h-8 w-8 rounded bg-zinc-100 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 flex items-center justify-center text-zinc-600 dark:text-zinc-400 shrink-0">
          {icon}
        </div>
        <div className="flex flex-col gap-0.5">
          <span className="font-bold text-sm tracking-tight text-zinc-900 dark:text-zinc-100">
            {name}
          </span>
          <span className="text-zinc-500 dark:text-zinc-400 text-[11px] font-sans">
            {description}
          </span>
        </div>
      </div>

      <div className="flex items-center gap-4 mt-3 sm:mt-0 w-full sm:w-auto justify-between sm:justify-end shrink-0">
        <div className="flex items-center gap-1.5">
          {tags.map((tag) => (
            <span
              key={tag}
              className="px-2 py-0.5 rounded-full border border-zinc-200 bg-zinc-50 text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900 text-[10px] uppercase font-semibold"
            >
              {tag}
            </span>
          ))}
        </div>
        <div className="w-24 text-right">
          <span className="font-mono font-bold text-base text-zinc-800 dark:text-zinc-200">
            {metric}
          </span>
        </div>
        <div className="w-28 text-right">
          <span
            className={`px-2.5 py-1 rounded border text-[9px] font-semibold uppercase tracking-wider ${statusStyles[status]}`}
          >
            {status}
          </span>
        </div>
      </div>
    </div>
  )
}

const mockLogPool = [
  'Establishing socket connection to stratum-core backend...',
  'Authentication token confirmed.',
  'Initializing openalex.DownloadPapers worker pool (size: 4)...',
  'Executing OpenAlex works query using keywords_file and topics_file...',
  'Discovered total match count: 12,450 records.',
  'Ingesting page cursor 1, fetching works...',
  'Parsed 200 works from page 1.',
  'Ingesting page cursor 2, fetching works...',
  'Parsed 200 works from page 2.',
  'Ingesting page cursor 3, fetching works...',
  'Parsed 200 works from page 3.',
  'Analyzing contribution affiliation strings...',
  'Invoking Crossref metadata lookups for missing records...',
  'Triggering ImputeLLM worker utilizing Ollama llama3 provider...',
  'Synchronizing records to disk at data/jsonl/collected_papers.jsonl...',
  'Ingestion completed. 1000 works synchronized to JSONL.',
  'Starting DB conversion. Running dbMgr.CreateSchema...',
  'Importing collected_papers.jsonl into DuckDB dynamic schema...',
  'Finished DuckDB load. Papers: 1000, Authors: 3200, Contributions: 4100.',
  'Sync cycle complete. Database is stable and query-ready.',
]

export function Index() {
  const [syncing, setSyncing] = useState(false)
  const [progress, setProgress] = useState(0)
  const [logs, setLogs] = useState<string[]>([])

  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const logIndexRef = useRef(0)
  const consoleBottomRef = useRef<HTMLDivElement | null>(null)

  useEffect(() => {
    if (syncing) {
      // Setup progress simulation
      timerRef.current = setInterval(() => {
        setProgress((prev) => {
          if (prev >= 100) {
            setSyncing(false)
            if (timerRef.current) clearInterval(timerRef.current)
            return 100
          }

          // Print logs corresponding to progress
          const logCount = Math.floor((prev / 100) * mockLogPool.length)
          const newLogs: string[] = []
          while (logIndexRef.current <= logCount && logIndexRef.current < mockLogPool.length) {
            const timestamp = new Date().toLocaleTimeString()
            newLogs.push(`[${timestamp}] [INFO] ${mockLogPool[logIndexRef.current]}`)
            logIndexRef.current++
          }
          if (newLogs.length > 0) {
            setLogs((oldLogs) => [...oldLogs, ...newLogs])
          }

          return prev + Math.floor(Math.random() * 8) + 4
        })
      }, 300)
    } else {
      if (timerRef.current) clearInterval(timerRef.current)
    }

    return () => {
      if (timerRef.current) clearInterval(timerRef.current)
    }
  }, [syncing])

  useEffect(() => {
    if (consoleBottomRef.current) {
      consoleBottomRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs])

  const handleSyncToggle = () => {
    if (syncing) {
      setSyncing(false)
    } else {
      setProgress(0)
      setLogs([
        `[${new Date().toLocaleTimeString()}] [INFO] Pipeline synchronization initiated by host.`,
      ])
      logIndexRef.current = 1
      setSyncing(true)
    }
  }

  return (
    <div className="flex flex-col gap-8 w-full max-w-5xl mx-auto">
      {/* Header Row */}
      <div className="flex flex-col gap-1 border-b border-zinc-200 pb-5 dark:border-zinc-850">
        <h1 className="text-2xl font-mono font-bold tracking-tight text-zinc-950 dark:text-zinc-50 uppercase">
          Data Management Overview
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          Monitor dataset health coverage, ingest live pipelines, and manage the compiled analytical
          DuckDB database.
        </p>
      </div>

      {/* Aggregate Metric Grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="p-4 border border-zinc-200 bg-zinc-50/50 rounded flex flex-col gap-1.5 dark:border-zinc-850 dark:bg-zinc-900/10">
          <div className="flex items-center justify-between text-zinc-400">
            <span className="text-[10px] font-mono font-bold uppercase tracking-wider">
              Total Papers
            </span>
            <FileText className="h-4 w-4" />
          </div>
          <span className="text-2xl font-mono font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
            {mockMetrics.totalPapers.toLocaleString()}
          </span>
          <span className="text-[10px] text-zinc-400 font-sans">Collected from OpenAlex API</span>
        </div>

        <div className="p-4 border border-zinc-200 bg-zinc-50/50 rounded flex flex-col gap-1.5 dark:border-zinc-850 dark:bg-zinc-900/10">
          <div className="flex items-center justify-between text-zinc-400">
            <span className="text-[10px] font-mono font-bold uppercase tracking-wider">
              Open Access Ratio
            </span>
            <Globe className="h-4 w-4" />
          </div>
          <span className="text-2xl font-mono font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
            {Math.round((mockMetrics.openAccessCount / mockMetrics.totalPapers) * 100)}%
          </span>
          <span className="text-[10px] text-zinc-400 font-sans">
            {mockMetrics.openAccessCount.toLocaleString()} index papers
          </span>
        </div>

        <div className="p-4 border border-zinc-200 bg-zinc-50/50 rounded flex flex-col gap-1.5 dark:border-zinc-850 dark:bg-zinc-900/10">
          <div className="flex items-center justify-between text-zinc-400">
            <span className="text-[10px] font-mono font-bold uppercase tracking-wider">
              Imputed Institutions
            </span>
            <Building2 className="h-4 w-4" />
          </div>
          <span className="text-2xl font-mono font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
            {mockMetrics.imputedInstitutions.toLocaleString()}
          </span>
          <span className="text-[10px] text-zinc-400 font-sans">Resolved via LLM/Crossref</span>
        </div>

        <div className="p-4 border border-zinc-200 bg-zinc-50/50 rounded flex flex-col gap-1.5 dark:border-zinc-850 dark:bg-zinc-900/10">
          <div className="flex items-center justify-between text-zinc-400">
            <span className="text-[10px] font-mono font-bold uppercase tracking-wider">
              Pending Imputations
            </span>
            <Users className="h-4 w-4" />
          </div>
          <span className="text-2xl font-mono font-bold tracking-tight text-zinc-900 dark:text-zinc-50">
            {mockMetrics.unresolvedAffiliations.toLocaleString()}
          </span>
          <span className="text-[10px] text-zinc-400 font-sans">
            Requires further execution cycles
          </span>
        </div>
      </div>

      {/* PostHog Style Rows */}
      <div className="flex flex-col gap-3">
        <h2 className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500">
          Data Stream Indicators
        </h2>
        <div className="flex flex-col border-t border-zinc-200 dark:border-zinc-850">
          <TargetRow
            name="OpenAlex_Works_Download"
            description="Incoming JSONL publication items downloaded via OpenAlex API and stored locally."
            metric="12.4K records"
            status="STABLE"
            tags={['API', 'JSONL']}
            icon={<FileText className="h-4 w-4" />}
          />
          <TargetRow
            name="Imputed_Affiliations"
            description="Pipeline mapping unresolved affiliation strings to synthetic/ROR institution profiles."
            metric="3.8K resolved"
            status={syncing ? 'ACTIVE_SYNC' : 'STABLE'}
            tags={['LLM', 'CROSSREF']}
            icon={<ShuffleIcon />}
          />
          <TargetRow
            name="DuckDB_Analytical_Schema"
            description="Integrated relational tables providing microsecond SQL query execution."
            metric="5 tables"
            status="STABLE"
            tags={['SQL', 'DB']}
            icon={<Database className="h-4 w-4" />}
          />
        </div>
      </div>

      {/* Simulator console */}
      <div className="border border-zinc-200 dark:border-zinc-850 rounded overflow-hidden">
        {/* Console Header */}
        <div className="bg-zinc-50 dark:bg-zinc-900/40 p-4 border-b border-zinc-200 dark:border-zinc-850 flex items-center justify-between flex-wrap gap-3">
          <div className="flex items-center gap-2">
            <Terminal className="h-4 w-4 text-zinc-400" />
            <span className="text-xs font-mono font-bold uppercase tracking-wide">
              Sync Diagnostics Console
            </span>
          </div>

          <button
            onClick={handleSyncToggle}
            className={`flex items-center gap-2 px-3 py-1.5 rounded font-mono text-xs select-none border transition-all ${
              syncing
                ? 'bg-zinc-200 dark:bg-zinc-800 border-zinc-300 dark:border-zinc-700 hover:bg-zinc-300 dark:hover:bg-zinc-700'
                : 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800 dark:border-zinc-200 hover:bg-zinc-800 dark:hover:bg-zinc-200'
            }`}
          >
            {syncing ? (
              <>
                <Loader2 className="h-3 w-3 animate-spin" />
                <span>Pause Sync</span>
              </>
            ) : (
              <>
                <Play className="h-3 w-3 fill-current" />
                <span>Run OpenAlex Sync</span>
              </>
            )}
          </button>
        </div>

        {/* Console Progress Bar */}
        {(syncing || progress > 0) && (
          <div className="h-1.5 w-full bg-zinc-100 dark:bg-zinc-900 border-b border-zinc-200 dark:border-zinc-850">
            <div
              className="h-full bg-zinc-900 dark:bg-zinc-100 transition-all duration-300 ease-out"
              style={{ width: `${Math.min(progress, 100)}%` }}
            ></div>
          </div>
        )}

        {/* Console Terminal Screen */}
        <div className="bg-zinc-950 text-zinc-300 p-4 h-64 overflow-y-auto font-mono text-[11px] leading-relaxed flex flex-col gap-1 border-t-0 select-text">
          {logs.length === 0 ? (
            <div className="h-full flex flex-col items-center justify-center text-zinc-600 select-none">
              <span>Diagnostics Console Idle. Click "Run OpenAlex Sync" to initiate.</span>
            </div>
          ) : (
            <>
              {logs.map((log, index) => (
                <div
                  key={index}
                  className={
                    log.includes('[SUCCESS]')
                      ? 'text-green-400'
                      : log.includes('[ERROR]')
                        ? 'text-red-400'
                        : ''
                  }
                >
                  {log}
                </div>
              ))}
              {progress >= 100 && (
                <div className="text-green-400 font-bold flex items-center gap-1.5 mt-1">
                  <CheckCircle className="h-3 w-3" />
                  <span>[SUCCESS] Database sync completed successfully. All services ready.</span>
                </div>
              )}
              <div ref={consoleBottomRef}></div>
            </>
          )}
        </div>
      </div>
    </div>
  )
}

// Simple shuffle representation SVG
function ShuffleIcon() {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="16"
      height="16"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M16 3h5v5" />
      <path d="M4 20L21 3" />
      <path d="M21 16v5h-5" />
      <path d="M15 15l6 6" />
      <path d="M4 4l5 5" />
    </svg>
  )
}
