// src/routes/sql.tsx
import { useState } from 'react'
import { useProject } from '../context/ProjectContext'
import { Play, Download, Database, ChevronRight, ChevronDown, Check, Loader2 } from 'lucide-react'
import { mockSchemas, mockQueries, executeMockQuery } from '../lib/mock-stratum'

export function Sql() {
  const { activeProject } = useProject()
  const [sqlText, setSqlText] = useState(mockQueries[0].sql)
  const [expandedTable, setExpandedTable] = useState<string | null>('papers')
  const [executing, setExecuting] = useState(false)

  // Dynamic query results state
  const [results, setResults] = useState(() => executeMockQuery(mockQueries[0].sql))
  const [exporting, setExporting] = useState(false)
  const [exportSuccess, setExportSuccess] = useState(false)

  const handleRunQuery = async () => {
    setExecuting(true)
    try {
      const response = await fetch(`/api/query?project=${activeProject}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: sqlText }),
      })
      const data = await response.json()
      if (!response.ok) {
        alert(data.error || 'Query failed')
        setExecuting(false)
        return
      }
      const rows = data as Record<string, string | number | boolean>[]
      const columns = rows.length > 0 ? Object.keys(rows[0]) : []
      setResults({ columns, rows })
    } catch (err: unknown) {
      alert('Failed to connect to database: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      setExecuting(false)
    }
  }

  const handleExportCSV = () => {
    setExporting(true)
    setTimeout(() => {
      const { columns, rows } = results
      if (columns.length === 0 || rows.length === 0) {
        setExporting(false)
        return
      }

      // Convert rows object array to CSV format
      const headers = columns.join(',')
      const csvLines = rows.map((row) =>
        columns
          .map((col) => {
            const val = row[col]
            return typeof val === 'string' ? `"${val.replace(/"/g, '""')}"` : val
          })
          .join(','),
      )
      const csvContent = [headers, ...csvLines].join('\n')

      // Create download link using Blob
      const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.setAttribute('download', `stratum_query_export_${Date.now()}.csv`)
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)

      setExporting(false)
      setExportSuccess(true)
      setTimeout(() => setExportSuccess(false), 3000)
    }, 500)
  }

  const handleQuerySelect = (querySql: string) => {
    setSqlText(querySql)
  }

  const toggleTable = (tableName: string) => {
    if (expandedTable === tableName) {
      setExpandedTable(null)
    } else {
      setExpandedTable(tableName)
    }
  }

  return (
    <div className="flex flex-col gap-8 w-full">
      {/* Header Row */}
      <div className="flex flex-col gap-1 border-b border-zinc-200 pb-5 dark:border-zinc-850">
        <h1 className="text-2xl font-mono font-bold tracking-tight text-zinc-950 dark:text-zinc-50 uppercase">
          SQL Explorer & Query Studio
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          Direct read-only access to compiled relational paper and contribution tables in DuckDB.
        </p>
      </div>

      {exportSuccess && (
        <div className="flex items-center gap-3 p-4 border border-green-200 bg-green-50/50 text-green-700 dark:border-green-800/40 dark:bg-green-950/20 dark:text-green-400 font-mono text-xs rounded">
          <Check className="h-4 w-4 shrink-0" />
          <span>[SUCCESS] CSV file compiled. Browser download initiated.</span>
        </div>
      )}

      {/* Main Split-Pane */}
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* Left Pane: Console Area */}
        <div className="lg:col-span-3 flex flex-col gap-4">
          <div className="flex items-center justify-between flex-wrap gap-2">
            <span className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500">
              Query Console
            </span>
            <div className="flex items-center gap-2">
              <label htmlFor="query-select" className="sr-only">
                Pre-configured Query
              </label>
              <select
                id="query-select"
                onChange={(e) => handleQuerySelect(e.target.value)}
                className="bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 px-3 py-1.5 rounded font-mono text-[11px] text-zinc-600 dark:text-zinc-400 focus:outline-none"
              >
                <option value="">-- Load Pre-configured Query --</option>
                {mockQueries.map((q, idx) => (
                  <option key={idx} value={q.sql}>
                    {q.label}
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="border border-zinc-200 dark:border-zinc-850 rounded overflow-hidden">
            <textarea
              aria-label="SQL Editor Console"
              value={sqlText}
              onChange={(e) => setSqlText(e.target.value)}
              className="w-full h-64 p-4 font-mono text-xs bg-zinc-950 text-zinc-300 leading-relaxed focus:outline-none focus:ring-0"
              spellCheck={false}
            />
            {/* Control Bar */}
            <div className="bg-zinc-50 dark:bg-zinc-900/40 border-t border-zinc-200 dark:border-zinc-850 p-3 flex items-center justify-between">
              <button
                onClick={handleRunQuery}
                disabled={executing || !sqlText.trim()}
                className="flex items-center gap-2 px-4 py-1.5 rounded font-mono text-xs font-bold uppercase tracking-wider select-none border transition-all cursor-pointer bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800 dark:border-zinc-200 hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {executing ? (
                  <>
                    <Loader2 className="h-3 w-3 animate-spin" />
                    <span>Executing...</span>
                  </>
                ) : (
                  <>
                    <Play className="h-3 w-3 fill-current" />
                    <span>Run Query</span>
                  </>
                )}
              </button>

              <button
                onClick={handleExportCSV}
                disabled={exporting || results.rows.length === 0}
                className="flex items-center gap-2 px-3 py-1.5 rounded border border-zinc-300 dark:border-zinc-800 bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-850 text-zinc-700 dark:text-zinc-300 transition text-xs font-mono select-none"
              >
                {exporting ? (
                  <>
                    <Loader2 className="h-3 w-3 animate-spin" />
                    <span>Compiling...</span>
                  </>
                ) : (
                  <>
                    <Download className="h-3.5 w-3.5" />
                    <span>Export CSV</span>
                  </>
                )}
              </button>
            </div>
          </div>
        </div>

        {/* Right Pane: Table Schema Reference */}
        <div className="lg:col-span-1 flex flex-col gap-4 border border-zinc-200 dark:border-zinc-850 bg-zinc-50/50 dark:bg-zinc-900/10 p-4 rounded overflow-y-auto">
          <div className="flex items-center gap-2 border-b border-zinc-200 dark:border-zinc-850 pb-2.5">
            <Database className="h-3.5 w-3.5 text-zinc-400" />
            <span className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500">
              Database Schema
            </span>
          </div>

          <div className="flex flex-col gap-2.5">
            {mockSchemas.map((table) => (
              <div key={table.name} className="flex flex-col">
                {/* Table Toggle Header */}
                <button
                  onClick={() => toggleTable(table.name)}
                  className="flex items-center justify-between w-full text-left font-mono text-xs font-bold hover:text-zinc-500 py-1"
                >
                  <span className="text-zinc-800 dark:text-zinc-200">{table.name}</span>
                  {expandedTable === table.name ? (
                    <ChevronDown className="h-3.5 w-3.5 text-zinc-400" />
                  ) : (
                    <ChevronRight className="h-3.5 w-3.5 text-zinc-400" />
                  )}
                </button>

                {/* Table Fields expanded panel */}
                {expandedTable === table.name && (
                  <div className="pl-3 border-l border-zinc-200 dark:border-zinc-800 flex flex-col gap-1 mt-1 font-mono text-[10px]">
                    {table.fields.map((field) => (
                      <div
                        key={field.name}
                        className="flex flex-col py-0.5"
                        title={field.description}
                      >
                        <div className="flex items-center justify-between">
                          <span className="text-zinc-700 dark:text-zinc-300 font-semibold">
                            {field.name}
                          </span>
                          <span className="text-zinc-400 uppercase">{field.type}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Query Results Section */}
      <div className="flex flex-col gap-3">
        <div className="flex items-center justify-between">
          <span className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500">
            Query Results ({results.rows.length} rows returned)
          </span>
        </div>

        <div className="w-full overflow-x-auto border border-zinc-200 dark:border-zinc-850 rounded">
          <table className="w-full border-collapse text-left font-mono text-xs">
            <thead>
              <tr className="bg-zinc-50 dark:bg-zinc-900 border-b border-zinc-200 dark:border-zinc-850 text-zinc-500">
                {results.columns.map((col) => (
                  <th
                    key={col}
                    className="p-3 border-r border-zinc-200 dark:border-zinc-850 font-bold uppercase select-none"
                  >
                    {col}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {results.rows.map((row, idx) => (
                <tr
                  key={idx}
                  className="border-b border-zinc-100 dark:border-zinc-900 hover:bg-zinc-50/50 dark:hover:bg-zinc-900/20 transition-colors"
                >
                  {results.columns.map((col) => (
                    <td
                      key={col}
                      className="p-3 border-r border-zinc-100 dark:border-zinc-900 truncate max-w-xs text-zinc-800 dark:text-zinc-200"
                    >
                      {typeof row[col] === 'boolean'
                        ? String(row[col]).toUpperCase()
                        : (row[col] ?? 'NULL')}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
export { Sql as default }
