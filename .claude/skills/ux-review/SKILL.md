---
name: ux-review
description: Performs a comprehensive UX review of a web application using Nielsen's 10 heuristics with multi-agent evaluation, producing a detailed actionable report.
allowed-tools: Bash, Read, Write, Glob, Grep, Task, Skill
---

# UX Review Skill

You are performing a comprehensive UX review of a web application. Follow these phases exactly.

## Input

The user provides a target URL (e.g., `http://localhost:8080`). If no URL is given, assume `http://localhost:8080`.

Parse the URL from the user's invocation: `$ARGUMENTS`

## Phase 1: Discovery & Navigation

1. Use the `playwright-cli` skill via the Skill tool to navigate the target application.
2. Start at the root URL. Take a snapshot and screenshot of the landing/login page.
3. If there's a login page, ask the user for credentials or check if the app has a default/seed login.
4. Systematically visit every reachable page by following navigation links, sidebar items, and menu entries.
5. For each page visited:
   - Take a screenshot: `playwright-cli screenshot <url> --output screenshots/ux-review/<page-name>.png`
   - Take a snapshot (accessibility tree): `playwright-cli snapshot <url>`
   - Record the page URL, title, and purpose
6. Optionally read project templates (`templates/` directory), CSS, and JS files to understand design intent.

Store all discovery notes in a working document.

## Phase 2: User Journey Mapping

From the pages discovered, identify and document:

1. **Primary user journeys** — the main workflows (e.g., login → dashboard → create item → view list → export)
2. **Navigation paths** — how pages connect, sidebar structure, breadcrumbs
3. **Form flows** — multi-step forms, validation patterns, success/error states
4. **Information architecture** — hierarchy of content, grouping of features

Write a concise journey map summarizing these flows.

## Phase 3: Multi-Perspective Heuristic Evaluation

Launch **5 parallel Task agents** (subagent_type=general-purpose), each evaluating from a distinct expert perspective.

Provide each agent with:
- The screenshots directory path
- The snapshot/accessibility data collected
- The user journey map from Phase 2
- Their specific evaluation focus (see below)
- The heuristics reference at `.claude/skills/ux-review/references/heuristics.md`
- The agent prompts at `.claude/skills/ux-review/references/agent-prompts.md`

### Agent Assignments

Read the detailed prompts from `.claude/skills/ux-review/references/agent-prompts.md` and use them for each agent:

| Agent | Focus Area |
|-------|-----------|
| **Usability Analyst** | Visibility of system status, User control & freedom, Error prevention |
| **Consistency Reviewer** | Consistency & standards, Match between system and real world |
| **Cognitive Load Auditor** | Recognition vs recall, Aesthetic & minimalist design |
| **Accessibility & Efficiency Expert** | Flexibility & efficiency, Help & documentation |
| **Error Recovery Specialist** | Error recognition/diagnosis/recovery, Error prevention |

Launch all 5 agents in parallel using the Task tool. Each agent should return structured findings in this format per issue:

```
### [Issue Title]
- **Severity:** Critical | Major | Minor | Enhancement
- **Heuristic:** [Which heuristic is violated]
- **Location:** [Page/component where the issue occurs]
- **Screenshot:** [Reference to screenshot file if applicable]
- **Description:** [What the issue is]
- **Recommendation:** [How to fix it]
```

## Phase 4: Report Generation

After all agents complete, consolidate findings into a single report.

Read the report template from `.claude/skills/ux-review/references/report-template.md` and fill it in with:

1. **Executive Summary** — Overall UX quality assessment, top 3 strengths, top 3 concerns
2. **Scorecard** — Rate each of Nielsen's 10 heuristics on a 1-5 scale based on agent findings
3. **User Journey Maps** — From Phase 2
4. **Findings by Heuristic** — All issues grouped by heuristic principle, sorted by severity
5. **Prioritized Recommendations** — Top action items ranked by impact and effort
6. **Appendix** — List of all pages visited and screenshots taken

Write the final report to `ux-review-report.md` (or a user-specified path).

## Important Notes

- Be thorough but practical — focus on actionable findings, not theoretical concerns
- Reference specific pages and UI elements in every finding
- Include both problems AND things done well (strengths)
- Severity guide:
  - **Critical**: Blocks users or causes data loss
  - **Major**: Significant usability problem affecting many users
  - **Minor**: Cosmetic or low-frequency issue
  - **Enhancement**: Opportunity to improve, not a current problem
