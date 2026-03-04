---
name: ui-review
description: Performs a comprehensive UI design review of a web application using Playwright for real browser inspection — capturing screenshots at multiple viewports, extracting CSS/computed styles, and running multi-agent visual design evaluation to produce an actionable improvement report.
allowed-tools: Bash(playwright-cli:*), Read, Write, Glob, Grep, Task, Skill
---

# UI Design Review Skill

You are performing a comprehensive UI design review of a web application using Playwright for real browser inspection. Follow these phases exactly.

## Input

The user provides a target URL and optionally login credentials.

Parse from the invocation: `$ARGUMENTS`

- If no URL is given, assume `http://localhost:8080`
- If credentials are needed and not provided, ask the user before proceeding
- Optional: `--output <path>` to set report output file (default: `ui-review-report.md`)

## Phase 1: Browser Setup & Authentication

1. Open a named Playwright session:
   ```bash
   playwright-cli -s=ui-review open <URL> --browser=chrome
   ```
2. Take an initial screenshot:
   ```bash
   playwright-cli -s=ui-review screenshot --filename=screenshots/ui-review/00-landing.png
   playwright-cli -s=ui-review snapshot --filename=snapshots/ui-review/00-landing.yaml
   ```
3. If a login form is present, authenticate using provided credentials:
   ```bash
   playwright-cli -s=ui-review fill <ref> "username"
   playwright-cli -s=ui-review fill <ref> "password"
   playwright-cli -s=ui-review click <submit-ref>
   ```
4. Save authenticated state for reuse:
   ```bash
   playwright-cli -s=ui-review state-save screenshots/ui-review/auth.json
   ```
5. Check for JS console errors that may affect rendering:
   ```bash
   playwright-cli -s=ui-review console error
   ```

## Phase 2: Full-Application Discovery with Screenshots

Systematically visit every reachable page. For each page:

1. Navigate to the page
2. Take a **desktop screenshot** (default viewport):
   ```bash
   playwright-cli -s=ui-review screenshot --filename=screenshots/ui-review/<page-slug>-desktop.png
   ```
3. Resize to tablet and take screenshot:
   ```bash
   playwright-cli -s=ui-review resize 768 1024
   playwright-cli -s=ui-review screenshot --filename=screenshots/ui-review/<page-slug>-tablet.png
   playwright-cli -s=ui-review resize 1280 800
   ```
4. Resize to mobile and take screenshot:
   ```bash
   playwright-cli -s=ui-review resize 390 844
   playwright-cli -s=ui-review screenshot --filename=screenshots/ui-review/<page-slug>-mobile.png
   playwright-cli -s=ui-review resize 1280 800
   ```
5. Capture the accessibility snapshot:
   ```bash
   playwright-cli -s=ui-review snapshot --filename=snapshots/ui-review/<page-slug>.yaml
   ```
6. Extract key visual data using `eval`:
   ```bash
   # Extract all font families in use
   playwright-cli -s=ui-review eval "Array.from(new Set(Array.from(document.querySelectorAll('*')).map(el => getComputedStyle(el).fontFamily))).slice(0, 20).join(' | ')"

   # Extract all background colors in use
   playwright-cli -s=ui-review eval "Array.from(new Set(Array.from(document.querySelectorAll('*')).map(el => getComputedStyle(el).backgroundColor).filter(c => c !== 'rgba(0, 0, 0, 0)'))).slice(0, 20).join(' | ')"

   # Check button styles
   playwright-cli -s=ui-review eval "Array.from(document.querySelectorAll('button, [type=submit], .btn')).slice(0,5).map(el => ({ text: el.textContent.trim().slice(0,30), bg: getComputedStyle(el).backgroundColor, color: getComputedStyle(el).color, radius: getComputedStyle(el).borderRadius, padding: getComputedStyle(el).padding })).map(b => JSON.stringify(b)).join('\n')"

   # Check heading hierarchy
   playwright-cli -s=ui-review eval "Array.from(document.querySelectorAll('h1,h2,h3,h4,h5,h6')).map(el => el.tagName + ': ' + el.textContent.trim().slice(0,50) + ' | ' + getComputedStyle(el).fontSize).join('\n')"

   # Check focus-visible styles (keyboard accessibility)
   playwright-cli -s=ui-review eval "getComputedStyle(document.querySelector(':focus-visible') || document.body, ':focus-visible').outline"

   # Extract spacing/padding patterns on main containers
   playwright-cli -s=ui-review eval "Array.from(document.querySelectorAll('main, .container, .content, [class*=page], [class*=wrapper]')).slice(0,5).map(el => el.className.slice(0,40) + ' | padding: ' + getComputedStyle(el).padding).join('\n')"
   ```
7. Record page metadata: URL, title, purpose, nav path

## Phase 3: Targeted UI Component Inspection

After discovery, perform deep inspection of key UI patterns found across the app:

