import { useState, useEffect, useRef, useCallback } from 'react'
import { useProject } from '../context/ProjectContext'
import {
  Save,
  Check,
  Loader2,
  Upload,
  AlertCircle,
  Key,
  Calendar,
  Copy,
  Play,
  CheckCircle,
  Search,
} from 'lucide-react'

interface ScoredKeyword {
  term: string
  score: number
  selected: boolean
}

interface AppConfig {
  api?: {
    keys?: string[]
    email?: string
  }
  filters?: {
    date_from?: string
    date_to?: string
    doc_types?: string[]
  }
}

interface ConfigRevision {
  version: number
  timestamp: string
  label: string
  keywords: string
  topics: string
  anchors: string
}

export function Ingest() {
  const { activeProject } = useProject()
  // API Config States
  const [keywords, setKeywords] = useState('')
  const [topics, setTopics] = useState('')
  const [anchors, setAnchors] = useState('')
  const [appConfig, setAppConfig] = useState<AppConfig | null>(null)
  const [apiKeysStr, setApiKeysStr] = useState('')
  const [apiEmail, setApiEmail] = useState('')
  const [dateFrom, setDateFrom] = useState('2003-01-01')
  const [dateTo, setDateTo] = useState('2024-12-31')
  const [selectedDocTypes, setSelectedDocTypes] = useState<Record<string, boolean>>({
    article: true,
    review: true,
    'proceedings-article': true,
    preprint: false,
    'book-chapter': false,
    dataset: false,
  })
  const [configHistory, setConfigHistory] = useState<ConfigRevision[]>([])
  const [saveLabel, setSaveLabel] = useState('')

  // Upload States
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [uploadSuccess, setUploadSuccess] = useState(false)
  const [uploadedFilename, setUploadedFilename] = useState('')
  const [columns, setColumns] = useState<string[]>([])
  const [selectedTitleCol, setSelectedTitleCol] = useState('')
  const [selectedAbstractCol, setSelectedAbstractCol] = useState('')
  const [selectedDoiCol, setSelectedDoiCol] = useState('')

  // TF-IDF Extraction States
  const [topN, setTopN] = useState(50)
  const [ngramMin, setNgramMin] = useState(2)
  const [ngramMax, setNgramMax] = useState(3)
  const [minDF, setMinDF] = useState(2)
  const [maxDF] = useState(0.85)
  const [extracting, setExtracting] = useState(false)
  const [extractedKeywords, setExtractedKeywords] = useState<ScoredKeyword[]>([])

  // Validation States
  const [validating, setValidating] = useState(false)
  const [queryValid, setQueryValid] = useState<boolean | null>(null)
  const [queryErrors, setQueryErrors] = useState<string[]>([])

  // OpenAlex Count States
  const [checkingCount, setCheckingCount] = useState(false)
  const [openalexCount, setOpenalexCount] = useState<number | null>(null)
  const [anchorsCount, setAnchorsCount] = useState<number | null>(null)
  const [anchorsTotal, setAnchorsTotal] = useState<number | null>(null)
  const [anchorsMatched, setAnchorsMatched] = useState<number | null>(null)
  const [anchorsMissing, setAnchorsMissing] = useState<string[]>([])

  // Save States
  const [saving, setSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState(false)

  const fileInputRef = useRef<HTMLInputElement>(null)

  const fetchConfig = useCallback(async () => {
    try {
      const response = await fetch(`/api/config?project=${activeProject}`)
      if (response.ok) {
        const data = await response.json()
        const cfg = data.config
        setAppConfig(cfg)
        setKeywords(data.keywords || '')
        setTopics(data.topics || '')
        setAnchors(data.anchors || '')
        setConfigHistory(data.history || [])

        if (cfg) {
          setApiKeysStr(cfg.api?.keys?.join(', ') || '')
          setApiEmail(cfg.api?.email || '')
          setDateFrom(cfg.filters?.date_from || '2003-01-01')
          setDateTo(cfg.filters?.date_to || '2024-12-31')

          const types = cfg.filters?.doc_types || []
          const typeMap: Record<string, boolean> = {
            article: false,
            review: false,
            'proceedings-article': false,
            preprint: false,
            'book-chapter': false,
            dataset: false,
          }
          types.forEach((t: string) => {
            typeMap[t] = true
          })
          setSelectedDocTypes(typeMap)
        }
      }
    } catch (err) {
      console.error('Failed to load configuration:', err)
    }
  }, [activeProject])

  // Load Initial Config
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect
    fetchConfig()
  }, [fetchConfig])

  // File Upload Handler
  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      setFile(e.target.files[0])
      setUploadSuccess(false)
      setExtractedKeywords([])
    }
  }

  const triggerUpload = async () => {
    if (!file) return
    setUploading(true)
    setUploadSuccess(false)
    const formData = new FormData()
    formData.append('file', file)

    try {
      const response = await fetch(`/api/upload?project=${activeProject}`, {
        method: 'POST',
        body: formData,
      })
      if (response.ok) {
        const data = await response.json()
        setUploadedFilename(data.filename)
        setColumns(data.columns || [])
        setUploadSuccess(true)

        // Guess columns
        const cols = data.columns || []
        const tCol = cols.find((c: string) => /title/i.test(c)) || cols[0] || ''
        const aCol =
          cols.find((c: string) => /abstract|summary/i.test(c)) || cols[1] || cols[0] || ''
        const dCol = cols.find((c: string) => /doi/i.test(c)) || cols[2] || cols[0] || ''
        setSelectedTitleCol(tCol)
        setSelectedAbstractCol(aCol)
        setSelectedDoiCol(dCol)
      } else {
        const errData = await response.json()
        alert('Upload failed: ' + (errData.error || response.statusText))
      }
    } catch (err: unknown) {
      alert('Upload failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      setUploading(false)
    }
  }

  // TF-IDF Extraction Handler
  const handleExtractKeywords = async () => {
    if (!uploadedFilename) return
    setExtracting(true)

    try {
      const response = await fetch(`/api/tfidf?project=${activeProject}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          filename: uploadedFilename,
          title_column: selectedTitleCol,
          abstract_column: selectedAbstractCol,
          doi_column: selectedDoiCol,
          top_n: Number(topN),
          ngram_min: Number(ngramMin),
          ngram_max: Number(ngramMax),
          min_df: Number(minDF),
          max_df: Number(maxDF),
        }),
      })

      if (response.ok) {
        const data = await response.json()
        const list = (data.keywords || []).map((k: { term: string; score: number }) => ({
          ...k,
          selected: true,
        }))
        setExtractedKeywords(list)
        setAnchorsCount(data.anchors_count || 0)
        fetchConfig() // Reload config to reflect newly saved anchors in textarea
      } else {
        const errData = await response.json()
        alert('Extraction failed: ' + (errData.error || response.statusText))
      }
    } catch (err: unknown) {
      alert('Extraction failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      setExtracting(false)
    }
  }

  const toggleKeywordSelection = (index: number) => {
    setExtractedKeywords((prev) =>
      prev.map((k, i) => (i === index ? { ...k, selected: !k.selected } : k)),
    )
  }

  const selectAllKeywords = (selected: boolean) => {
    setExtractedKeywords((prev) => prev.map((k) => ({ ...k, selected })))
  }

  const appendKeywordsToQuery = () => {
    const selectedTerms = extractedKeywords.filter((k) => k.selected).map((k) => `"${k.term}"`)

    if (selectedTerms.length === 0) return

    const joined = selectedTerms.join(' OR ')
    setKeywords((prev) => {
      const trimmed = prev.trim()
      if (!trimmed) {
        return `(\n  ${joined}\n)`
      }
      // Append to the existing query
      if (trimmed.endsWith(')')) {
        return trimmed.slice(0, -1) + `\n  OR ${joined}\n)`
      }
      return trimmed + ` OR (\n  ${joined}\n)`
    })
  }

  // Query Validation Handler
  const handleValidateQuery = async () => {
    if (!keywords.trim()) {
      setQueryValid(null)
      setQueryErrors([])
      return
    }
    setValidating(true)
    setQueryValid(null)

    try {
      const response = await fetch('/api/query/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query: keywords }),
      })
      if (response.ok) {
        const data = await response.json()
        setQueryValid(data.valid)
        setQueryErrors(data.errors || [])
      } else {
        alert('Failed to validate query')
      }
    } catch (err: unknown) {
      alert('Connection error: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      setValidating(false)
    }
  }

  // Fetch Real Count Handler
  const handleGetOpenAlexCount = async () => {
    if (!keywords.trim()) {
      alert('Please enter a query first.')
      return
    }
    setCheckingCount(true)
    setOpenalexCount(null)
    setAnchorsTotal(null)
    setAnchorsMatched(null)
    setAnchorsMissing([])

    const keysList = apiKeysStr
      .split(',')
      .map((k) => k.trim())
      .filter(Boolean)
    const topicsList = topics
      .split('\n')
      .map((t) => t.trim())
      .filter((t) => t && !t.startsWith('#'))
    const docTypesList = Object.keys(selectedDocTypes).filter((k) => selectedDocTypes[k])

    try {
      const response = await fetch(`/api/openalex/count?project=${activeProject}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: keywords,
          keys: keysList,
          email: apiEmail,
          date_from: dateFrom,
          date_to: dateTo,
          doc_types: docTypesList,
          topics: topicsList,
        }),
      })

      if (response.ok) {
        const data = await response.json()
        setOpenalexCount(data.count)
        setAnchorsTotal(data.anchors_total || 0)
        setAnchorsMatched(data.anchors_matched || 0)
        setAnchorsMissing(data.anchors_missing || [])
      } else {
        const errData = await response.json()
        alert('Count failed: ' + (errData.error || response.statusText))
      }
    } catch (err: unknown) {
      alert('Count request failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      setCheckingCount(false)
    }
  }

  // Save Config Handler
  const handleSaveConfig = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setSaveSuccess(false)

    const keysList = apiKeysStr
      .split(',')
      .map((k) => k.trim())
      .filter(Boolean)
    const docTypesList = Object.keys(selectedDocTypes).filter((k) => selectedDocTypes[k])

    const updatedConfig = {
      ...appConfig,
      api: {
        ...appConfig?.api,
        keys: keysList,
        email: apiEmail,
      },
      filters: {
        ...appConfig?.filters,
        date_from: dateFrom,
        date_to: dateTo,
        doc_types: docTypesList,
      },
    }

    try {
      const response = await fetch(`/api/config?project=${activeProject}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          config: updatedConfig,
          keywords,
          topics,
          anchors,
          label: saveLabel,
        }),
      })

      if (response.ok) {
        setSaveSuccess(true)
        setSaveLabel('')
        fetchConfig()
        setTimeout(() => setSaveSuccess(false), 4000)
      } else {
        alert('Failed to save configuration')
      }
    } catch (err: unknown) {
      alert('Save failed: ' + (err instanceof Error ? err.message : String(err)))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-8 w-full max-w-7xl mx-auto">
      {/* Page Header */}
      <div className="flex flex-col gap-1 border-b border-zinc-200 pb-5 dark:border-zinc-850">
        <h1 className="text-2xl font-mono font-bold tracking-tight text-zinc-950 dark:text-zinc-50 uppercase">
          Pipeline Setup & Keywords Studio
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          Upload local paper catalogs to extract TF-IDF terms, refine boolean queries, configure
          multiple keys, and validate query volumes.
        </p>
      </div>

      {saveSuccess && (
        <div className="flex items-center gap-3 p-4 border border-green-200 bg-green-50/50 text-green-700 dark:border-green-800/40 dark:bg-green-950/20 dark:text-green-400 font-mono text-xs rounded">
          <CheckCircle className="h-4 w-4 shrink-0" />
          <span>
            [SUCCESS] Configuration successfully saved. Keywords, topics, and API keys updated.
          </span>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start">
        {/* LEFT COLUMN: TF-IDF Extraction (cols: 5) */}
        <div className="lg:col-span-5 flex flex-col gap-6 border border-zinc-200 dark:border-zinc-850 p-5 bg-zinc-50/20 dark:bg-zinc-950/20 rounded">
          <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-850 pb-2">
            <span className="text-sm font-mono font-bold uppercase tracking-wider text-zinc-850 dark:text-zinc-200">
              TF-IDF Keyword Extraction
            </span>
            <span className="text-[10px] font-mono text-zinc-400">LOCAL CATALOGS</span>
          </div>

          {/* Step 1: File Uploader */}
          <div className="flex flex-col gap-2.5">
            <span className="text-xs font-mono font-bold uppercase text-zinc-500">
              1. Select CSV or Excel File
            </span>
            <div className="flex gap-2">
              <input
                type="file"
                accept=".csv,.xlsx,.xls"
                className="hidden"
                ref={fileInputRef}
                onChange={handleFileSelect}
              />
              <button
                onClick={() => fileInputRef.current?.click()}
                className="flex-1 flex items-center justify-center gap-2 px-3 py-3 border border-dashed border-zinc-300 dark:border-zinc-800 rounded font-mono text-xs cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-900/60 transition"
              >
                <Upload className="h-4 w-4 text-zinc-400" />
                <span className="text-zinc-700 dark:text-zinc-300 truncate">
                  {file ? file.name : 'Choose catalog file...'}
                </span>
              </button>
              {file && (
                <button
                  onClick={triggerUpload}
                  disabled={uploading}
                  className="px-4 py-2 border rounded font-mono text-xs font-bold uppercase tracking-wider bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50 cursor-pointer"
                >
                  {uploading ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Upload'}
                </button>
              )}
            </div>
            {uploadSuccess && (
              <span className="text-[10px] font-mono text-green-600 dark:text-green-400 flex items-center gap-1">
                <Check className="h-3 w-3" /> Catalog parsed. {columns.length} columns detected.
              </span>
            )}
          </div>

          {/* Step 2: Column Selection & Parameters (Only after upload) */}
          {uploadSuccess && (
            <div className="flex flex-col gap-4 border-t border-zinc-200 dark:border-zinc-850 pt-4">
              <span className="text-xs font-mono font-bold uppercase text-zinc-500">
                2. Configure Text Mining
              </span>

              <div className="grid grid-cols-3 gap-2.5">
                <div className="flex flex-col gap-1">
                  <label className="text-[10px] font-mono uppercase text-zinc-400">
                    Title Column
                  </label>
                  <select
                    value={selectedTitleCol}
                    onChange={(e) => setSelectedTitleCol(e.target.value)}
                    className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs w-full"
                  >
                    {columns.map((c) => (
                      <option key={c} value={c}>
                        {c}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex flex-col gap-1">
                  <label className="text-[10px] font-mono uppercase text-zinc-400">
                    Abstract Column
                  </label>
                  <select
                    value={selectedAbstractCol}
                    onChange={(e) => setSelectedAbstractCol(e.target.value)}
                    className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs w-full"
                  >
                    {columns.map((c) => (
                      <option key={c} value={c}>
                        {c}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="flex flex-col gap-1">
                  <label className="text-[10px] font-mono uppercase text-zinc-400 truncate">
                    DOI / Anchor Column
                  </label>
                  <select
                    value={selectedDoiCol}
                    onChange={(e) => setSelectedDoiCol(e.target.value)}
                    className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs w-full"
                  >
                    {columns.map((c) => (
                      <option key={c} value={c}>
                        {c}
                      </option>
                    ))}
                  </select>
                </div>
              </div>

              {/* Advanced TF-IDF params */}
              <div className="grid grid-cols-3 gap-2 border-t border-zinc-150 dark:border-zinc-850/50 pt-3 mt-1">
                <div className="flex flex-col gap-1">
                  <label className="text-[9px] font-mono uppercase text-zinc-400">
                    N-gram Range
                  </label>
                  <div className="flex items-center gap-1 font-mono text-xs">
                    <input
                      type="number"
                      value={ngramMin}
                      onChange={(e) => setNgramMin(Number(e.target.value))}
                      className="w-full bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1 rounded text-center"
                    />
                    <span className="text-zinc-400">-</span>
                    <input
                      type="number"
                      value={ngramMax}
                      onChange={(e) => setNgramMax(Number(e.target.value))}
                      className="w-full bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1 rounded text-center"
                    />
                  </div>
                </div>
                <div className="flex flex-col gap-1">
                  <label className="text-[9px] font-mono uppercase text-zinc-400">
                    Min DF (Docs)
                  </label>
                  <input
                    type="number"
                    value={minDF}
                    onChange={(e) => setMinDF(Number(e.target.value))}
                    className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1 rounded font-mono text-xs text-center"
                  />
                </div>
                <div className="flex flex-col gap-1">
                  <label className="text-[9px] font-mono uppercase text-zinc-400">
                    Top N terms
                  </label>
                  <input
                    type="number"
                    value={topN}
                    onChange={(e) => setTopN(Number(e.target.value))}
                    className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-850 p-1 rounded font-mono text-xs text-center"
                  />
                </div>
              </div>

              <button
                onClick={handleExtractKeywords}
                disabled={extracting}
                className="mt-2 w-full flex items-center justify-center gap-2 px-3 py-2 border rounded font-mono text-xs font-bold uppercase bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800 hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50 cursor-pointer"
              >
                {extracting ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    <span>Extracting Terms...</span>
                  </>
                ) : (
                  <span>Extract Scored Keywords</span>
                )}
              </button>
              {anchorsCount !== null && anchorsCount > 0 && (
                <span className="text-[10px] font-mono text-green-600 dark:text-green-400 flex items-center gap-1 mt-1">
                  <Check className="h-3 w-3" /> Extracted {anchorsCount} anchor DOIs to anchor.txt.
                </span>
              )}
            </div>
          )}

          {/* Step 3: Extracted Terms List */}
          {extractedKeywords.length > 0 && (
            <div className="flex flex-col gap-3 border-t border-zinc-200 dark:border-zinc-850 pt-4">
              <div className="flex items-center justify-between">
                <span className="text-xs font-mono font-bold uppercase text-zinc-500">
                  3. Select Scored Keywords ({extractedKeywords.filter((k) => k.selected).length}{' '}
                  selected)
                </span>
                <div className="flex gap-2 text-[10px] font-mono">
                  <button
                    onClick={() => selectAllKeywords(true)}
                    className="text-zinc-500 hover:underline"
                  >
                    All
                  </button>
                  <span className="text-zinc-300">|</span>
                  <button
                    onClick={() => selectAllKeywords(false)}
                    className="text-zinc-500 hover:underline"
                  >
                    None
                  </button>
                </div>
              </div>

              {/* Scrollable checklist */}
              <div className="max-h-60 overflow-y-auto border border-zinc-200 dark:border-zinc-850 rounded bg-white dark:bg-zinc-950/40 p-1 flex flex-col gap-0.5">
                {extractedKeywords.map((kw, i) => (
                  <label
                    key={kw.term}
                    className={`flex items-center justify-between p-2 rounded cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-900/60 font-mono text-xs select-none ${kw.selected ? 'bg-zinc-50/50 dark:bg-zinc-900/20' : ''}`}
                  >
                    <div className="flex items-center gap-2">
                      <input
                        type="checkbox"
                        checked={kw.selected}
                        onChange={() => toggleKeywordSelection(i)}
                        className="rounded border-zinc-300 text-zinc-900 focus:ring-0 cursor-pointer"
                      />
                      <span className="font-semibold text-zinc-800 dark:text-zinc-200">
                        {kw.term}
                      </span>
                    </div>
                    <span className="text-[10px] text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-950/20 px-1.5 py-0.5 rounded font-mono font-bold">
                      {kw.score.toFixed(4)}
                    </span>
                  </label>
                ))}
              </div>

              <button
                onClick={appendKeywordsToQuery}
                className="w-full flex items-center justify-center gap-2 px-3 py-2 border border-zinc-300 dark:border-zinc-800 rounded bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-850 text-zinc-700 dark:text-zinc-300 font-mono text-xs cursor-pointer select-none font-bold uppercase"
              >
                <Copy className="h-4 w-4" />
                Append Selected to Query
              </button>
            </div>
          )}
        </div>

        {/* RIGHT COLUMN: Query Builder, Validator, Estimation & Config (cols: 7) */}
        <form onSubmit={handleSaveConfig} className="lg:col-span-7 flex flex-col gap-6">
          {/* Section A: Boolean Keywords Editor */}
          <div className="flex flex-col gap-3 border border-zinc-200 dark:border-zinc-850 p-5 rounded">
            <div className="flex items-center justify-between">
              <label
                htmlFor="keywords-input"
                className="text-sm font-mono font-bold uppercase tracking-wider text-zinc-800 dark:text-zinc-200"
              >
                Boolean Keywords Query (keywords.txt)
              </label>
              <span className="px-2 py-0.5 rounded border border-zinc-200 bg-zinc-50 text-zinc-600 dark:border-zinc-800 dark:bg-zinc-900 font-mono text-[9px] font-semibold uppercase">
                Syntax check
              </span>
            </div>

            <textarea
              id="keywords-input"
              disabled={saving}
              value={keywords}
              onChange={(e) => setKeywords(e.target.value)}
              className="w-full h-64 p-4 border border-zinc-200 dark:border-zinc-850 bg-zinc-950 text-zinc-300 font-mono text-xs leading-relaxed focus:outline-none focus:ring-0 rounded"
              placeholder="Enter boolean query using OR / AND / NOT operators..."
            />

            {/* Live syntax checker panel */}
            <div className="flex items-center justify-between flex-wrap gap-3 mt-1.5">
              <button
                type="button"
                onClick={handleValidateQuery}
                disabled={validating || !keywords.trim()}
                className="flex items-center gap-2 px-3.5 py-1.5 border border-zinc-300 dark:border-zinc-800 rounded bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-850 text-zinc-700 dark:text-zinc-300 font-mono text-xs cursor-pointer disabled:opacity-50 select-none font-bold uppercase"
              >
                {validating ? (
                  <>
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    <span>Validating...</span>
                  </>
                ) : (
                  <>
                    <Play className="h-3 w-3 fill-current" />
                    <span>Validate Query Syntax</span>
                  </>
                )}
              </button>

              <div className="flex items-center">
                {queryValid === true && (
                  <span className="text-xs font-mono text-green-600 dark:text-green-400 flex items-center gap-1.5">
                    <CheckCircle className="h-4 w-4" /> [VALID] Boolean grammar checks passed.
                  </span>
                )}
                {queryValid === false && (
                  <span className="text-xs font-mono text-red-600 dark:text-red-400 flex items-center gap-1.5">
                    <AlertCircle className="h-4 w-4" /> [ERROR]{' '}
                    {queryErrors[0] || 'Syntax validation failed.'}
                  </span>
                )}
              </div>
            </div>
          </div>

          {/* Section B: OpenAlex API Keys Configuration */}
          <div className="flex flex-col gap-4 border border-zinc-200 dark:border-zinc-850 p-5 rounded">
            <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-850 pb-2">
              <span className="text-sm font-mono font-bold uppercase tracking-wider text-zinc-850 dark:text-zinc-250">
                API Keys & Polite Pool Configuration
              </span>
              <Key className="h-4 w-4 text-zinc-400" />
            </div>

            <div className="flex flex-col gap-3">
              <div className="flex flex-col gap-1.5">
                <label className="text-[11px] font-mono uppercase text-zinc-400">
                  OpenAlex API Keys (Comma-separated for rotation)
                </label>
                <input
                  type="text"
                  value={apiKeysStr}
                  onChange={(e) => setApiKeysStr(e.target.value)}
                  placeholder="Key1, Key2, Key3..."
                  className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-2 rounded font-mono text-xs focus:outline-none"
                />
                <span className="text-[9px] text-zinc-400 font-sans">
                  If multiple keys are set, Stratum rotates queries across them and automatically
                  sets aside keys that encounter quota exceptions.
                </span>
              </div>

              <div className="flex flex-col gap-1.5 mt-1">
                <label className="text-[11px] font-mono uppercase text-zinc-400">
                  Polite Pool Email Address (Contact UserAgent)
                </label>
                <input
                  type="email"
                  value={apiEmail}
                  onChange={(e) => setApiEmail(e.target.value)}
                  placeholder="your@email.com"
                  className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-2 rounded font-mono text-xs focus:outline-none"
                />
              </div>
            </div>
          </div>

          {/* Section C: Query Filters & Size Estimation */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 border border-zinc-200 dark:border-zinc-850 p-5 rounded">
            {/* Left Sub-column: Custom Filters */}
            <div className="flex flex-col gap-4">
              <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-850 pb-2">
                <span className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500">
                  API Query Filters
                </span>
                <Calendar className="h-3.5 w-3.5 text-zinc-400" />
              </div>

              {/* Date Boundaries */}
              <div className="grid grid-cols-2 gap-3">
                <div className="flex flex-col gap-1">
                  <label className="text-[10px] font-mono uppercase text-zinc-400">Date From</label>
                  <input
                    type="text"
                    value={dateFrom}
                    onChange={(e) => setDateFrom(e.target.value)}
                    className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs text-center"
                  />
                </div>
                <div className="flex flex-col gap-1">
                  <label className="text-[10px] font-mono uppercase text-zinc-400">Date To</label>
                  <input
                    type="text"
                    value={dateTo}
                    onChange={(e) => setDateTo(e.target.value)}
                    className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs text-center"
                  />
                </div>
              </div>

              {/* Doc Types */}
              <div className="flex flex-col gap-2">
                <label className="text-[10px] font-mono uppercase text-zinc-400">
                  Document Types
                </label>
                <div className="grid grid-cols-2 gap-1.5">
                  {Object.keys(selectedDocTypes).map((t) => (
                    <label
                      key={t}
                      className="flex items-center gap-2 font-mono text-xs select-none"
                    >
                      <input
                        type="checkbox"
                        checked={selectedDocTypes[t]}
                        onChange={() =>
                          setSelectedDocTypes((prev) => ({
                            ...prev,
                            [t]: !prev[t],
                          }))
                        }
                        className="rounded border-zinc-300 text-zinc-950 focus:ring-0"
                      />
                      <span className="text-zinc-700 dark:text-zinc-300">{t}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>

            {/* Right Sub-column: Volume Estimation count */}
            <div className="flex flex-col gap-4 border-t md:border-t-0 md:border-l border-zinc-200 dark:border-zinc-850 pt-4 md:pt-0 md:pl-6 justify-between">
              <div className="flex flex-col gap-2.5">
                <span className="text-xs font-mono font-bold uppercase text-zinc-500">
                  Size Estimation count
                </span>
                <p className="text-[11px] text-zinc-400 font-sans leading-relaxed">
                  Query the live OpenAlex database with your current query expression and filters.
                  Only the estimated work count will be loaded for safety checks.
                </p>

                {/* Count badge */}
                <div className="min-h-[6rem] py-4 px-3 border border-zinc-200 dark:border-zinc-850 bg-zinc-50/50 dark:bg-zinc-900/10 rounded flex flex-col items-center justify-center gap-1 mt-2.5">
                  {checkingCount ? (
                    <Loader2 className="h-5 w-5 animate-spin text-zinc-400" />
                  ) : openalexCount !== null ? (
                    <div className="w-full flex flex-col items-center gap-3">
                      <div className="flex flex-col items-center justify-center gap-1">
                        <span className="text-2xl font-mono font-bold text-zinc-950 dark:text-zinc-50">
                          {openalexCount.toLocaleString()}
                        </span>
                        <span className="text-[9px] font-mono text-green-600 dark:text-green-400 uppercase tracking-widest">
                          MATCHING PAPERS
                        </span>
                      </div>

                      {anchorsTotal !== null && anchorsTotal > 0 && (
                        <div className="border-t border-zinc-200 dark:border-zinc-850 pt-3 mt-1 w-full flex flex-col gap-1.5">
                          <div className="flex justify-between items-center text-[10px] font-mono uppercase tracking-wider text-zinc-400">
                            <span>Anchor Paper Match</span>
                            <span className="font-bold text-zinc-700 dark:text-zinc-350">
                              {anchorsMatched} / {anchorsTotal} (
                              {anchorsTotal > 0
                                ? Math.round((anchorsMatched! / anchorsTotal) * 100)
                                : 0}
                              %)
                            </span>
                          </div>
                          <div className="w-full bg-zinc-250 dark:bg-zinc-800 h-1.5 rounded-full overflow-hidden">
                            <div
                              className="bg-green-500 h-full rounded-full transition-all duration-300"
                              style={{
                                width: `${anchorsTotal > 0 ? (anchorsMatched! / anchorsTotal) * 100 : 0}%`,
                              }}
                            />
                          </div>
                          {anchorsMissing.length > 0 && (
                            <div className="mt-1 flex flex-col gap-1">
                              <span className="text-[9px] font-mono text-amber-600 dark:text-amber-400 uppercase tracking-wide flex items-center gap-1 font-semibold">
                                <AlertCircle className="h-3 w-3 shrink-0" /> Missing{' '}
                                {anchorsMissing.length} anchors from results
                              </span>
                              <div className="bg-amber-50/50 dark:bg-amber-950/10 border border-amber-200/50 dark:border-amber-900/30 p-2 rounded text-[10px] font-mono text-zinc-500 dark:text-zinc-400 max-h-24 overflow-y-auto w-full text-left">
                                {anchorsMissing.map((doi) => (
                                  <div key={doi} className="truncate">
                                    • {doi}
                                  </div>
                                ))}
                              </div>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  ) : (
                    <span className="text-xs font-mono text-zinc-400 uppercase tracking-wider">
                      No Estimation Yet
                    </span>
                  )}
                </div>
              </div>

              <button
                type="button"
                onClick={handleGetOpenAlexCount}
                disabled={checkingCount || !keywords.trim()}
                className="mt-3 w-full flex items-center justify-center gap-2 px-3 py-2 border rounded font-mono text-xs font-bold uppercase bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800 hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50 cursor-pointer"
              >
                <Search className="h-3.5 w-3.5" />
                Get OpenAlex Count
              </button>
            </div>
          </div>

          {/* Bottom Config Fields: Topics & Anchors (Standard fields) */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6 border border-zinc-200 dark:border-zinc-850 p-5 rounded">
            <div className="flex flex-col gap-2">
              <label
                htmlFor="topics-input"
                className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500"
              >
                Target Topics (topics.txt)
              </label>
              <textarea
                id="topics-input"
                disabled={saving}
                value={topics}
                onChange={(e) => setTopics(e.target.value)}
                className="w-full h-40 p-3 border border-zinc-200 dark:border-zinc-850 bg-zinc-50 dark:bg-zinc-900/10 font-mono text-xs leading-relaxed focus:outline-none rounded"
                placeholder="T10020..."
              />
            </div>
            <div className="flex flex-col gap-2">
              <label
                htmlFor="anchors-input"
                className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500"
              >
                Anchor DOIs (anchor.txt)
              </label>
              <textarea
                id="anchors-input"
                disabled={saving}
                value={anchors}
                onChange={(e) => setAnchors(e.target.value)}
                className="w-full h-40 p-3 border border-zinc-200 dark:border-zinc-850 bg-zinc-50 dark:bg-zinc-900/10 font-mono text-xs leading-relaxed focus:outline-none rounded"
                placeholder="10.1016/j.renene..."
              />
            </div>
          </div>

          {/* Revisions History Panel */}
          {configHistory.length > 0 && (
            <div className="flex flex-col gap-4 border border-zinc-200 dark:border-zinc-850 p-5 rounded mt-6">
              <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-850 pb-2">
                <span className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500">
                  Configuration Revisions (Source Control)
                </span>
                <span className="text-[10px] font-mono text-zinc-400">PROJECT HISTORY</span>
              </div>
              <div className="max-h-60 overflow-y-auto flex flex-col gap-2 pr-2">
                {configHistory
                  .slice()
                  .reverse()
                  .map((rev) => (
                    <div
                      key={rev.version}
                      className="flex flex-col sm:flex-row sm:items-center justify-between p-3 border border-zinc-200 dark:border-zinc-850 bg-zinc-50/20 dark:bg-zinc-950/20 rounded font-mono text-xs gap-3"
                    >
                      <div className="flex flex-col gap-1">
                        <div className="flex items-center gap-2">
                          <span className="px-1.5 py-0.5 rounded bg-zinc-200 dark:bg-zinc-800 text-[10px] font-bold">
                            v{rev.version}
                          </span>
                          <span className="font-semibold text-zinc-800 dark:text-zinc-200">
                            {rev.label}
                          </span>
                        </div>
                        <span className="text-[10px] text-zinc-400">
                          Saved: {new Date(rev.timestamp).toLocaleString()}
                        </span>
                      </div>
                      <button
                        type="button"
                        onClick={() => {
                          setKeywords(rev.keywords || '')
                          setTopics(rev.topics || '')
                          setAnchors(rev.anchors || '')
                          alert(
                            `Loaded revision v${rev.version} to editor. Click "Save Configuration" to apply changes.`,
                          )
                        }}
                        className="px-2.5 py-1.5 border border-zinc-350 dark:border-zinc-800 rounded bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-850 text-zinc-700 dark:text-zinc-300 font-mono text-[10px] font-bold uppercase cursor-pointer self-start sm:self-center shrink-0"
                      >
                        Restore to Editor
                      </button>
                    </div>
                  ))}
              </div>
            </div>
          )}

          {/* Action Buttons: Save Config */}
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center justify-between border-t border-zinc-200 dark:border-zinc-850 pt-5 mt-2 gap-4">
            <div className="flex-1 flex flex-col gap-1 max-w-md">
              <label htmlFor="save-label" className="text-[10px] font-mono uppercase text-zinc-400">
                Revision Label / Commit Message (Optional)
              </label>
              <input
                id="save-label"
                type="text"
                disabled={saving}
                value={saveLabel}
                onChange={(e) => setSaveLabel(e.target.value)}
                placeholder="e.g. Added quantum keywords, fixed topics..."
                className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-2 rounded font-mono text-xs focus:outline-none"
              />
            </div>
            <button
              type="submit"
              disabled={saving}
              className="flex items-center justify-center gap-2 px-5 py-2.5 border rounded font-mono text-xs font-bold uppercase tracking-wider select-none transition-all cursor-pointer bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800 hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50 self-end sm:self-center shrink-0"
            >
              {saving ? (
                <>
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  <span>Saving configuration...</span>
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
    </div>
  )
}
export { Ingest as default }
