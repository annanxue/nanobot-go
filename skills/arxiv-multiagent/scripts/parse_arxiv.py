#!/usr/bin/env python3
import sys
import xml.etree.ElementTree as ET
from datetime import datetime

# Read XML from stdin
xml_content = sys.stdin.read()
root = ET.fromstring(xml_content)

# Define namespace
ns = {'atom': 'http://www.w3.org/2005/Atom'}

entries = root.findall('atom:entry', ns)
print(f"Found {len(entries)} entries")
for entry in entries:
    # Skip error entries
    title_elem = entry.find('atom:title', ns)
    if title_elem is None or title_elem.text == 'Error':
        continue
    title = title_elem.text.strip()
    summary_elem = entry.find('atom:summary', ns)
    summary = summary_elem.text.strip() if summary_elem is not None else ''
    id_elem = entry.find('atom:id', ns)
    arxiv_id = id_elem.text.strip() if id_elem is not None else ''
    # Get PDF link
    pdf_link = ''
    for link in entry.findall('atom:link', ns):
        if link.get('title') == 'pdf':
            pdf_link = link.get('href')
            break
    # Get published date
    published_elem = entry.find('atom:published', ns)
    published = published_elem.text.strip() if published_elem is not None else ''
    # Authors
    authors = []
    for author in entry.findall('atom:author', ns):
        name_elem = author.find('atom:name', ns)
        if name_elem is not None:
            authors.append(name_elem.text.strip())
    # Categories
    categories = []
    for cat in entry.findall('atom:category', ns):
        term = cat.get('term')
        if term:
            categories.append(term)
    
    # Print markdown
    print(f"### {title}")
    print(f"- **Authors**: {', '.join(authors)}")
    print(f"- **Published**: {published}")
    print(f"- **Categories**: {', '.join(categories)}")
    print(f"- **Link**: {arxiv_id}")
    if pdf_link:
        print(f"- **PDF**: {pdf_link}")
    # Print full summary (truncate only very long ones to 2000 chars)
    if len(summary) > 2000:
        summary = summary[:2000] + '...'
    print(f"- **Summary**: {summary}")
    print()