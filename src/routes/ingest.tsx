// src/routes/ingest.tsx
import { useState } from 'react'
import { Save, Check, Loader2 } from 'lucide-react'

export function Ingest() {
  const [keywords, setKeywords] = useState(
    '("large language model" OR "generative ai" OR "transformer architecture" OR "agentic ai") AND ("reasoning" OR "eval" OR "benchmarking")',
  )
  const [topics, setTopics] = useState(
    `# Active topic filters (T + 5 digits format)\n# Lines starting with # are ignored\nT10098\nT10145\nT10221\nT11489`,
  )
  const [saving, setSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState(false)

  const handleSave = (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setSaveSuccess(false)

    // Simulate 3 seconds backend pipeline write
    setTimeout(() => {
      setSaving(false)
      setSaveSuccess(true)

      // Clear success banner after 4 seconds
      setTimeout(() => {
        setSaveSuccess(false)
      }, 4000)
    }, 3000)
  }

  return (
    <div className="flex flex-col gap-8 w-full max-w-5xl mx-auto">
      {/* Header Row */}
      <div className="flex flex-col gap-1 border-b border-zinc-200 pb-5 dark:border-zinc-850">
        <h1 className="text-2xl font-mono font-bold tracking-tight text-zinc-950 dark:text-zinc-50 uppercase">
          Pipeline Extraction Rules
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          Configure extraction parameters, keyword search queries, and targeted topics to filter the
          OpenAlex database collection pipeline.
        </p>
      </div>

      {saveSuccess && (
        <div className="flex items-center gap-3 p-4 border border-green-200 bg-green-50/50 text-green-700 dark:border-green-800/40 dark:bg-green-950/20 dark:text-green-400 font-mono text-xs rounded">
          <Check className="h-4 w-4 shrink-0" />
          <span>
            [SUCCESS] Configuration saved. assets/keywords.txt and assets/topics.txt successfully
            synchronized.
          </span>
        </div>
      )}

      {/* Configuration Form */}
      <form onSubmit={handleSave} className="flex flex-col gap-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Column 1: Keywords */}
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <label
                htmlFor="keywords-input"
                className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500"
              >
                Search Keywords (keywords.txt)
              </label>
              <span className="px-2 py-0.5 rounded border border-zinc-200 bg-zinc-50 text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900 font-mono text-[9px] font-semibold uppercase tracking-wider">
                BOOLEAN_LOGIC
              </span>
            </div>
            <div className="relative">
              <textarea
                id="keywords-input"
                disabled={saving}
                value={keywords}
                onChange={(e) => setKeywords(e.target.value)}
                className="w-full h-80 p-4 border border-zinc-200 dark:border-zinc-850 bg-zinc-50 dark:bg-zinc-900/10 font-mono text-xs leading-relaxed focus:outline-zinc-500 disabled:opacity-50 transition-opacity"
                placeholder="Enter boolean search queries..."
              />
            </div>
            <p className="text-[10px] text-zinc-400 font-sans">
              Parentheses and operators (AND, OR, NOT) are validated before matching on paper title
              and abstracts.
            </p>
          </div>

          {/* Column 2: Topics */}
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <label
                htmlFor="topics-input"
                className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500"
              >
                Target Topics (topics.txt)
              </label>
              <span className="px-2 py-0.5 rounded border border-zinc-200 bg-zinc-50 text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900 font-mono text-[9px] font-semibold uppercase tracking-wider">
                REGEX_MATCHING
              </span>
            </div>
            <div className="relative">
              <textarea
                id="topics-input"
                disabled={saving}
                value={topics}
                onChange={(e) => setTopics(e.target.value)}
                className="w-full h-80 p-4 border border-zinc-200 dark:border-zinc-850 bg-zinc-50 dark:bg-zinc-900/10 font-mono text-xs leading-relaxed focus:outline-zinc-500 disabled:opacity-50 transition-opacity"
                placeholder="Enter topic IDs, one per line..."
              />
            </div>
            <p className="text-[10px] text-zinc-400 font-sans">
              Validated using regex against standard topic code patterns (e.g. T12345).
            </p>
          </div>
        </div>

        {/* Action Button Row */}
        <div className="flex items-center justify-end border-t border-zinc-200 dark:border-zinc-850 pt-5 mt-4">
          <button
            type="submit"
            disabled={saving}
            className="flex items-center gap-2 px-4 py-2 border rounded font-mono text-xs font-bold uppercase tracking-wider select-none transition-all cursor-pointer bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800 dark:border-zinc-250 hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {saving ? (
              <>
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                <span>Ingesting Assets...</span>
              </>
            ) : (
              <>
                <Save className="h-3.5 w-3.5" />
                <span>Save Configuration</span>
              </>
            )}
          </button>
        </div>
      </form>
    </div>
  )
}
export { Ingest as default }
