---
name: arxiv-multiagent
description: "Fetch latest Multiagent Systems papers from arXiv. Use this skill when user asks for the latest arXiv papers on multiagent systems or MAS topics."
metadata: {"nanobot":{"emoji":"📚","requires":{"bins":["curl","python3"]}}}
---

# ArXiv Multiagent Systems

Fetch the latest papers from arXiv's Multiagent Systems category (cs.MA).

## URL

- Main page: https://arxiv.org/list/cs.MA/recent
- RSS feed (recommended): https://export.arxiv.org/api/query?search_query=cat:cs.MA&sortBy=submittedDate&sortOrder=descending&max_results=20

## Fetch Latest Papers (Recommended)

Use the RSS feed for structured data:

```bash
curl -s "https://export.arxiv.org/api/query?search_query=cat:cs.MA&sortBy=submittedDate&sortOrder=descending&max_results=20"
```

## Parse with Python

The project includes a parser script. Run:

```bash
curl -s "https://export.arxiv.org/api/query?search_query=cat:cs.MA&sortBy=submittedDate&sortOrder=descending&max_results=20" | python3 parse_arxiv.py
```

Note: The parser expects XML input via stdin and outputs markdown formatted results.

## HTML Page (Fallback)

If RSS is unavailable, fetch the HTML page:

```bash
curl -s "https://arxiv.org/list/cs.MA/recent"
```

Then parse manually - look for paper titles in `<a href="/abs/...">` tags and abstracts in `<span class="abstract-full">`.

## Output Format

The parser outputs:
- Paper title
- Authors
- Published date
- Categories
- arXiv link
- PDF link (if available)
- Summary/abstract (truncated to 2000 chars if too long)

## Example Output

```markdown
### [Paper Title]
- **Authors**: Author1, Author2
- **Published**: 2026-02-25
- **Categories**: cs.MA, cs.AI
- **Link**: http://arxiv.org/abs/xxx.xxxxx
- **PDF**: http://arxiv.org/pdf/xxx.xxxxx
- **Summary**: This paper discusses...
```

## Tips

- `max_results=N` controls number of papers (default 20)
- Add `&start=N` to skip first N results for pagination
- Today's papers: check the main HTML page for "new" submissions
