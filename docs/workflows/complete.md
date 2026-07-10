# Complete Ingestion & Imputation Workflow

This workflow represents the end-to-end cycle to establish a project database:

1.  **Configure**: Set project metadata (API keys, email contact).
2.  **Upload**: Import reference file (`upload_file`).
3.  **Analyze**: Run keywords/anchors extraction (`extract_query_and_anchors`).
4.  **Refine**: Test hits, review topic tables, and edit config.
5.  **Validate**: Verify anchors are covered and syntax is correct.
6.  **Ingest**: Start concurrent downloads (`download`) and load records to DuckDB (`convert_db`).
7.  **Enrich**: Run imputation (`impute`) with Crossref first, then LLM, then PDF.
8.  **Analyze**: Verify database records counts and execute custom analytical queries.
