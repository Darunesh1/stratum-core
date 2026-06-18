// src/routes/docs.tsx
import { useState } from 'react'
import { BookOpen, ArrowRight, HelpCircle, Code2, Database } from 'lucide-react'

interface DocGuide {
  id: string
  title: string
  subtitle: string
  icon: React.ReactNode
  content: React.ReactNode
}

export function Docs() {
  const [activeGuideId, setActiveGuideId] = useState('ingestion')

  const guides: DocGuide[] = [
    {
      id: 'ingestion',
      title: 'OpenAlex Ingestion Flow',
      subtitle: 'Data pipeline architecture',
      icon: <HelpCircle className="h-4 w-4" />,
      content: (
        <div className="flex flex-col gap-5 text-sm leading-relaxed">
          <h2 className="text-lg font-mono font-bold border-b border-zinc-200 dark:border-zinc-800 pb-2 uppercase tracking-wide">
            Pipeline Architecture
          </h2>
          <p>
            The OpenAlex Ingest engine downloads and structures academic research works according to
            keywords and topics filters.
          </p>

          <h3 className="font-mono font-bold uppercase text-xs text-zinc-500">Pipeline Stages</h3>
          <ol className="list-decimal pl-5 flex flex-col gap-2.5">
            <li>
              <strong>Validate filters</strong>: Checks boolean expression grammar and parses
              OpenAlex topic IDs (TXXXXX) stored in the project's SQLite <code>config.db</code>.
            </li>
            <li>
              <strong>Retrieve counts</strong>: Queries OpenAlex API endpoint to calculate the
              matching papers count before downloading.
            </li>
            <li>
              <strong>Concurrent download</strong>: Initiates parallel worker pools downloading
              JSONL assets concurrently, tracking progress using cursors.
            </li>
            <li>
              <strong>DOI Catalog Upload & Extraction</strong>: Supports CSV, XLSX, and legacy XLS
              formats for keyword mining and seed DOI extraction. Features a robust row-scanning
              fallback to bypass parser column shifts and retrieve all DOIs successfully.
            </li>
          </ol>

          <div className="p-4 border border-zinc-200 bg-zinc-50/50 dark:border-zinc-850 dark:bg-zinc-900/10 font-mono text-[11px] leading-relaxed">
            <span className="font-bold text-zinc-800 dark:text-zinc-200 uppercase tracking-wide block mb-1">
              Configuration Storage
            </span>
            <span>Database: projects/&#123;project-slug&#125;/config.db</span>
            <br />
            <span>
              Output Database: projects/&#123;project-slug&#125;/&#123;project-slug&#125;.db
            </span>
          </div>
        </div>
      ),
    },
    {
      id: 'imputation',
      title: 'Affiliation Imputation',
      subtitle: 'Metadata restoration guide',
      icon: <BookOpen className="h-4 w-4" />,
      content: (
        <div className="flex flex-col gap-5 text-sm leading-relaxed">
          <h2 className="text-lg font-mono font-bold border-b border-zinc-200 dark:border-zinc-800 pb-2 uppercase tracking-wide">
            Affiliation Imputation Engine
          </h2>
          <p>
            When papers lack institution mapping or country codes, the Imputations engine runs three
            stages:
          </p>

          <h3 className="font-mono font-bold uppercase text-xs text-zinc-500">
            Imputation Pipeline Stages
          </h3>
          <ol className="list-decimal pl-5 flex flex-col gap-3">
            <li>
              <strong>Stage 1: Crossref Lookups</strong>
              <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1 font-sans">
                Queries Crossref works metadata using DOIs. Matches authors via ORCIDs or position
                indexes to resolve original affiliation strings.
              </p>
            </li>
            <li>
              <strong>Stage 2: LLM Named Entity Extraction</strong>
              <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1 font-sans">
                Submits unresolved strings to local Ollama (e.g. Llama-3) or Gemini API. Extracts
                company/university names and alpha-2 country codes.
              </p>
            </li>
            <li>
              <strong>Stage 3: Rule-Based Fallback</strong>
              <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1 font-sans">
                Runs regular expressions and word boundaries matching against country names/aliases
                list to resolve remaining country codes.
              </p>
            </li>
            <li>
              <strong>Stage 4: PDF Extraction (Optional)</strong>
              <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1 font-sans">
                Downloads article PDFs, decompresses first-page object streams, extracts text, and
                queries LLMs to match affiliations.
              </p>
            </li>
          </ol>
        </div>
      ),
    },
    {
      id: 'schema',
      title: 'SQL Schema Reference',
      subtitle: 'DuckDB database design',
      icon: <Database className="h-4 w-4" />,
      content: (
        <div className="flex flex-col gap-5 text-sm leading-relaxed">
          <h2 className="text-lg font-mono font-bold border-b border-zinc-200 dark:border-zinc-800 pb-2 uppercase tracking-wide">
            Relational DuckDB Schema
          </h2>
          <p>DuckDB tables store academic indexes. Below are the structural descriptions:</p>

          <div className="overflow-x-auto border border-zinc-200 dark:border-zinc-800 rounded">
            <table className="w-full border-collapse text-left font-mono text-[11px]">
              <thead>
                <tr className="bg-zinc-50 dark:bg-zinc-900 border-b border-zinc-200 dark:border-zinc-800 text-zinc-500">
                  <th className="p-2.5 border-r border-zinc-200 dark:border-zinc-800 font-bold uppercase">
                    Table
                  </th>
                  <th className="p-2.5 border-r border-zinc-200 dark:border-zinc-800 font-bold uppercase">
                    Columns
                  </th>
                  <th className="p-2.5 font-bold uppercase">Primary Keys / Indexes</th>
                </tr>
              </thead>
              <tbody>
                <tr className="border-b border-zinc-150 dark:border-zinc-850 hover:bg-zinc-50/40">
                  <td className="p-2.5 border-r border-zinc-200 dark:border-zinc-800 font-bold text-zinc-800 dark:text-zinc-200">
                    papers
                  </td>
                  <td className="p-2.5 border-r border-zinc-200 dark:border-zinc-800">
                    id, doi, title, publication_year, journal_name, is_oa, cited_by_count, fwci
                  </td>
                  <td className="p-2.5">id (VARCHAR) PRIMARY KEY</td>
                </tr>
                <tr className="border-b border-zinc-150 dark:border-zinc-850 hover:bg-zinc-50/40">
                  <td className="p-2.5 border-r border-zinc-200 dark:border-zinc-800 font-bold text-zinc-800 dark:text-zinc-200">
                    contributions
                  </td>
                  <td className="p-2.5 border-r border-zinc-200 dark:border-zinc-800">
                    row_id, paper_id, author_id, institution_id, country_code, author_position
                  </td>
                  <td className="p-2.5">row_id (INTEGER) PRIMARY KEY</td>
                </tr>
                <tr className="hover:bg-zinc-50/40">
                  <td className="p-2.5 border-r border-zinc-200 dark:border-zinc-800 font-bold text-zinc-800 dark:text-zinc-200">
                    institutions
                  </td>
                  <td className="p-2.5 border-r border-zinc-200 dark:border-zinc-800">
                    id, display_name, country_code, type, is_synthetic
                  </td>
                  <td className="p-2.5">id (VARCHAR) PRIMARY KEY</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      ),
    },
    {
      id: 'api',
      title: 'REST & MCP API',
      subtitle: 'Integration specifications',
      icon: <Code2 className="h-4 w-4" />,
      content: (
        <div className="flex flex-col gap-5 text-sm leading-relaxed">
          <h2 className="text-lg font-mono font-bold border-b border-zinc-200 dark:border-zinc-800 pb-2 uppercase tracking-wide">
            System Integration
          </h2>
          <p>
            Stratum operates as an MCP stdio server for AI clients or as an HTTP daemon serving
            dashboard APIs.
          </p>

          <h3 className="font-mono font-bold uppercase text-xs text-zinc-500">
            MCP Tool Definitions
          </h3>
          <ul className="list-disc pl-5 flex flex-col gap-2">
            <li>
              <code>validate</code>: Parses SQLite config settings and reports validation status.
            </li>
            <li>
              <code>search</code>: Returns count of matching works.
            </li>
            <li>
              <code>download</code>: Pulls works and saves to JSONL output path.
            </li>
            <li>
              <code>convert_db</code>: Ingests JSONL database records into DuckDB.
            </li>
            <li>
              <code>impute</code>: Triggers affiliation and country code imputation pipelines.
            </li>
          </ul>

          <h3 className="font-mono font-bold uppercase text-xs text-zinc-500">REST Endpoints</h3>
          <div className="p-4 border border-zinc-200 bg-zinc-50/50 dark:border-zinc-850 dark:bg-zinc-900/10 font-mono text-[11px] leading-relaxed flex flex-col gap-1.5">
            <div>
              <span className="text-zinc-900 dark:text-zinc-100 font-bold">GET /api/projects</span>{' '}
              — Lists all dynamic project technology workspaces
            </div>
            <div>
              <span className="text-zinc-900 dark:text-zinc-100 font-bold">
                POST /api/projects/create
              </span>{' '}
              — Creates a new project database and configuration workspace
            </div>
            <div>
              <span className="text-zinc-900 dark:text-zinc-100 font-bold">POST /api/upload</span> —
              Uploads catalog files (.csv, .xlsx, .xls) and returns column headers
            </div>
            <div>
              <span className="text-zinc-900 dark:text-zinc-100 font-bold">GET /api/stats</span> —
              Returns paper database summary metrics and counts
            </div>
            <div>
              <span className="text-zinc-900 dark:text-zinc-100 font-bold">GET /api/config</span> —
              Retrieves current project settings and anchor DOIs
            </div>
            <div>
              <span className="text-zinc-900 dark:text-zinc-100 font-bold">POST /api/config</span> —
              Saves updated configuration revisions to the sqlite database
            </div>
            <div>
              <span className="text-zinc-900 dark:text-zinc-100 font-bold">POST /api/query</span> —
              Submits a query to the dynamic DuckDB database and returns JSON rows
            </div>
          </div>
        </div>
      ),
    },
  ]

  const activeGuide = guides.find((g) => g.id === activeGuideId) || guides[0]

  return (
    <div className="flex flex-col gap-8 w-full">
      {/* Header Row */}
      <div className="flex flex-col gap-1 border-b border-zinc-200 pb-5 dark:border-zinc-850">
        <h1 className="text-2xl font-mono font-bold tracking-tight text-zinc-950 dark:text-zinc-50 uppercase">
          Developer Documentation
        </h1>
        <p className="text-sm text-zinc-500 dark:text-zinc-400">
          Integrated user manuals, pipeline descriptions, schema structures, and API references.
        </p>
      </div>

      {/* Side-by-Side Reading Layout */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-6 items-start">
        {/* Left Side: Navigation Sidebar */}
        <div className="md:col-span-1 flex flex-col gap-1 border border-zinc-200 dark:border-zinc-850 bg-zinc-50/40 dark:bg-zinc-900/10 p-2.5 rounded">
          {guides.map((guide) => (
            <button
              key={guide.id}
              onClick={() => setActiveGuideId(guide.id)}
              className={`w-full flex items-center justify-between px-3 py-2.5 rounded text-left transition font-mono text-xs select-none ${
                activeGuideId === guide.id
                  ? 'bg-zinc-900 text-white dark:bg-zinc-100 dark:text-zinc-950 font-bold'
                  : 'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-900/60'
              }`}
            >
              <div className="flex items-center gap-2.5 truncate">
                <span className="shrink-0">{guide.icon}</span>
                <span className="truncate">{guide.title}</span>
              </div>
              <ArrowRight
                className={`h-3 w-3 shrink-0 ${activeGuideId === guide.id ? 'opacity-100' : 'opacity-0'}`}
              />
            </button>
          ))}
        </div>

        {/* Right Side: Read Panel */}
        <div className="md:col-span-3 border border-zinc-200 dark:border-zinc-850 p-6 rounded select-text">
          <div className="flex flex-col gap-1.5 border-b border-zinc-200 dark:border-zinc-850 pb-4 mb-6">
            <h1 className="text-xl font-mono font-bold tracking-tight text-zinc-900 dark:text-zinc-50 uppercase">
              {activeGuide.title}
            </h1>
            <p className="text-xs text-zinc-500 dark:text-zinc-400 uppercase tracking-wider font-mono font-semibold">
              {activeGuide.subtitle}
            </p>
          </div>
          {activeGuide.content}
        </div>
      </div>
    </div>
  )
}
export { Docs as default }
