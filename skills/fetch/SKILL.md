---
name: fetch
description: "Fetch and parse web page content. Use this skill to get content from URLs, including HTML, JSON, or plain text."
metadata: {"nanobot":{"emoji":"🌐","requires":{"bins":["curl","lynx"]}}}
---

# Fetch Skill

Use `curl` and `lynx` to fetch web page content.

## Basic Usage

Fetch a URL and output as plain text:
```bash
curl -s "https://example.com"
```

Fetch and display only text (strip HTML):
```bash
lynx -dump -nolist "https://example.com"
```

## JSON APIs

Fetch JSON and format with jq:
```bash
curl -s "https://api.example.com/data" | jq '.'
```

Pretty print JSON:
```bash
curl -s "https://api.example.com/data" | jq .
```

## Headers

Show response headers:
```bash
curl -I "https://example.com"
```

Send custom headers:
```bash
curl -H "Accept: application/json" -H "Authorization: Bearer token" "https://api.example.com"
```

## POST Requests

Send POST request with JSON body:
```bash
curl -X POST -H "Content-Type: application/json" -d '{"key":"value"}' "https://api.example.com"
```

## Download Files

Download file:
```bash
curl -O "https://example.com/file.zip"
```

Save to specific filename:
```bash
curl -o "myfile.zip" "https://example.com/file.zip"
```

## Tips

- Use `-s` for silent mode (no progress bar)
- Use `-L` to follow redirects
- Use `-A` to set User-Agent
- Use `-k` to allow insecure SSL connections
