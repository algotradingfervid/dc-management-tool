# UI Design Review Report — [Application Name]

**Date:** [Date]
**URL:** [Target URL]
**Reviewer:** Claude Code UI Review Skill
**Viewports Tested:** Desktop (1280×800), Tablet (768×1024), Mobile (390×844)

---

## Executive Summary

[2–3 paragraph overview of UI design quality, design maturity, and overall visual coherence]

### Top 3 Strengths
1. [Strength — be specific]
2. [Strength — be specific]
3. [Strength — be specific]

### Top 3 Critical Issues
1. [Issue — Severity — which pages affected]
2. [Issue — Severity — which pages affected]
3. [Issue — Severity — which pages affected]

---

## Design Scorecard

| # | Design Dimension | Score (1–5) | Notes |
|---|-----------------|:-----------:|-------|
| 1 | Visual Hierarchy | /5 | |
| 2 | Typography System | /5 | |
| 3 | Color System | /5 | |
| 4 | Spacing & Layout | /5 | |
| 5 | Component Consistency | /5 | |
| 6 | Responsive Design | /5 | |
| 7 | Interactive States | /5 | |
| 8 | Iconography & Assets | /5 | |
| 9 | Empty States & Edge Cases | /5 | |
| 10 | Information Density | /5 | |
| | **Overall** | **/5** | |

*Scale: 1 = Severe/broken, 2 = Major issues, 3 = Adequate, 4 = Good, 5 = Excellent*

---

## Color & Typography Audit

### Color Palette (Extracted)

| Role | Value | Used On | Notes |
|------|-------|---------|-------|
| Primary | | | |
| Secondary | | | |
| Background | | | |
| Surface | | | |
| Text (primary) | | | |
| Text (secondary) | | | |
| Success | | | |
| Error | | | |
| Warning | | | |

**Total unique colors detected:** [N]
**Assessment:** [Too many / Coherent system / Needs consolidation]

### Contrast Issues

| Text Color | Background | Estimated Ratio | WCAG AA Pass? | Location |
|------------|------------|:--------------:|:-------------:|----------|
| | | | | |

### Typography Stack (Extracted)

| Font Family | Used For | Weight(s) Used | Notes |
|-------------|----------|---------------|-------|
| | | | |

**Total font families detected:** [N]
**Font size range:** [min]px – [max]px
**Scale assessment:** [Modular / Arbitrary / Partially consistent]

---

## Findings by Category

### 1. Visual Hierarchy

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

### 2. Typography System

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

### 3. Color System

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

### 4. Spacing & Layout

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

### 5. Component Consistency

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

### 6. Responsive Design

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

### 7. Interactive States

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

### 8. Iconography & Assets

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

### 9. Empty States & Edge Cases

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

### 10. Information Density

#### Critical
[Findings or "No critical issues found."]

#### Major
[Findings]

#### Minor
[Findings]

#### Enhancements
[Findings]

---

## Prioritized Action Plan

### High Priority — Address Immediately
*Critical issues that harm usability, accessibility, or brand trust*

| # | Recommendation | Category | Effort (S/M/L) | Impact |
|---|---------------|----------|:--------------:|--------|
| 1 | | | | |
| 2 | | | | |
| 3 | | | | |

### Medium Priority — Next Sprint
*Major issues that degrade the experience but aren't blocking*

| # | Recommendation | Category | Effort (S/M/L) | Impact |
|---|---------------|----------|:--------------:|--------|
| 1 | | | | |
| 2 | | | | |

### Low Priority — Backlog
*Minor polish and enhancements*

| # | Recommendation | Category | Effort (S/M/L) | Impact |
|---|---------------|----------|:--------------:|--------|
| 1 | | | | |
| 2 | | | | |

---

## Component Improvement Suggestions

### Buttons
**Current state:** [Description with CSS values]
```css
/* Current */
.btn-primary {
  /* extracted values */
}
```
**Suggested improvement:**
```css
/* Recommended */
.btn-primary {
  /* improved values */
}
```

### Form Inputs
**Current state:** [Description]
**Suggested improvement:** [Specific CSS changes]

### [Other key component]
**Current state:** [Description]
**Suggested improvement:** [Specific CSS changes]

---

## Responsive Design Notes

### Mobile (390×844)
[Summary of mobile-specific issues and observations]

### Tablet (768×1024)
[Summary of tablet-specific issues and observations]

### Desktop (1280×800)
[Summary of desktop observations]

---

## Appendix

### Pages Visited
| # | Page | URL | Desktop Screenshot | Mobile Screenshot |
|---|------|-----|--------------------|-------------------|
| 1 | | | | |

### Screenshots Taken
[List of all screenshot files with descriptions]

### CSS Data Extracted
[Summarized raw CSS extraction output from Phase 3]

### Methodology
This review was conducted using:
- **Playwright** for real browser automation, multi-viewport screenshots, and CSS extraction via `eval`
- **5 specialist agents** evaluating: Typography & Visual Hierarchy, Color & Branding, Spacing & Layout, Component Consistency, and Responsive Design
- **10 UI design dimensions** (see `references/design-principles.md`)
- **Viewports:** Desktop (1280×800), Tablet (768×1024), Mobile (390×844)
