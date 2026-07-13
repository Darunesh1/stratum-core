# Query Refinement Strategies

If validation checks indicate low precision or missing anchors, run query refinement.

## Actions to Take
*   **Missing Anchors**: Broaden the search by adding terms with OR, or removing restrictive topic codes.
*   **Irrelevant Noise**: Narrow the search by appending negative keyword exclusions (e.g. `NOT "quantum chemistry"`) or restricting to selected topic lists.
*   **Verify Changes**: Check the hit count after every change to ensure they produce logical scaling before downloading.
