# Ingestion & Enrichment Checklist

Run these steps after verifying query precision and coverage:

- [ ] Execute `download` to pull matching OpenAlex papers into `collected_papers.jsonl`.
- [ ] Monitor log updates and hit rates.
- [ ] Initialize the DuckDB schema and insert records using `convert_db`.
- [ ] Run Crossref affiliation imputation (`impute crossref`).
- [ ] Run rule-based country boundary checks.
- [ ] Run LLM affiliation classification (`impute llm`).
- [ ] Run PDF text extraction for remaining missing links (`impute pdf`).
- [ ] Check DuckDB totals and imputation audit logs.
