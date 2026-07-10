# Staged Random Sampling

Before executing downloads that consume time, bandwidth, and API limits, we run random sampling checks.

## Sampling Strategy
*   **Precision Check**: Extract a random set of 50-100 papers from the query hit list.
*   **Evaluate Noise**: Analyze titles, journals, and topics to check if unrelated fields (e.g. quantum chemical chemistry vs. quantum hardware) are polluting the hits.
*   **Dynamic Stage Checks**: Run sampling after keyword definition, and again after topic filtering, to isolate what step introduced noise.
