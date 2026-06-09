// src/lib/mock-stratum.ts

export interface MockPaper {
  id: string
  title: string
  doi: string
  publication_year: number
  journal_name: string
  is_oa: boolean
  cited_by_count: number
  fwci: number
}

export interface MockContribution {
  row_id: number
  paper_id: string
  author_name: string
  institution_name: string
  country_code: string
  author_position: string
}

export interface DashboardMetrics {
  totalPapers: number
  openAccessCount: number
  imputedInstitutions: number
  unresolvedAffiliations: number
}

// 1. Dashboard Metrics
export const mockMetrics: DashboardMetrics = {
  totalPapers: 12450,
  openAccessCount: 8715,
  imputedInstitutions: 3820,
  unresolvedAffiliations: 142,
}

// 2. Mock Papers Data
export const mockPapers: MockPaper[] = [
  {
    id: 'w4389021481',
    title: 'Model Context Protocol: A Standardized API for AI Agent Tooling',
    doi: '10.48550/arxiv.2406.12345',
    publication_year: 2024,
    journal_name: 'arXiv preprint',
    is_oa: true,
    cited_by_count: 48,
    fwci: 2.45,
  },
  {
    id: 'w4312984920',
    title: 'Fuzzy String Alignment at Scale for Academic Metadata Cross-Linking',
    doi: '10.1016/j.joi.2023.101452',
    publication_year: 2023,
    journal_name: 'Journal of Informetrics',
    is_oa: false,
    cited_by_count: 12,
    fwci: 1.15,
  },
  {
    id: 'w4289043211',
    title: 'Measuring Scientific Impact: OpenAccess vs Paywalled Distribution Models',
    doi: '10.1371/journal.pone.0289124',
    publication_year: 2022,
    journal_name: 'PLOS ONE',
    is_oa: true,
    cited_by_count: 89,
    fwci: 1.62,
  },
  {
    id: 'w4392019482',
    title: 'Ingestion of Semi-Structured JSONL In Bibliometric Databases',
    doi: '10.1109/tse.2024.3214521',
    publication_year: 2024,
    journal_name: 'IEEE Transactions on Software Engineering',
    is_oa: false,
    cited_by_count: 5,
    fwci: 0.98,
  },
  {
    id: 'w4210984931',
    title: 'Pure Go In-Memory PDF Parser using Object Stream Decompression',
    doi: '10.5281/zenodo.5123984',
    publication_year: 2021,
    journal_name: 'SoftwareX',
    is_oa: true,
    cited_by_count: 22,
    fwci: 1.41,
  },
]

// 3. Mock Contributions Data
export const mockContributions: MockContribution[] = [
  {
    row_id: 1,
    paper_id: 'w4389021481',
    author_name: 'John Doe',
    institution_name: 'Stanford University',
    country_code: 'US',
    author_position: 'first',
  },
  {
    row_id: 2,
    paper_id: 'w4389021481',
    author_name: 'Jane Smith',
    institution_name: 'Tsinghua University',
    country_code: 'CN',
    author_position: 'last',
  },
  {
    row_id: 3,
    paper_id: 'w4312984920',
    author_name: 'Hiroshi Tanaka',
    institution_name: 'University of Tokyo',
    country_code: 'JP',
    author_position: 'first',
  },
  {
    row_id: 4,
    paper_id: 'w4289043211',
    author_name: 'Elena Rostova',
    institution_name: 'Sorbonne University',
    country_code: 'FR',
    author_position: 'middle',
  },
]

// 4. Mock SQL Schema Info
export interface TableField {
  name: string
  type: string
  description?: string
}

export interface TableSchema {
  name: string
  fields: TableField[]
}

export const mockSchemas: TableSchema[] = [
  {
    name: 'papers',
    fields: [
      { name: 'id', type: 'VARCHAR', description: 'Primary OpenAlex ID (e.g. w4389021481)' },
      { name: 'doi', type: 'VARCHAR', description: 'Digital Object Identifier' },
      { name: 'title', type: 'TEXT', description: 'Cleaned English title' },
      { name: 'publication_year', type: 'INTEGER', description: 'Year of publication' },
      { name: 'journal_name', type: 'VARCHAR', description: 'Source journal description' },
      { name: 'is_oa', type: 'BOOLEAN', description: 'Open Access flag status' },
      { name: 'cited_by_count', type: 'INTEGER', description: 'Total incoming citations' },
      { name: 'fwci', type: 'DOUBLE', description: 'Field Weighted Citation Impact' },
    ],
  },
  {
    name: 'contributions',
    fields: [
      { name: 'row_id', type: 'INTEGER', description: 'Unique primary row ID' },
      { name: 'paper_id', type: 'VARCHAR', description: 'Linked paper.id value' },
      { name: 'author_id', type: 'VARCHAR', description: 'Linked author.id value' },
      { name: 'institution_id', type: 'VARCHAR', description: 'Linked institution ID' },
      { name: 'country_code', type: 'VARCHAR', description: 'ISO-3166-1 alpha-2 code' },
      { name: 'author_name', type: 'VARCHAR', description: 'Author display name' },
      { name: 'author_position', type: 'VARCHAR', description: 'first, last, or middle' },
      {
        name: 'raw_affiliation_string',
        type: 'VARCHAR',
        description: 'Original parsed metadata string',
      },
    ],
  },
  {
    name: 'institutions',
    fields: [
      { name: 'id', type: 'VARCHAR', description: 'Primary ID or synthetic IMP_ prefix' },
      { name: 'display_name', type: 'VARCHAR', description: 'Standard organization name' },
      { name: 'country_code', type: 'VARCHAR', description: 'Country affiliation code' },
      { name: 'type', type: 'VARCHAR', description: 'education, facility, healthcare, etc.' },
      { name: 'is_synthetic', type: 'BOOLEAN', description: 'True if generated from imputation' },
    ],
  },
]

