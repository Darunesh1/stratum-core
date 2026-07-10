# Anchor Paper Validation

Anchor papers are verified, high-impact publications that **must** be present in the final target dataset.

## Role of Anchors
*   **Recall Proxy**: If your query misses anchor papers, it indicates that your boolean search filters are too narrow or exclude critical keywords.
*   **Refinement Gate**: We test anchor coverage *before* launching the full download pipeline.
*   **Addressing Indexing Failures**: Sometimes, a missing anchor does not mean the query is wrong; it might indicate the DOI is missing from OpenAlex. Check indexing explicitly using the search filters.
