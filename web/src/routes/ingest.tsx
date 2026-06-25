import { useState, useEffect, useRef, useCallback, useMemo } from 'react'
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
  PieChart,
  Sliders,
  Bell,
  FileText,
  X,
  Activity,
  Maximize2,
  Minimize2,
  Plus,
  Trash2,
  Download,
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

interface SearchableListEditorProps {
  id: string
  label: string
  value: string
  onChange: (val: string) => void
  placeholder: string
  disabled?: boolean
  maxCollapsedItems?: number
  validate?: (val: string) => { valid: boolean; error?: string; normalized?: string }
}

function SearchableListEditor({
  id,
  label,
  value,
  onChange,
  placeholder,
  disabled,
  maxCollapsedItems = 10,
  validate,
}: SearchableListEditorProps) {
  const [activeTab, setActiveTab] = useState<'list' | 'raw'>('list')
  const [searchQuery, setSearchQuery] = useState('')
  const [newItem, setNewItem] = useState('')
  const [isExpanded, setIsExpanded] = useState(false)
  const [validationError, setValidationError] = useState<string | null>(null)

  // Clear validation error when user types a new value, toggles tabs, or changes expanded state
  useEffect(() => {
    setValidationError(null)
  }, [newItem, activeTab, isExpanded])

  // Parse lines
  const items = useMemo(() => {
    return value
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
  }, [value])

  const filteredItems = useMemo(() => {
    if (!searchQuery.trim()) return items
    return items.filter((item) =>
      item.toLowerCase().includes(searchQuery.toLowerCase())
    )
  }, [items, searchQuery])

  const invalidLines = useMemo(() => {
    if (!validate) return []
    return value
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
      .filter((line) => !validate(line).valid)
  }, [value, validate])

  const handleAddItem = (e?: React.FormEvent | React.KeyboardEvent) => {
    if (e) e.preventDefault()
    const cleaned = newItem.trim()
    if (!cleaned) return

    let itemToAdd = cleaned
    if (validate) {
      const res = validate(cleaned)
      if (!res.valid) {
        setValidationError(res.error || 'Invalid format')
        return
      }
      itemToAdd = res.normalized || cleaned
    }

    setValidationError(null)

    if (items.includes(itemToAdd)) {
      setNewItem('')
      return // Avoid duplicates if already present
    }
    const newValue = [...items, itemToAdd].join('\n')
    onChange(newValue)
    setNewItem('')
  }

  const handleDeleteItem = (itemToDelete: string) => {
    const newValue = items.filter((item) => item !== itemToDelete).join('\n')
    onChange(newValue)
  }

  const handleClearAll = () => {
    if (window.confirm(`Are you sure you want to clear all ${label.toLowerCase()}?`)) {
      onChange('')
    }
  }

  const handleRawBlur = () => {
    if (!validate) return
    const normalizedLines = value
      .split('\n')
      .map((line) => {
        const trimmed = line.trim()
        if (!trimmed) return ''
        const res = validate(trimmed)
        return res.valid ? (res.normalized || trimmed) : line
      })
    onChange(normalizedLines.join('\n'))
  }

  const EditorContent = (isModal: boolean) => {
    const visibleItems = isModal ? filteredItems : filteredItems.slice(0, maxCollapsedItems)
    return (
      <div className="flex flex-col h-full gap-3">
        {/* Header Controls */}
        <div className="flex items-center justify-between gap-2 border-b border-zinc-150 dark:border-zinc-800/80 pb-2">
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => setActiveTab('list')}
              className={`px-3 py-1 text-xs font-mono font-bold uppercase rounded transition-colors cursor-pointer ${
                activeTab === 'list'
                  ? 'bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100'
                  : 'text-zinc-400 hover:text-zinc-650 dark:hover:text-zinc-350'
              }`}
            >
              List ({items.length})
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('raw')}
              className={`px-3 py-1 text-xs font-mono font-bold uppercase rounded transition-colors cursor-pointer ${
                activeTab === 'raw'
                  ? 'bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100'
                  : 'text-zinc-400 hover:text-zinc-650 dark:hover:text-zinc-350'
              }`}
            >
              Raw Editor
            </button>
          </div>

          <div className="flex items-center gap-3">
            {items.length > 0 && (
              <button
                type="button"
                onClick={handleClearAll}
                className="text-[10px] font-mono text-red-500 hover:text-red-650 dark:hover:text-red-400 uppercase font-bold cursor-pointer"
              >
                Clear All
              </button>
            )}
            {!isModal && (
              <button
                type="button"
                onClick={() => setIsExpanded(true)}
                className="p-1 hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded text-zinc-450 hover:text-zinc-700 dark:hover:text-zinc-200 transition-colors cursor-pointer"
                title="Expand to big editor"
              >
                <Maximize2 className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        </div>

        {activeTab === 'list' ? (
          <div className="flex flex-col gap-3 flex-1 min-h-0">
            {/* Add & Search Controls */}
            <div className="flex flex-col sm:flex-row gap-2">
              {/* Search Input */}
              <div className="relative flex-1">
                <span className="absolute inset-y-0 left-0 flex items-center pl-2.5 pointer-events-none text-zinc-450">
                  <Search className="h-3.5 w-3.5" />
                </span>
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder={`Search ${label.toLowerCase()}...`}
                  className="w-full pl-8 pr-8 py-1.5 border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded font-mono text-xs focus:outline-none"
                />
                {searchQuery && (
                  <button
                    type="button"
                    onClick={() => setSearchQuery('')}
                    className="absolute inset-y-0 right-0 flex items-center pr-2.5 text-zinc-455 hover:text-zinc-650 dark:hover:text-zinc-200 cursor-pointer"
                  >
                    <X className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>

              {/* Quick Add */}
              <div className="flex gap-1">
                <input
                  type="text"
                  value={newItem}
                  onChange={(e) => setNewItem(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      handleAddItem(e)
                    }
                  }}
                  placeholder="Add item..."
                  className="w-32 px-3 py-1.5 border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 rounded font-mono text-xs focus:outline-none"
                />
                <button
                  type="button"
                  onClick={(e) => handleAddItem(e)}
                  className="flex items-center justify-center p-1.5 border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-850 rounded hover:bg-zinc-50 dark:hover:bg-zinc-800 text-zinc-600 dark:text-zinc-350 cursor-pointer"
                >
                  <Plus className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>

            {/* Validation Error Message */}
            {validationError && (
              <div className="text-[11px] font-mono text-red-500 dark:text-red-400 flex items-center gap-1.5 px-1 py-0.5">
                <AlertCircle className="h-3.5 w-3.5 animate-pulse" />
                <span>{validationError}</span>
              </div>
            )}

            {/* List display */}
            <div
              className={`flex-1 overflow-y-auto border border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/10 rounded p-2.5 font-mono text-xs ${
                isModal ? 'h-[400px]' : 'h-40'
              }`}
            >
              {filteredItems.length === 0 ? (
                <div className="h-full flex items-center justify-center text-zinc-400 text-xs py-8">
                  {searchQuery ? 'No matching items' : 'List is empty'}
                </div>
              ) : (
                <div className="flex flex-col gap-2">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-1.5">
                    {visibleItems.map((item) => (
                      <div
                        key={item}
                        className="flex items-center justify-between gap-2 px-2.5 py-1.5 bg-white dark:bg-zinc-900/50 border border-zinc-150 dark:border-zinc-800/80 rounded group hover:border-zinc-300 dark:hover:border-zinc-700 transition-colors"
                      >
                        <span className="truncate select-all text-zinc-800 dark:text-zinc-200">{item}</span>
                        <button
                          type="button"
                          onClick={() => handleDeleteItem(item)}
                          className="text-zinc-400 hover:text-red-500 p-0.5 rounded cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </div>
                    ))}
                  </div>
                  {!isModal && filteredItems.length > maxCollapsedItems && (
                    <div className="text-center pt-2 border-t border-dashed border-zinc-200 dark:border-zinc-800/50 mt-1">
                      <button
                        type="button"
                        onClick={() => setIsExpanded(true)}
                        className="text-[10px] font-mono text-zinc-455 hover:text-zinc-700 dark:hover:text-zinc-300 uppercase font-bold cursor-pointer transition-colors"
                      >
                        + {filteredItems.length - maxCollapsedItems} more items (click expand to edit/view all)
                      </button>
                    </div>
                  )}
                </div>
              )}
            </div>
          </div>
        ) : (
          <div className="flex-1 flex flex-col min-h-0">
            <textarea
              id={id}
              disabled={disabled}
              value={value}
              onChange={(e) => onChange(e.target.value)}
              onBlur={handleRawBlur}
              className={`w-full p-3 border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/10 font-mono text-xs leading-relaxed focus:outline-none rounded resize-none flex-1 ${
                isModal ? 'h-[400px]' : 'h-40'
              }`}
              placeholder={placeholder}
            />
            {invalidLines.length > 0 && (
              <div className="mt-2 p-2 bg-red-50 dark:bg-red-950/20 border border-red-200 dark:border-red-900/50 rounded flex flex-col gap-1">
                <span className="text-[11px] font-mono font-bold text-red-650 dark:text-red-400 flex items-center gap-1">
                  <AlertCircle className="h-3.5 w-3.5" />
                  Warning: {invalidLines.length} invalid {invalidLines.length === 1 ? 'item' : 'items'} detected
                </span>
                <span className="text-[10px] font-mono text-red-500 dark:text-red-400/80 max-h-16 overflow-y-auto">
                  Invalid values: {invalidLines.join(', ')}
                </span>
              </div>
            )}
          </div>
        )}
      </div>
    )
  }

  return (
    <>
      <div className="flex flex-col gap-2 border border-zinc-200 dark:border-zinc-800 p-4 rounded bg-white dark:bg-zinc-950/10 shadow-sm">
        <label
          htmlFor={id}
          className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-500"
        >
          {label}
        </label>
        {EditorContent(false)}
      </div>

      {/* Expanded Modal */}
      {isExpanded && (
        <div className="fixed inset-0 z-55 flex items-center justify-center p-4 bg-zinc-950/60 backdrop-blur-sm">
          <div className="w-full max-w-3xl flex flex-col border border-zinc-200 dark:border-zinc-800 rounded-lg bg-white dark:bg-zinc-900 shadow-2xl p-6 h-[580px]">
            {/* Modal Header */}
            <div className="flex items-center justify-between border-b border-zinc-150 dark:border-zinc-800 pb-3 mb-4">
              <div>
                <h3 className="text-sm font-mono font-bold uppercase tracking-wider text-zinc-850 dark:text-zinc-100">
                  {label} Editor
                </h3>
                <p className="text-[10px] text-zinc-400 font-mono">
                  Manage list items individually or edit raw text. Changes are saved automatically.
                </p>
              </div>
              <button
                type="button"
                onClick={() => setIsExpanded(false)}
                className="p-1 hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200 transition-colors cursor-pointer"
              >
                <Minimize2 className="h-5 w-5" />
              </button>
            </div>

            {/* Modal Body */}
            <div className="flex-1 min-h-0">
              {EditorContent(true)}
            </div>

            {/* Modal Footer */}
            <div className="flex justify-end pt-3 border-t border-zinc-150 dark:border-zinc-800 mt-4">
              <button
                type="button"
                onClick={() => setIsExpanded(false)}
                className="px-4 py-2 bg-zinc-900 dark:bg-zinc-100 hover:bg-zinc-800 dark:hover:bg-zinc-200 text-white dark:text-zinc-900 font-mono text-xs font-bold uppercase rounded cursor-pointer"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  )
}

const validateTopic = (val: string) => {
  const trimmed = val.trim()
  if (!trimmed) {
    return { valid: true, normalized: '' }
  }
  const match = trimmed.match(/^([tT])(\d{5})$/)
  if (!match) {
    return {
      valid: false,
      error: 'Topic ID must be in the format T12345 (T followed by exactly 5 digits)',
    }
  }
  return {
    valid: true,
    normalized: `T${match[2]}`,
  }
}

const validateDoi = (val: string) => {
  const trimmed = val.trim()
  if (!trimmed) {
    return { valid: true, normalized: '' }
  }
  // Relaxed DOI validator: must start with 10. and have a slash separating prefix and suffix
  const match = trimmed.match(/^10\.\d{4,9}\/.+$/)
  if (!match) {
    return {
      valid: false,
      error: 'DOI must start with "10." followed by a 4-9 digit prefix and a slash (e.g. 10.1016/...)',
    }
  }
  return {
    valid: true,
    normalized: trimmed,
  }
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
  const [rowCount, setRowCount] = useState<number | null>(null)
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
  const [checkAnchors, setCheckAnchors] = useState(true)

  // OpenAlex Topics States
  interface OpenAlexTopic {
    topic_id: string
    display_name: string
    description: string
    paper_count: number
    percentage: number
  }
  const [checkingTopics, setCheckingTopics] = useState(false)
  const [openalexTopics, setOpenalexTopics] = useState<OpenAlexTopic[] | null>(null)
  const [openalexTopicsTotal, setOpenalexTopicsTotal] = useState<number | null>(null)
  const [openalexTopicsTotalPapers, setOpenalexTopicsTotalPapers] = useState<number | null>(null)
  const [showAllTopics, setShowAllTopics] = useState(false)

  const selectedTopicsList = useMemo(() => {
    return topics
      .split('\n')
      .map((t) => t.trim())
      .filter(Boolean)
  }, [topics])

  const isTopicSelected = useCallback((topicId: string) => {
    return selectedTopicsList.includes(topicId.trim())
  }, [selectedTopicsList])

  const handleToggleTopic = useCallback((topicId: string, checked: boolean) => {
    const trimmedId = topicId.trim()
    let newList: string[]
    if (checked) {
      if (selectedTopicsList.includes(trimmedId)) return
      newList = [...selectedTopicsList, trimmedId]
    } else {
      newList = selectedTopicsList.filter((t) => t !== trimmedId)
    }
    setTopics(newList.join('\n'))
  }, [selectedTopicsList])

  // Save States
  const [saving, setSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState(false)

  // Navigation & Tab States
  const [activeTab, setActiveTab] = useState<'keywords' | 'filters' | 'execution'>('keywords')

  // Toast Notifications States
  interface Toast {
    id: string
    type: 'success' | 'error' | 'info'
    title: string
    message: string
  }
  const [toasts, setToasts] = useState<Toast[]>([])

  const cleanErrorMessage = (msg: string): string => {
    if (!msg) return 'An unknown error occurred.'
    if (
      msg.includes('non-JSON response') &&
      (msg.includes('<html') || msg.includes('<!doctype html>'))
    ) {
      const httpStatusMatch = msg.match(/HTTP \d+/i)
      const statusStr = httpStatusMatch ? ` (${httpStatusMatch[0]})` : ''
      const titleMatch = msg.match(/<title>([\s\S]*?)<\/title>/i)
      if (titleMatch && titleMatch[1]) {
        return `OpenAlex server error${statusStr}: ${titleMatch[1].trim()}`
      }
      const h1Match = msg.match(/<h1>([\s\S]*?)<\/h1>/i)
      if (h1Match && h1Match[1]) {
        return `OpenAlex server error${statusStr}: ${h1Match[1].trim()}`
      }
      return `OpenAlex server returned an invalid HTML error response${statusStr}. Please try again later.`
    }
    return msg.replace(/<[^>]*>/g, '').trim()
  }

  const addToast = (type: 'success' | 'error' | 'info', title: string, message: string) => {
    const id = Math.random().toString(36).substring(2, 9)
    setToasts((prev) => [...prev, { id, type, title, message: cleanErrorMessage(message) }])
    setTimeout(() => {
      removeToast(id)
    }, 6000)
  }

  const removeToast = (id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }

  // Web Browser Notification Permission State
  const [notificationPermission, setNotificationPermission] = useState<'granted' | 'denied' | 'default' | 'unsupported'>('default')

  useEffect(() => {
    if (!('Notification' in window)) {
      setNotificationPermission('unsupported')
    } else {
      setNotificationPermission(Notification.permission)
    }
  }, [])

  const requestNotificationPermission = async () => {
    if (!('Notification' in window)) return
    const permission = await Notification.requestPermission()
    setNotificationPermission(permission)
    if (permission === 'granted') {
      addToast('success', 'Notifications Enabled', 'You will now receive desktop notifications when processes finish.')
    }
  }

  const sendNotification = (title: string, body: string) => {
    if ('Notification' in window && Notification.permission === 'granted') {
      try {
        new Notification(title, { body: cleanErrorMessage(body) })
      } catch (err) {
        console.error('Failed to trigger notification:', err)
      }
    }
  }

  // Custom Alert Modal State
  const [alertConfig, setAlertConfig] = useState<{
    type: 'success' | 'error' | 'info'
    title: string
    message: string
  } | null>(null)

  const triggerAlert = (type: 'success' | 'error' | 'info', title: string, message: string) => {
    setAlertConfig({ type, title, message: cleanErrorMessage(message) })
  }

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
        setRowCount(data.row_count !== undefined ? data.row_count : null)
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
        triggerAlert('error', 'Upload Failed', errData.error || response.statusText)
      }
    } catch (err: unknown) {
      triggerAlert('error', 'Upload Failed', err instanceof Error ? err.message : String(err))
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
        triggerAlert('error', 'Extraction Failed', errData.error || response.statusText)
      }
    } catch (err: unknown) {
      triggerAlert('error', 'Extraction Failed', err instanceof Error ? err.message : String(err))
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
        triggerAlert('error', 'Validation Failed', 'Failed to validate query syntax.')
      }
    } catch (err: unknown) {
      triggerAlert('error', 'Connection Error', err instanceof Error ? err.message : String(err))
    } finally {
      setValidating(false)
    }
  }

  // Fetch Real Count Handler
  const handleGetOpenAlexCount = async () => {
    if (!keywords.trim()) {
      triggerAlert('info', 'Empty Query', 'Please enter a search keywords query first.')
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
          check_anchors: checkAnchors,
        }),
      })

      if (response.ok) {
        const data = await response.json()
        setOpenalexCount(data.count)
        setAnchorsTotal(data.anchors_total || 0)
        setAnchorsMatched(data.anchors_matched || 0)
        setAnchorsMissing(data.anchors_missing || [])

        const countVal = data.count || 0
        const matched = data.anchors_matched || 0
        const total = data.anchors_total || 0

        if (total > 0) {
          addToast(
            'success',
            'Search Count Completed',
            `Found ${countVal.toLocaleString()} papers. Anchor match: ${matched}/${total}.`,
          )
        } else {
          addToast(
            'success',
            'Search Count Completed',
            `Found ${countVal.toLocaleString()} papers.`,
          )
        }
        sendNotification(
          'Stratum: Search Completed',
          `Estimated papers matching keywords: ${countVal.toLocaleString()}. Anchor match: ${matched}/${total}.`,
        )
      } else {
        const errData = await response.json()
        const errMsg = errData.error || response.statusText
        triggerAlert('error', 'Count Failed', errMsg)
        addToast('error', 'Search Count Failed', errMsg)
        sendNotification('Stratum: Search Count Failed', errMsg)
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err)
      triggerAlert(
        'error',
        'Count Request Failed',
        errMsg,
      )
      addToast('error', 'Search Count Failed', errMsg)
      sendNotification('Stratum: Search Count Failed', errMsg)
    } finally {
      setCheckingCount(false)
    }
  }

  const handleGetOpenAlexTopics = async () => {
    if (!keywords.trim()) {
      triggerAlert('info', 'Empty Query', 'Please enter a search keywords query first.')
      return
    }
    setCheckingTopics(true)
    setOpenalexTopics(null)
    setOpenalexTopicsTotal(null)
    setOpenalexTopicsTotalPapers(null)

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
      const response = await fetch(`/api/openalex/topics`, {
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
          details: true,
        }),
      })

      if (response.ok) {
        const data = await response.json()
        setOpenalexTopics(data.topics || [])
        setOpenalexTopicsTotal(data.total_topics || 0)
        setOpenalexTopicsTotalPapers(data.total_papers || 0)

        const topicsCount = data.total_topics || 0
        const papersCount = data.total_papers || 0
        addToast(
          'success',
          'Topics Analysis Completed',
          `Resolved ${topicsCount} topics across ${papersCount.toLocaleString()} papers.`,
        )
        sendNotification(
          'Stratum: Topics Analysis Completed',
          `Found ${topicsCount} active research topics in current search results.`,
        )
      } else {
        const errData = await response.json()
        const errMsg = errData.error || response.statusText
        triggerAlert('error', 'Topics Fetch Failed', errMsg)
        addToast('error', 'Topics Analysis Failed', errMsg)
        sendNotification('Stratum: Topics Analysis Failed', errMsg)
      }
    } catch (err: unknown) {
      const errMsg = err instanceof Error ? err.message : String(err)
      triggerAlert(
        'error',
        'Topics Request Failed',
        errMsg,
      )
      addToast('error', 'Topics Analysis Failed', errMsg)
      sendNotification('Stratum: Topics Analysis Failed', errMsg)
    } finally {
      setCheckingTopics(false)
    }
  }

  const handleDownloadTopicsCSV = () => {
    if (!openalexTopics || openalexTopics.length === 0) return

    // CSV Headers
    const headers = ['Topic ID', 'Topic Name', 'Description', 'Paper Count', 'Percentage']

    // Helper to escape values for CSV
    const escapeCSV = (val: string) => {
      const escaped = val.replace(/"/g, '""')
      return `"${escaped}"`
    }

    // Map rows
    const rows = openalexTopics.map((t) => [
      t.topic_id,
      escapeCSV(t.display_name),
      escapeCSV(t.description || ''),
      t.paper_count,
      `${t.percentage.toFixed(4)}%`
    ])

    // Construct CSV content
    const csvContent = [
      headers.join(','),
      ...rows.map((row) => row.join(','))
    ].join('\n')

    // Create a blob and trigger download
    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.setAttribute('href', url)
    link.setAttribute('download', `openalex_topics_${activeProject || 'export'}.csv`)
    link.style.visibility = 'hidden'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  // Save Config Handler
  const handleSaveConfig = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setSaveSuccess(false)

    // Validate topics
    const invalidTopicsList = topics
      .split('\n')
      .map((t) => t.trim())
      .filter(Boolean)
      .filter((t) => !validateTopic(t).valid)

    if (invalidTopicsList.length > 0) {
      triggerAlert(
        'error',
        'Validation Error',
        `Cannot save configuration: Target Topics contains invalid entries: ${invalidTopicsList.join(', ')}`
      )
      setSaving(false)
      return
    }

    // Validate DOIs
    const invalidAnchorsList = anchors
      .split('\n')
      .map((a) => a.trim())
      .filter(Boolean)
      .filter((a) => !validateDoi(a).valid)

    if (invalidAnchorsList.length > 0) {
      triggerAlert(
        'error',
        'Validation Error',
        `Cannot save configuration: Anchor DOIs contains invalid entries: ${invalidAnchorsList.join(', ')}`
      )
      setSaving(false)
      return
    }

    const normalizedTopics = topics
      .split('\n')
      .map((t) => {
        const trimmed = t.trim()
        if (!trimmed) return ''
        const res = validateTopic(trimmed)
        return res.valid ? (res.normalized || trimmed) : t
      })
      .join('\n')

    const normalizedAnchors = anchors
      .split('\n')
      .map((a) => {
        const trimmed = a.trim()
        if (!trimmed) return ''
        const res = validateDoi(trimmed)
        return res.valid ? (res.normalized || trimmed) : a
      })
      .join('\n')

    setTopics(normalizedTopics)
    setAnchors(normalizedAnchors)

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
          topics: normalizedTopics,
          anchors: normalizedAnchors,
          label: saveLabel,
        }),
      })

      if (response.ok) {
        setSaveSuccess(true)
        setSaveLabel('')
        fetchConfig()
        setTimeout(() => setSaveSuccess(false), 4000)
      } else {
        const errData = await response.json().catch(() => ({}))
        if (errData.errors && Array.isArray(errData.errors)) {
          triggerAlert('error', 'Failed to save configuration', '- ' + errData.errors.join('\n- '))
        } else {
          triggerAlert(
            'error',
            'Failed to save configuration',
            errData.error || response.statusText || 'Unknown error',
          )
        }
      }
    } catch (err: unknown) {
      triggerAlert('error', 'Save Failed', err instanceof Error ? err.message : String(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex flex-col gap-8 w-full max-w-7xl mx-auto relative">
      {/* Floating Toast Notification Alerts */}
      <div className="fixed top-6 right-6 z-50 flex flex-col gap-3 w-80 pointer-events-none">
        {toasts.map((toast) => (
          <div
            key={toast.id}
            className={`p-4 rounded-lg shadow-lg border text-xs font-mono flex items-start justify-between gap-3 pointer-events-auto transition-all duration-300 animate-slide-in ${
              toast.type === 'success'
                ? 'bg-emerald-50 dark:bg-emerald-950/20 border-emerald-200 dark:border-emerald-800 text-emerald-800 dark:text-emerald-300'
                : toast.type === 'error'
                  ? 'bg-red-50 dark:bg-red-950/20 border-red-200 dark:border-red-800 text-red-800 dark:text-red-300'
                  : 'bg-blue-50 dark:bg-blue-950/20 border-blue-200 dark:border-blue-800 text-blue-800 dark:text-blue-300'
            }`}
          >
            <div className="flex flex-col gap-1">
              <span className="font-bold uppercase tracking-wider text-[10px]">
                {toast.title}
              </span>
              <p className="font-sans leading-relaxed text-zinc-600 dark:text-zinc-400">
                {toast.message}
              </p>
            </div>
            <button
              type="button"
              onClick={() => removeToast(toast.id)}
              className="text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-200 cursor-pointer shrink-0"
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
      </div>

      {/* Page Header */}
      <div className="flex flex-col md:flex-row md:items-center md:justify-between border-b border-zinc-200 pb-5 dark:border-zinc-800 gap-4">
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-mono font-bold tracking-tight text-zinc-950 dark:text-zinc-50 uppercase">
            Pipeline Setup & Keywords Studio
          </h1>
          <p className="text-sm text-zinc-500 dark:text-zinc-400">
            Upload local paper catalogs to extract TF-IDF terms, refine boolean queries, configure
            multiple keys, and validate query volumes.
          </p>
        </div>

        {/* Tab Switcher - Premium Monochrome Design */}
        <div className="flex items-center bg-zinc-100 dark:bg-zinc-900 p-1 rounded-lg border border-zinc-200 dark:border-zinc-800 font-mono text-xs font-bold uppercase select-none shrink-0 self-start md:self-center">
          <button
            type="button"
            onClick={() => setActiveTab('keywords')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-md transition-all cursor-pointer ${
              activeTab === 'keywords'
                ? 'bg-white dark:bg-zinc-800 shadow-sm text-zinc-955 dark:text-white border border-zinc-200/50 dark:border-zinc-700/50'
                : 'text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300'
            }`}
          >
            <FileText className="h-3.5 w-3.5 text-zinc-500" />
            Keywords & Anchors
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('filters')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-md transition-all cursor-pointer ${
              activeTab === 'filters'
                ? 'bg-white dark:bg-zinc-800 shadow-sm text-zinc-955 dark:text-white border border-zinc-200/50 dark:border-zinc-700/50'
                : 'text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300'
            }`}
          >
            <Sliders className="h-3.5 w-3.5 text-zinc-500" />
            Filters & APIs
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('execution')}
            className={`flex items-center gap-2 px-3.5 py-2 rounded-md transition-all cursor-pointer ${
              activeTab === 'execution'
                ? 'bg-white dark:bg-zinc-800 shadow-sm text-zinc-955 dark:text-white border border-zinc-200/50 dark:border-zinc-700/50'
                : 'text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300'
            }`}
          >
            <Activity className="h-3.5 w-3.5 text-zinc-500" />
            Execution & Analysis
          </button>
        </div>
      </div>

      {saveSuccess && (
        <div className="flex items-center gap-3 p-4 border border-green-200 bg-green-50/50 text-green-700 dark:border-green-800/40 dark:bg-green-950/20 dark:text-green-400 font-mono text-xs rounded">
          <CheckCircle className="h-4 w-4 shrink-0" />
          <span>
            [SUCCESS] Configuration successfully saved. Keywords, topics, and API keys updated.
          </span>
        </div>
      )}

      <form onSubmit={handleSaveConfig} className="w-full flex flex-col gap-8">
        {/* Top Save & Version Selector Panel */}
        <div className="flex flex-col md:flex-row items-stretch md:items-center justify-between border border-zinc-200 dark:border-zinc-800 p-4 rounded bg-white dark:bg-zinc-950/10 shadow-sm gap-4">
          {/* Left Side: Load dropdown & Label input */}
          <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-4 flex-1">
            {configHistory.length > 0 && (
              <div className="flex flex-col gap-1 min-w-[200px]">
                <label htmlFor="version-selector" className="text-[10px] font-mono uppercase text-zinc-400 font-bold">
                  Load Saved Version
                </label>
                <select
                  id="version-selector"
                  onChange={(e) => {
                    const verNum = parseInt(e.target.value, 10)
                    if (isNaN(verNum)) return
                    const rev = configHistory.find((r) => r.version === verNum)
                    if (rev) {
                      setKeywords(rev.keywords)
                      setTopics(rev.topics)
                      setAnchors(rev.anchors)
                      addToast(
                        'info',
                        'Revision Restored',
                        `Loaded parameters from revision v${rev.version}. Click Save to apply.`,
                      )
                    }
                    e.target.value = ''
                  }}
                  defaultValue=""
                  className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-2.5 rounded font-mono text-xs focus:outline-none cursor-pointer"
                >
                  <option value="" disabled>
                    Select version to restore...
                  </option>
                  {configHistory
                    .slice()
                    .reverse()
                    .map((rev) => (
                      <option key={rev.version} value={rev.version}>
                        v{rev.version} - {rev.label || 'No description'} ({rev.timestamp})
                      </option>
                    ))}
                </select>
              </div>
            )}

            <div className="flex flex-col gap-1 flex-1 max-w-md">
              <label htmlFor="save-label" className="text-[10px] font-mono uppercase text-zinc-400 font-bold">
                Revision Label / Commit Message (Optional)
              </label>
              <input
                id="save-label"
                type="text"
                disabled={saving}
                value={saveLabel}
                onChange={(e) => setSaveLabel(e.target.value)}
                placeholder="e.g. Configured new keyword search options..."
                className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-2.5 rounded font-mono text-xs focus:outline-none"
              />
            </div>
          </div>

          {/* Right Side: Save Button */}
          <div className="flex items-end self-end md:self-center">
            <button
              type="submit"
              disabled={saving || queryValid === false}
              className="flex items-center gap-2 px-5 py-2.5 bg-zinc-950 dark:bg-zinc-50 text-white dark:text-zinc-950 rounded hover:bg-zinc-900 dark:hover:bg-zinc-200 font-mono text-xs font-bold uppercase cursor-pointer disabled:opacity-50 select-none shadow-sm transition-colors"
            >
              {saving ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span>Saving...</span>
                </>
              ) : (
                <>
                  <Save className="h-4 w-4" />
                  <span>Save Configuration</span>
                </>
              )}
            </button>
          </div>
        </div>

        {/* Tab 1: Keywords & Anchors */}
        {activeTab === 'keywords' && (
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start w-full">
            {/* LEFT SUB-COLUMN: TF-IDF Extraction (cols: 5) */}
            <div className="lg:col-span-5 flex flex-col gap-6 border border-zinc-200 dark:border-zinc-800 p-5 bg-zinc-50/20 dark:bg-zinc-950/20 rounded">
              <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-800 pb-2">
                <span className="text-sm font-mono font-bold uppercase tracking-wider text-zinc-800 dark:text-zinc-200">
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
                    type="button"
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
                      type="button"
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
                    <Check className="h-3 w-3" /> Catalog parsed. {rowCount !== null ? `${rowCount.toLocaleString()} rows and ` : ''}{columns.length} columns detected.
                  </span>
                )}
              </div>

              {/* Step 2: Column Selection & Parameters (Only after upload) */}
              {uploadSuccess && (
                <div className="flex flex-col gap-4 border-t border-zinc-200 dark:border-zinc-800 pt-4">
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
                        className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs w-full focus:outline-none"
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
                        className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs w-full focus:outline-none"
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
                        DOI Column
                      </label>
                      <select
                        value={selectedDoiCol}
                        onChange={(e) => setSelectedDoiCol(e.target.value)}
                        className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs w-full focus:outline-none"
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
                  <div className="grid grid-cols-3 gap-2 border-t border-zinc-100 dark:border-zinc-800 pt-3 mt-1">
                    <div className="flex flex-col gap-1">
                      <label className="text-[9px] font-mono uppercase text-zinc-400">
                        N-gram Range
                      </label>
                      <div className="flex items-center gap-1 font-mono text-xs">
                        <input
                          type="number"
                          value={ngramMin}
                          onChange={(e) => setNgramMin(Number(e.target.value))}
                          className="w-full bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1 rounded text-center focus:outline-none"
                        />
                        <span className="text-zinc-400">-</span>
                        <input
                          type="number"
                          value={ngramMax}
                          onChange={(e) => setNgramMax(Number(e.target.value))}
                          className="w-full bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1 rounded text-center focus:outline-none"
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
                        className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1 rounded font-mono text-xs text-center focus:outline-none"
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
                        className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1 rounded font-mono text-xs text-center focus:outline-none"
                      />
                    </div>
                  </div>

                  <button
                    type="button"
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
                <div className="flex flex-col gap-3 border-t border-zinc-200 dark:border-zinc-800 pt-4">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-mono font-bold uppercase text-zinc-500">
                      3. Select Scored Keywords ({extractedKeywords.filter((k) => k.selected).length}{' '}
                      selected)
                    </span>
                    <div className="flex gap-2 text-[10px] font-mono">
                      <button
                        type="button"
                        onClick={() => selectAllKeywords(true)}
                        className="text-zinc-500 hover:underline cursor-pointer font-semibold"
                      >
                        All
                      </button>
                      <span className="text-zinc-300">|</span>
                      <button
                        type="button"
                        onClick={() => selectAllKeywords(false)}
                        className="text-zinc-500 hover:underline cursor-pointer font-semibold"
                      >
                        None
                      </button>
                    </div>
                  </div>

                  {/* Scrollable checklist */}
                  <div className="max-h-60 overflow-y-auto border border-zinc-200 dark:border-zinc-800 rounded bg-white dark:bg-zinc-950/40 p-1 flex flex-col gap-0.5">
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
                    type="button"
                    onClick={appendKeywordsToQuery}
                    className="w-full flex items-center justify-center gap-2 px-3 py-2 border border-zinc-300 dark:border-zinc-800 rounded bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 font-mono text-xs cursor-pointer select-none font-bold uppercase"
                  >
                    <Copy className="h-4 w-4" />
                    Append Selected to Query
                  </button>
                </div>
              )}
            </div>

            {/* RIGHT SUB-COLUMN: Query Builder, Topics & Anchors Editor (cols: 7) */}
            <div className="lg:col-span-7 flex flex-col gap-6">
              <div className="flex flex-col gap-3 border border-zinc-200 dark:border-zinc-800 p-5 rounded bg-white dark:bg-zinc-950/10">
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
                  onChange={(e) => {
                    setKeywords(e.target.value)
                    setQueryValid(null)
                    setQueryErrors([])
                  }}
                  className="w-full h-64 p-4 border border-zinc-200 dark:border-zinc-800 bg-zinc-950 text-zinc-300 font-mono text-xs leading-relaxed focus:outline-none focus:ring-0 rounded"
                  placeholder="Enter boolean query using OR / AND / NOT operators..."
                />

                <div className="flex items-center justify-between flex-wrap gap-3 mt-1.5">
                  <button
                    type="button"
                    onClick={handleValidateQuery}
                    disabled={validating || !keywords.trim()}
                    className="flex items-center gap-2 px-3.5 py-1.5 border border-zinc-300 dark:border-zinc-800 rounded bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 font-mono text-xs cursor-pointer disabled:opacity-50 select-none font-bold uppercase"
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

              {/* Anchor DOIs Config Panel */}
              <div className="flex flex-col gap-6">
                <SearchableListEditor
                  id="anchors-input"
                  label="Anchor DOIs (anchor.txt)"
                  disabled={saving}
                  value={anchors}
                  onChange={setAnchors}
                  placeholder="10.1016/j.renene..."
                  maxCollapsedItems={8}
                  validate={validateDoi}
                />
              </div>
            </div>
          </div>
        )}

        {/* Tab 2: Filters & Configuration */}
        {activeTab === 'filters' && (
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start w-full">
            {/* LEFT SUB-COLUMN: API Keys & Polite Pool Email config (cols: 6) */}
            <div className="lg:col-span-6 flex flex-col gap-6">
              <div className="border border-zinc-200 dark:border-zinc-800 p-5 bg-zinc-50/20 dark:bg-zinc-950/20 rounded flex flex-col gap-6">
                <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-800 pb-2">
                  <span className="text-sm font-mono font-bold uppercase tracking-wider text-zinc-800 dark:text-zinc-200">
                    API Keys & Polite Pool
                  </span>
                  <Key className="h-4 w-4 text-zinc-400" />
                </div>

                <div className="flex flex-col gap-4">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-[11px] font-mono uppercase text-zinc-400 font-bold">
                      OpenAlex API Keys (Comma-separated for rotation)
                    </label>
                    <input
                      type="text"
                      value={apiKeysStr}
                      onChange={(e) => setApiKeysStr(e.target.value)}
                      placeholder="Key1, Key2, Key3..."
                      className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-2.5 rounded font-mono text-xs focus:outline-none"
                    />
                    <span className="text-[9px] text-zinc-400 font-sans leading-normal">
                      If multiple keys are set, Stratum rotates queries across them and automatically
                      sets aside keys that encounter quota exceptions.
                    </span>
                  </div>

                  <div className="flex flex-col gap-1.5 mt-1 border-t border-zinc-200 dark:border-zinc-800 pt-3">
                    <label className="text-[11px] font-mono uppercase text-zinc-400 font-bold">
                      Polite Pool Email Address (Contact UserAgent)
                    </label>
                    <input
                      type="email"
                      value={apiEmail}
                      onChange={(e) => setApiEmail(e.target.value)}
                      placeholder="your.name@institution.edu"
                      className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-2.5 rounded font-mono text-xs focus:outline-none"
                    />
                    <span className="text-[9px] text-zinc-400 font-sans leading-normal">
                      OpenAlex reserves a dedicated "polite pool" with faster response times for users
                      who send their contact email in the headers.
                    </span>
                  </div>
                </div>
              </div>

              {/* Target Topics Panel is now its own card here */}
              <SearchableListEditor
                id="topics-input"
                label="Target Topics (topics.txt)"
                disabled={saving}
                value={topics}
                onChange={setTopics}
                placeholder="T10020..."
                maxCollapsedItems={6}
                validate={validateTopic}
              />
            </div>

            {/* RIGHT SUB-COLUMN: Date Filters & Document Types (cols: 6) */}
            <div className="lg:col-span-6 flex flex-col gap-6">
              <div className="flex flex-col gap-5 border border-zinc-200 dark:border-zinc-800 p-5 rounded bg-white dark:bg-zinc-950/10">
                <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-800 pb-2">
                  <span className="text-sm font-mono font-bold uppercase tracking-wider text-zinc-800 dark:text-zinc-200">
                    Search Constraints & Filtering
                  </span>
                  <Calendar className="h-4 w-4 text-zinc-400" />
                </div>

                {/* Date Filters */}
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  <div className="flex flex-col gap-1.5">
                    <label className="text-[11px] font-mono uppercase text-zinc-500 font-bold">
                      Publication Date From
                    </label>
                    <input
                      type="date"
                      value={dateFrom}
                      onChange={(e) => setDateFrom(e.target.value)}
                      className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-2 rounded font-mono text-xs focus:outline-none"
                    />
                  </div>
                  <div className="flex flex-col gap-1.5">
                    <label className="text-[11px] font-mono uppercase text-zinc-500 font-bold">
                      Publication Date To
                    </label>
                    <input
                      type="date"
                      value={dateTo}
                      onChange={(e) => setDateTo(e.target.value)}
                      className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-2 rounded font-mono text-xs focus:outline-none"
                    />
                  </div>
                </div>

                {/* Document Types Checkboxes */}
                <div className="flex flex-col gap-2.5 mt-2 border-t border-zinc-200 dark:border-zinc-800 pt-4">
                  <span className="text-[11px] font-mono uppercase text-zinc-500 font-bold">
                    Target Document Types
                  </span>
                  <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                    {Object.keys(selectedDocTypes).map((type) => (
                      <label
                        key={type}
                        className={`flex items-center gap-2 p-2 rounded border border-zinc-200 dark:border-zinc-800/80 cursor-pointer select-none font-mono text-[11px] ${
                          selectedDocTypes[type]
                            ? 'bg-zinc-50/50 border-zinc-300 dark:bg-zinc-900/10 dark:border-zinc-700 text-zinc-900 dark:text-zinc-100 font-semibold'
                            : 'bg-white dark:bg-zinc-950/20 hover:bg-zinc-50 dark:hover:bg-zinc-900/40 text-zinc-400'
                        }`}
                      >
                        <input
                          type="checkbox"
                          checked={selectedDocTypes[type]}
                          onChange={() =>
                            setSelectedDocTypes((prev) => ({
                              ...prev,
                              [type]: !prev[type],
                            }))
                          }
                          className="rounded border-zinc-300 text-zinc-900 focus:ring-0 cursor-pointer"
                        />
                        <span className="capitalize">{type.replace('-', ' ')}</span>
                      </label>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Tab 3: Execution & Search Analysis */}
        {activeTab === 'execution' && (
          <div className="grid grid-cols-1 lg:grid-cols-12 gap-8 items-start w-full">
            {/* LEFT COLUMN: Notifications Permissions Settings & Volume Estimation count (cols: 6) */}
            <div className="lg:col-span-6 flex flex-col gap-6">
              
              {/* Desktop Notifications Panel */}
              <div className="flex flex-col gap-4 border border-zinc-200 dark:border-zinc-800 p-5 rounded bg-zinc-50/10 dark:bg-zinc-950/10">
                <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-800 pb-2">
                  <span className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-800 dark:text-zinc-200 flex items-center gap-2">
                    <Bell className="h-4 w-4 text-zinc-500 animate-pulse" />
                    Desktop Alerts Setting
                  </span>
                  {notificationPermission === 'granted' ? (
                    <span className="px-2 py-0.5 rounded bg-emerald-50 dark:bg-emerald-950/20 text-emerald-600 border border-emerald-200/50 font-mono text-[9px] font-bold uppercase">
                      Enabled
                    </span>
                  ) : notificationPermission === 'denied' ? (
                    <span className="px-2 py-0.5 rounded bg-red-50 dark:bg-red-950/20 text-red-600 border border-red-200/50 font-mono text-[9px] font-bold uppercase">
                      Blocked
                    </span>
                  ) : notificationPermission === 'unsupported' ? (
                    <span className="px-2 py-0.5 rounded bg-zinc-100 text-zinc-500 font-mono text-[9px] font-bold uppercase">
                      Unsupported
                    </span>
                  ) : (
                    <span className="px-2 py-0.5 rounded bg-amber-50 dark:bg-amber-950/20 text-amber-600 border border-amber-200/50 font-mono text-[9px] font-bold uppercase">
                      Setup needed
                    </span>
                  )}
                </div>

                <p className="text-[11px] text-zinc-400 font-sans leading-relaxed">
                  Receive browser push alerts when long-running search calculations or topic details fetches finish, so you are immediately notified even when working in other tabs.
                </p>

                {notificationPermission === 'default' && (
                  <button
                    type="button"
                    onClick={requestNotificationPermission}
                    className="w-full flex items-center justify-center gap-2 px-3 py-2 border border-zinc-300 dark:border-zinc-800 rounded font-mono text-xs font-bold uppercase bg-white dark:bg-zinc-900 hover:bg-zinc-50 dark:hover:bg-zinc-800 text-zinc-700 dark:text-zinc-300 cursor-pointer transition select-none"
                  >
                    <Bell className="h-3.5 w-3.5 text-zinc-400 shrink-0" />
                    Request Notification Permission
                  </button>
                )}

                {notificationPermission === 'granted' && (
                  <span className="text-[10px] font-mono text-emerald-600 dark:text-emerald-400 flex items-center gap-1.5 font-semibold">
                    <Check className="h-3.5 w-3.5 shrink-0" /> Desktop notifications are fully configured and ready.
                  </span>
                )}

                {notificationPermission === 'denied' && (
                  <span className="text-[10px] font-mono text-amber-600 dark:text-amber-400 flex items-center gap-1.5 leading-normal">
                    <AlertCircle className="h-3.5 w-3.5 shrink-0" /> Notifications are blocked. Please enable them in your browser site settings to receive completion alerts.
                  </span>
                )}
              </div>

              {/* Volume Estimation & Anchor Verification */}
              <div className="flex flex-col gap-4 border border-zinc-200 dark:border-zinc-800 p-5 rounded bg-white dark:bg-zinc-950/10">
                <div className="flex flex-col gap-2.5">
                  <span className="text-xs font-mono font-bold uppercase text-zinc-500">
                    Matching Works & Anchor Validation
                  </span>
                  <p className="text-[11px] text-zinc-400 font-sans leading-relaxed">
                    Query the live OpenAlex API using current search keywords, filters, and anchor DOIs to estimate total papers.
                  </p>

                  <div className="flex items-center gap-2 mt-1">
                    <label className="flex items-center gap-2 cursor-pointer select-none font-mono text-[10px] text-zinc-500 font-bold uppercase">
                      <input
                        type="checkbox"
                        checked={checkAnchors}
                        onChange={() => setCheckAnchors((prev) => !prev)}
                        className="rounded border-zinc-300 dark:border-zinc-800 text-zinc-900 focus:ring-0 cursor-pointer"
                      />
                      <span>Validate Anchor Paper Coverage</span>
                    </label>
                  </div>

                  {/* Count badge */}
                  <div className="min-h-[6rem] py-4 px-3 border border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/10 rounded flex flex-col items-center justify-center gap-1 mt-1">
                    {checkingCount ? (
                      <Loader2 className="h-5 w-5 animate-spin text-zinc-400" />
                    ) : openalexCount !== null ? (
                      <div className="w-full flex flex-col items-center gap-3">
                        <div className="flex flex-col items-center justify-center gap-1">
                          <span className="text-2xl font-mono font-bold text-zinc-900 dark:text-zinc-50">
                            {openalexCount.toLocaleString()}
                          </span>
                          <span className="text-[9px] font-mono text-green-600 dark:text-green-400 uppercase tracking-widest font-bold">
                            MATCHING PAPERS
                          </span>
                        </div>

                        {anchorsTotal !== null && anchorsTotal > 0 && (
                          <div className="border-t border-zinc-200 dark:border-zinc-800 pt-3 mt-1 w-full flex flex-col gap-1.5">
                            <div className="flex justify-between items-center text-[10px] font-mono uppercase tracking-wider text-zinc-400">
                              <span>Anchor Paper Match</span>
                              <span className="font-bold text-zinc-700 dark:text-zinc-300">
                                {anchorsMatched} / {anchorsTotal} (
                                {anchorsTotal > 0
                                  ? Math.round((anchorsMatched! / anchorsTotal) * 100)
                                  : 0}
                                %)
                              </span>
                            </div>
                            <div className="w-full bg-zinc-200 dark:bg-zinc-800 h-1.5 rounded-full overflow-hidden">
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
                  disabled={checkingCount || checkingTopics || !keywords.trim() || queryValid === false}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 border rounded font-mono text-xs font-bold uppercase bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800 hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50 cursor-pointer"
                >
                  {checkingCount ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <Search className="h-3.5 w-3.5" />
                  )}
                  Get Search Volume
                </button>
              </div>
            </div>

            {/* RIGHT COLUMN: Topic Distribution Analysis (cols: 6) */}
            <div className="lg:col-span-6 flex flex-col gap-6">
              <div className="flex flex-col gap-4 border border-zinc-200 dark:border-zinc-800 p-5 rounded bg-white dark:bg-zinc-950/10">
                <div className="flex flex-col gap-2.5">
                  <span className="text-xs font-mono font-bold uppercase text-zinc-500">
                    Topic Grouping & Distribution Analysis
                  </span>
                  <p className="text-[11px] text-zinc-400 font-sans leading-relaxed">
                    Run cursor pagination to fetch sorted counts of research topics matching keywords. Concurrent API requests resolve full names/descriptions.
                  </p>

                  {/* Topics summaries badge */}
                  <div className="min-h-[6rem] py-4 px-3 border border-zinc-200 dark:border-zinc-800 bg-zinc-50/50 dark:bg-zinc-900/10 rounded flex flex-col items-center justify-center gap-1 mt-1">
                    {checkingTopics ? (
                      <Loader2 className="h-5 w-5 animate-spin text-zinc-400" />
                    ) : openalexTopics !== null ? (
                      <div className="w-full flex flex-col items-center gap-2.5">
                        <div className="flex flex-col items-center justify-center gap-1">
                          <span className="text-2xl font-mono font-bold text-zinc-900 dark:text-zinc-50">
                            {openalexTopicsTotal?.toLocaleString() || 0}
                          </span>
                          <span className="text-[9px] font-mono text-zinc-400 uppercase tracking-widest font-bold">
                            UNIQUE TOPICS FOUND
                          </span>
                        </div>
                        <span className="text-[10px] font-sans text-zinc-400">
                          Mapped across <span className="font-semibold text-zinc-700 dark:text-zinc-300">{openalexTopicsTotalPapers?.toLocaleString()}</span> papers.
                        </span>
                      </div>
                    ) : (
                      <span className="text-xs font-mono text-zinc-400 uppercase tracking-wider">
                        No Topic Data Yet
                      </span>
                    )}
                  </div>
                </div>

                <button
                  type="button"
                  onClick={handleGetOpenAlexTopics}
                  disabled={checkingCount || checkingTopics || !keywords.trim() || queryValid === false}
                  className="w-full flex items-center justify-center gap-2 px-3 py-2 border rounded font-mono text-xs font-bold uppercase bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800 hover:bg-zinc-800 dark:hover:bg-zinc-200 disabled:opacity-50 cursor-pointer"
                >
                  {checkingTopics ? (
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  ) : (
                    <PieChart className="h-3.5 w-3.5" />
                  )}
                  Get Topic Distribution
                </button>
              </div>
            </div>

            {/* FULL-WIDTH ROW: OpenAlex Topics Distribution Table (cols: 12) */}
            {openalexTopics !== null && (
              <div className="lg:col-span-12 flex flex-col gap-4 border border-zinc-200 dark:border-zinc-800 p-5 rounded mt-2 w-full bg-white dark:bg-zinc-950/10">
                <div className="flex justify-between items-center border-b border-zinc-200 dark:border-zinc-800 pb-3">
                  <div className="flex flex-col gap-1">
                    <h3 className="text-xs font-mono font-bold uppercase tracking-wider text-zinc-900 dark:text-zinc-50 flex items-center gap-2">
                      <PieChart className="h-4 w-4 text-zinc-500" />
                      OpenAlex Topic Breakdown Results
                    </h3>
                    <p className="text-[11px] text-zinc-400 font-sans">
                      Showing research topics in matching publications.
                    </p>
                  </div>
                  <div className="flex items-center gap-3">
                    <button
                      type="button"
                      onClick={handleDownloadTopicsCSV}
                      className="flex items-center gap-1.5 text-[10px] font-mono uppercase text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-200 cursor-pointer font-bold border border-zinc-200 dark:border-zinc-800 px-2.5 py-1 rounded bg-zinc-50 hover:bg-zinc-100 dark:bg-zinc-900/50 dark:hover:bg-zinc-850 transition-colors"
                    >
                      <Download className="h-3 w-3" />
                      Download CSV
                    </button>
                    <button
                      type="button"
                      onClick={() => setOpenalexTopics(null)}
                      className="text-[10px] font-mono uppercase text-zinc-400 hover:text-zinc-650 dark:hover:text-zinc-200 cursor-pointer"
                    >
                      Clear Results
                    </button>
                  </div>
                </div>
                {openalexTopics.length === 0 ? (
                  <div className="py-8 text-center text-xs font-mono text-zinc-400 uppercase">
                    No topic distribution data retrieved
                  </div>
                ) : (
                  <div className="flex flex-col gap-3 w-full">
                    <div className="overflow-x-auto w-full border border-zinc-100 dark:border-zinc-800 rounded">
                      <table className="w-full text-left border-collapse text-xs font-sans">
                        <thead>
                          <tr className="bg-zinc-50 dark:bg-zinc-900/50 border-b border-zinc-150 dark:border-zinc-800 text-[10px] font-mono uppercase text-zinc-400">
                            <th className="p-3 w-16 text-center font-bold">Target</th>
                            <th className="p-3 w-28">Topic ID</th>
                            <th className="p-3">Topic Name</th>
                            <th className="p-3">Description</th>
                            <th className="p-3 text-right w-32 font-bold">Paper Count</th>
                            <th className="p-3 text-right w-40 font-bold">Percentage</th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-zinc-100 dark:divide-zinc-800">
                          {openalexTopics.slice(0, showAllTopics ? openalexTopics.length : 10).map((topic) => (
                            <tr key={topic.topic_id} className="hover:bg-zinc-50/55 dark:hover:bg-zinc-900/10">
                              <td className="p-3 text-center w-16">
                                <input
                                  type="checkbox"
                                  checked={isTopicSelected(topic.topic_id)}
                                  onChange={(e) => handleToggleTopic(topic.topic_id, e.target.checked)}
                                  className="rounded border-zinc-300 text-zinc-900 focus:ring-0 cursor-pointer"
                                />
                              </td>
                              <td className="p-3 font-mono text-[11px]">
                                <span className="bg-zinc-100 dark:bg-zinc-800 px-1.5 py-0.5 rounded text-zinc-800 dark:text-zinc-300 font-semibold border border-zinc-200/50 dark:border-zinc-800">
                                  {topic.topic_id}
                                </span>
                              </td>
                              <td className="p-3 font-medium text-zinc-900 dark:text-zinc-100 max-w-[200px] truncate font-semibold" title={topic.display_name}>
                                {topic.display_name}
                              </td>
                              <td className="p-3 text-zinc-400 dark:text-zinc-500 max-w-xs truncate" title={topic.description}>
                                {topic.description || '—'}
                              </td>
                              <td className="p-3 text-right font-mono font-medium text-zinc-950 dark:text-zinc-50">
                                {topic.paper_count.toLocaleString()}
                              </td>
                              <td className="p-3">
                                <div className="flex items-center justify-end gap-3 w-full">
                                  <div className="w-20 bg-zinc-200 dark:bg-zinc-800 h-1.5 rounded-full overflow-hidden shrink-0">
                                    <div
                                      className="bg-zinc-900 dark:bg-zinc-100 h-full rounded-full"
                                      style={{ width: `${topic.percentage}%` }}
                                    />
                                  </div>
                                  <span className="font-mono text-[11px] text-zinc-600 dark:text-zinc-400 w-12 text-right">
                                    {topic.percentage.toFixed(2)}%
                                  </span>
                                </div>
                              </td>
                            </tr>
                          ))}
                        </tbody>
                      </table>
                    </div>

                    {openalexTopics.length > 10 && (
                      <button
                        type="button"
                        onClick={() => setShowAllTopics(!showAllTopics)}
                        className="mx-auto mt-2 px-4 py-1.5 border border-zinc-200 dark:border-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-900/50 rounded font-mono text-[10px] uppercase font-bold text-zinc-600 dark:text-zinc-400 cursor-pointer"
                      >
                        {showAllTopics ? 'Show Less (Top 10)' : `Show All ${openalexTopics.length} Topics`}
                      </button>
                    )}
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </form>

      {/* Custom Themed Alert Modal */}
      {alertConfig && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-in fade-in duration-200">
          <div
            className="w-full max-w-sm bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded shadow-xl overflow-hidden animate-in zoom-in-95 duration-200"
            role="alertdialog"
            aria-modal="true"
          >
            {/* Modal Header */}
            <div className="px-5 py-4 border-b border-zinc-200 dark:border-zinc-800 flex items-center gap-2">
              <span
                className={`h-2 w-2 rounded-full ${alertConfig.type === 'success' ? 'bg-emerald-500' : alertConfig.type === 'error' ? 'bg-red-500' : 'bg-blue-500'}`}
              />
              <h3 className="text-xs font-bold font-mono uppercase tracking-wider text-zinc-900 dark:text-zinc-100">
                {alertConfig.title}
              </h3>
            </div>

            {/* Modal Body */}
            <div className="p-5 flex flex-col gap-3">
              <p className="font-mono text-[11px] leading-relaxed text-zinc-700 dark:text-zinc-350 whitespace-pre-line">
                {alertConfig.message}
              </p>
            </div>

            {/* Modal Footer */}
            <div className="px-5 py-3 border-t border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/50 flex items-center justify-end">
              <button
                type="button"
                onClick={() => setAlertConfig(null)}
                className="px-4 py-1.5 bg-zinc-900 dark:bg-zinc-100 hover:bg-zinc-800 dark:hover:bg-zinc-200 text-white dark:text-zinc-950 rounded text-[10px] font-mono font-bold uppercase tracking-wider cursor-pointer shadow transition"
              >
                OK
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
export { Ingest as default }