```bash
# Color contrast check — inspect text on backgrounds
playwright-cli -s=ui-review eval "Array.from(document.querySelectorAll('p, span, a, td, th, label, button')).slice(0,10).map(el => ({ tag: el.tagName, text: el.textContent.trim().slice(0,20), color: getComputedStyle(el).color, bg: getComputedStyle(el).backgroundColor, size: getComputedStyle(el).fontSize })).map(x => JSON.stringify(x)).join('\n')"

# Check for icon-only buttons (missing accessible labels)
playwright-cli -s=ui-review eval "Array.from(document.querySelectorAll('button, a[role=button]')).filter(el => !el.textContent.trim() && !el.getAttribute('aria-label') && !el.getAttribute('title')).map(el => el.outerHTML.slice(0,100)).join('\n')"

# Detect inconsistent border-radius values
playwright-cli -s=ui-review eval "Array.from(new Set(Array.from(document.querySelectorAll('button, input, .card, [class*=card], [class*=modal]')).map(el => getComputedStyle(el).borderRadius))).join(' | ')"

# Check input/form element styling consistency
playwright-cli -s=ui-review eval "Array.from(document.querySelectorAll('input, select, textarea')).slice(0,8).map(el => ({ type: el.type || el.tagName, border: getComputedStyle(el).border, radius: getComputedStyle(el).borderRadius, padding: getComputedStyle(el).padding, fontSize: getComputedStyle(el).fontSize })).map(x => JSON.stringify(x)).join('\n')"

# Check for z-index stacking issues / overlapping elements
playwright-cli -s=ui-review eval "Array.from(document.querySelectorAll('*')).filter(el => parseInt(getComputedStyle(el).zIndex) > 0).map(el => el.tagName + '.' + el.className.slice(0,30) + ' z-index:' + getComputedStyle(el).zIndex).join('\n')"

# Table styling consistency
playwright-cli -s=ui-review eval "Array.from(document.querySelectorAll('table, [class*=table]')).slice(0,3).map(el => ({ classes: el.className.slice(0,50), cellPadding: getComputedStyle(el.querySelector('td') || el).padding || 'n/a' })).map(x => JSON.stringify(x)).join('\n')"
```

Take targeted screenshots of specific components:
```bash
# Screenshot specific elements by ref
playwright-cli -s=ui-review screenshot <nav-ref> --filename=screenshots/ui-review/component-navigation.png
playwright-cli -s=ui-review screenshot <form-ref> --filename=screenshots/ui-review/component-form.png
playwright-cli -s=ui-review screenshot <table-ref> --filename=screenshots/ui-review/component-table.png
```

Hover over interactive elements to capture hover states:
```bash
playwright-cli -s=ui-review hover <button-ref>
playwright-cli -s=ui-review screenshot --filename=screenshots/ui-review/state-button-hover.png
```

## Phase 4: Multi-Agent Visual Design Evaluation

Launch **5 parallel Task agents** (subagent_type=general-purpose), each specializing in a distinct visual design domain.

Provide each agent with:
- Path to all screenshots: `screenshots/ui-review/`
- Path to all snapshots: `snapshots/ui-review/`
- All CSS/style data extracted via `eval` in Phase 3
- The full list of pages discovered and their URLs
- The design principles reference: `.claude/skills/ui-review/references/design-principles.md`
- Their specific agent prompt from: `.claude/skills/ui-review/references/agent-prompts.md`

### Agent Assignments

Read the detailed prompts from `.claude/skills/ui-review/references/agent-prompts.md` and use them for each agent:

| Agent | Focus Area |
|-------|-----------|
| **Typography & Visual Hierarchy Analyst** | Font system, heading hierarchy, readability, text contrast, visual flow |
| **Color & Branding Auditor** | Color palette, contrast ratios, semantic color usage, brand consistency |
| **Spacing, Layout & Grid Reviewer** | Spacing system, alignment, whitespace, grid consistency, density |
| **Component & Interaction Inspector** | Button/form/table styles, hover states, focus styles, component consistency across pages |
| **Responsive Design & Motion Reviewer** | Mobile/tablet/desktop layouts, breakpoints, animation quality, loading states |

Launch all 5 agents in **parallel** using the Task tool. Each agent must return structured findings:

```
### [Issue Title]
- **Severity:** Critical | Major | Minor | Enhancement
- **Category:** [Design principle violated]
- **Location:** [Page/component — be specific]
- **Screenshot:** [Filename if applicable]
- **Current State:** [What it looks like now / CSS values observed]
- **Problem:** [Why this is a UI problem]
- **Recommendation:** [Specific fix with example CSS/HTML if possible]
```

## Phase 5: Report Generation

After all agents complete, consolidate findings into a single report.

Read the report template from `.claude/skills/ui-review/references/report-template.md` and populate it with:

1. **Executive Summary** — Overall UI quality, design maturity, top 3 strengths, top 3 critical issues
2. **Design Scorecard** — Rate each of the 10 design dimensions on a 1-5 scale
3. **Color & Typography Audit** — Extracted palettes, font stacks, contrast issues
4. **Findings by Category** — All issues grouped by design dimension, sorted by severity
5. **Prioritized Action Plan** — Ranked recommendations with effort/impact estimates
6. **Component Improvement Sketches** — Specific before/after CSS suggestions for top issues
7. **Appendix** — All pages visited, all screenshots taken, responsive notes

Write the final report to `ui-review-report.md` (or user-specified path).

Close the browser session:
```bash
playwright-cli -s=ui-review close
```

## Important Notes

- **Be specific and actionable**: Every finding must reference exact page, component, and—where possible—the exact CSS property to change
- **Include extracted data**: Quote the actual CSS values you found (font sizes, colors, spacing) — don't generalize
- **Screenshot everything**: If you flag a visual issue, there should be a screenshot proving it
- **Mobile-first lens**: Note when desktop-only design neglects mobile UX
- **Distinguish cosmetic from structural**: A wrong border-radius is Minor; missing focus styles are Major
- **Severity guide:**
  - **Critical**: Accessibility failure, unreadable text, broken layout
  - **Major**: Inconsistent component system, poor contrast, confusing hierarchy
  - **Minor**: Cosmetic inconsistency, spacing irregularity
  - **Enhancement**: Polish opportunity, not a current problem
