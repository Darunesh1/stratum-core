# OpenAlex Topic Filtering

OpenAlex classifies works into hierarchical topic codes.

## Filtering Philosophy
*   **Identify Core Disciplines**: Query the top topics matching your keywords using `get_topics`.
*   **Filter Outlying Noise**: Add topic filters (`primary_topic.id`) to exclude disciplines that match search terms but represent different fields.
*   **Balanced recall**: Restricting to topics improves precision, but can hurt recall. Check anchor coverage to verify that topic bounds are not too restrictive.
