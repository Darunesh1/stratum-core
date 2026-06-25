// src/routes/wos.tsx
import { useState } from 'react'
import { Upload, Play, Terminal, Database, Key } from 'lucide-react'

export function Wos() {
  const [imputeProvider, setImputeProvider] = useState<'gemini' | 'ollama'>('gemini')
  const [modelName, setModelName] = useState('gemini-1.5-flash')
  const [apiKey, setApiKey] = useState('')
  const [ollamaURL, setOllamaURL] = useState('http://localhost:11434')

  return (
    <div className="flex flex-col gap-8 w-full font-sans">
      {/* Header Row */}
      <div className="flex flex-col gap-1 border-b border-zinc-200 pb-5 dark:border-zinc-850">
        <h1 className="text-2xl font-mono font-bold tracking-tight text-zinc-950 dark:text-zinc-50 uppercase">
          Web of Science & Imputation Studio
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          Import local WoS catalogs, map relational authorship contributions, and execute LLM country imputation pipelines.
        </p>
      </div>

      {/* Grid Layout: Import & Settings */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6 items-start">
        {/* LEFT COLUMN: Data Import (cols: 6) */}
        <div className="lg:col-span-6 flex flex-col gap-6">
          <div className="flex flex-col gap-4 border border-zinc-200 dark:border-zinc-800 p-5 rounded bg-white dark:bg-zinc-950/10">
            <div className="flex flex-col gap-2.5">
              <span className="text-xs font-mono font-bold uppercase text-zinc-500">
                1. Web of Science Data Import
              </span>
              <p className="text-[11px] text-zinc-400 font-sans leading-relaxed">
                Upload raw WoS export files (plain text formats, CSVs, or Excel spreadsheets) to parse records, extract DOIs, and stage them for contributor coverage checking.
              </p>

              {/* Upload Box */}
              <div className="border border-dashed border-zinc-300 dark:border-zinc-800 rounded p-6 flex flex-col items-center justify-center gap-3 bg-zinc-50/50 dark:bg-zinc-900/10 hover:bg-zinc-50 dark:hover:bg-zinc-900/30 transition cursor-pointer mt-1">
                <Upload className="h-6 w-6 text-zinc-400" />
                <span className="text-xs font-mono font-bold text-zinc-600 dark:text-zinc-400 uppercase">
                  Choose WoS catalog file...
                </span>
                <span className="text-[10px] text-zinc-400">
                  Accepts .txt (WoS Plain Text), .csv, .xls, or .xlsx
                </span>
              </div>

              {/* Schema mapping fields */}
              <div className="grid grid-cols-2 gap-3 mt-2">
                <div className="flex flex-col gap-1">
                  <label className="text-[9px] font-mono font-bold uppercase tracking-wider text-zinc-450">
                    DOI Column
                  </label>
                  <select disabled className="w-full bg-zinc-50 dark:bg-zinc-900/50 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs text-zinc-400">
                    <option>Select column...</option>
                  </select>
                </div>
                <div className="flex flex-col gap-1">
                  <label className="text-[9px] font-mono font-bold uppercase tracking-wider text-zinc-450">
                    Author Column
                  </label>
                  <select disabled className="w-full bg-zinc-50 dark:bg-zinc-900/50 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs text-zinc-400">
                    <option>Select column...</option>
                  </select>
                </div>
              </div>
            </div>

            <button
              type="button"
              disabled
              className="w-full flex items-center justify-center gap-2 mt-2 px-3 py-2 border border-zinc-250 dark:border-zinc-800 rounded font-mono text-xs font-bold uppercase bg-zinc-100 dark:bg-zinc-900 text-zinc-400 dark:text-zinc-500 cursor-not-allowed select-none"
            >
              <Database className="h-3.5 w-3.5 shrink-0" />
              Import & Parse Catalog
            </button>
          </div>
        </div>

        {/* RIGHT COLUMN: Imputation Configuration (cols: 6) */}
        <div className="lg:col-span-6 flex flex-col gap-6">
          <div className="flex flex-col gap-4 border border-zinc-200 dark:border-zinc-800 p-5 rounded bg-white dark:bg-zinc-950/10">
            <div className="flex flex-col gap-2.5">
              <span className="text-xs font-mono font-bold uppercase text-zinc-500">
                2. Affiliation Imputation Engine
              </span>
              <p className="text-[11px] text-zinc-400 font-sans leading-relaxed">
                Configure LLM credentials to scan paper authorship entries with missing institution countries and impute correct ISO country codes based on raw affiliation strings.
              </p>

              {/* Provider Selection */}
              <div className="flex flex-col gap-1 mt-1">
                <label className="text-[9px] font-mono font-bold uppercase tracking-wider text-zinc-450">
                  LLM Provider
                </label>
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => {
                      setImputeProvider('gemini')
                      setModelName('gemini-1.5-flash')
                    }}
                    className={`flex-1 py-1.5 border rounded font-mono text-[10px] font-bold uppercase transition ${
                      imputeProvider === 'gemini'
                        ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800'
                        : 'bg-transparent text-zinc-500 border-zinc-200 dark:border-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-900'
                    }`}
                  >
                    Gemini API
                  </button>
                  <button
                    type="button"
                    onClick={() => {
                      setImputeProvider('ollama')
                      setModelName('llama3')
                    }}
                    className={`flex-1 py-1.5 border rounded font-mono text-[10px] font-bold uppercase transition ${
                      imputeProvider === 'ollama'
                        ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 border-zinc-800'
                        : 'bg-transparent text-zinc-500 border-zinc-200 dark:border-zinc-800 hover:bg-zinc-50 dark:hover:bg-zinc-900'
                    }`}
                  >
                    Ollama (Local)
                  </button>
                </div>
              </div>

              {/* Model Choice */}
              <div className="flex flex-col gap-1">
                <label className="text-[9px] font-mono font-bold uppercase tracking-wider text-zinc-450">
                  Model Selection
                </label>
                {imputeProvider === 'gemini' ? (
                  <select
                    value={modelName}
                    onChange={(e) => setModelName(e.target.value)}
                    className="w-full bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs focus:outline-none"
                  >
                    <option value="gemini-1.5-flash">Gemini 1.5 Flash (Recommended)</option>
                    <option value="gemini-1.5-pro">Gemini 1.5 Pro</option>
                  </select>
                ) : (
                  <input
                    type="text"
                    value={modelName}
                    onChange={(e) => setModelName(e.target.value)}
                    className="w-full bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs focus:outline-none"
                    placeholder="e.g. llama3, mistral"
                  />
                )}
              </div>

              {/* Provider Config Fields */}
              {imputeProvider === 'gemini' ? (
                <div className="flex flex-col gap-1">
                  <label className="text-[9px] font-mono font-bold uppercase tracking-wider text-zinc-450 flex items-center gap-1">
                    <Key className="h-2.5 w-2.5" /> Gemini API Key
                  </label>
                  <input
                    type="password"
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    className="w-full bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs focus:outline-none"
                    placeholder="Enter GEMINI_API_KEY..."
                  />
                </div>
              ) : (
                <div className="flex flex-col gap-1">
                  <label className="text-[9px] font-mono font-bold uppercase tracking-wider text-zinc-450">
                    Ollama Base URL
                  </label>
                  <input
                    type="text"
                    value={ollamaURL}
                    onChange={(e) => setOllamaURL(e.target.value)}
                    className="w-full bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-1.5 rounded font-mono text-xs focus:outline-none"
                    placeholder="http://localhost:11434"
                  />
                </div>
              )}
            </div>

            <button
              type="button"
              disabled
              className="w-full flex items-center justify-center gap-2 mt-2 px-3 py-2 border border-zinc-250 dark:border-zinc-800 rounded font-mono text-xs font-bold uppercase bg-zinc-100 dark:bg-zinc-900 text-zinc-400 dark:text-zinc-500 cursor-not-allowed select-none"
            >
              <Play className="h-3.5 w-3.5 shrink-0" />
              Run Imputation Pipeline
            </button>
          </div>
        </div>
      </div>

      {/* Bottom Section: Logs console */}
      <div className="flex flex-col gap-4 border border-zinc-200 dark:border-zinc-800 p-5 rounded bg-white dark:bg-zinc-950/10">
        <div className="flex items-center justify-between border-b border-zinc-200 dark:border-zinc-850 pb-3">
          <div className="flex items-center gap-2">
            <Terminal className="h-4 w-4 text-zinc-400 shrink-0" />
            <span className="text-xs font-mono font-bold uppercase text-zinc-500">
              3. Imputation Diagnostics & Logs
            </span>
          </div>
          <span className="px-2 py-0.5 rounded bg-zinc-100 dark:bg-zinc-900 border border-zinc-250 dark:border-zinc-800 text-zinc-500 font-mono text-[9px] font-bold uppercase">
            Standby
          </span>
        </div>

        <div className="bg-zinc-950 text-zinc-100 p-4 rounded font-mono text-[10px] min-h-[10rem] flex flex-col gap-1.5 leading-relaxed overflow-y-auto max-h-60 border border-zinc-900 text-left">
          <span className="text-zinc-500">
            [{new Date().toLocaleTimeString()}] [INFO] Ready for WoS data import and imputation.
          </span>
          <span className="text-zinc-500">
            [{new Date().toLocaleTimeString()}] [INFO] Stage a Web of Science export catalog to trigger diagnostics checking.
          </span>
        </div>
      </div>
    </div>
  )
}