// 5. Default Mock SQL Queries
export const mockQueries = [
  {
    label: 'Publication Year Distribution',
    sql: `SELECT publication_year,\n       COUNT(*) as total_papers,\n       SUM(CASE WHEN is_oa = TRUE THEN 1 ELSE 0 END) as open_access\nFROM papers\nGROUP BY publication_year\nORDER BY publication_year DESC;`,
  },
  {
    label: 'Top 10 Affiliated Countries',
    sql: `SELECT country_code,\n       COUNT(*) as contribution_count\nFROM contributions\nWHERE country_code IS NOT NULL\nGROUP BY country_code\nORDER BY contribution_count DESC\nLIMIT 10;`,
  },
  {
    label: 'List Synthetic Imputed Institutions',
    sql: `SELECT id, display_name, country_code, type\nFROM institutions\nWHERE is_synthetic = TRUE\nORDER BY display_name ASC\nLIMIT 5;`,
  },
]

// Helper function to execute mock query and return rows and columns
export function executeMockQuery(sqlQuery: string): {
  columns: string[]
  rows: Record<string, string | number | boolean>[]
} {
  const queryClean = sqlQuery.toLowerCase().replace(/\s+/g, ' ')

  if (queryClean.includes('publication_year')) {
    return {
      columns: ['publication_year', 'total_papers', 'open_access'],
      rows: [
        { publication_year: 2024, total_papers: 5820, open_access: 4120 },
        { publication_year: 2023, total_papers: 4100, open_access: 2900 },
        { publication_year: 2022, total_papers: 2120, open_access: 1450 },
        { publication_year: 2021, total_papers: 410, open_access: 245 },
      ],
    }
  }

  if (queryClean.includes('country_code')) {
    return {
      columns: ['country_code', 'contribution_count'],
      rows: [
        { country_code: 'US', contribution_count: 5124 },
        { country_code: 'CN', contribution_count: 3892 },
        { country_code: 'GB', contribution_count: 1452 },
        { country_code: 'DE', contribution_count: 981 },
        { country_code: 'JP', contribution_count: 820 },
        { country_code: 'FR', contribution_count: 754 },
        { country_code: 'CA', contribution_count: 512 },
        { country_code: 'IN', contribution_count: 489 },
        { country_code: 'KR', contribution_count: 310 },
        { country_code: 'AU', contribution_count: 298 },
      ],
    }
  }

  if (queryClean.includes('is_synthetic') || queryClean.includes('synthetic')) {
    return {
      columns: ['id', 'display_name', 'country_code', 'type'],
      rows: [
        {
          id: 'IMP_a1b2c3d4e5',
          display_name: 'Advanced AI Labs',
          country_code: 'US',
          type: 'company',
        },
        {
          id: 'IMP_f6g7h8i9j0',
          display_name: 'Munich Quantum Research Hub',
          country_code: 'DE',
          type: 'education',
        },
        {
          id: 'IMP_k1l2m3n4o5',
          display_name: 'Kyoto Robotics Institute',
          country_code: 'JP',
          type: 'facility',
        },
        {
          id: 'IMP_p6q7r8s9t0',
          display_name: 'Paris Neural Networks Group',
          country_code: 'FR',
          type: 'education',
        },
        {
          id: 'IMP_u1v2w3x4y5',
          display_name: 'Ontario Climate Consortium',
          country_code: 'CA',
          type: 'nonprofit',
        },
      ],
    }
  }

  // Fallback: return papers
  return {
    columns: ['id', 'title', 'doi', 'publication_year', 'journal_name'],
    rows: mockPapers.map((p) => ({
      id: p.id,
      title: p.title,
      doi: p.doi,
      publication_year: p.publication_year,
      journal_name: p.journal_name,
    })),
  }
}
