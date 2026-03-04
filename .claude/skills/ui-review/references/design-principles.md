# UI Design Principles — Reference Guide

## 1. Visual Hierarchy

The layout should guide the user's eye through content in a logical sequence — from most important to least important.

**Look for:**
- Clear distinction between primary, secondary, and tertiary headings (size, weight, color)
- Content that draws attention without a clear purpose (multiple competing focal points)
- Important actions buried in visual noise
- Proper use of size, weight, color, and whitespace to establish priority
- Above-the-fold content that communicates the page's purpose immediately
- Lists and tables that lack clear row hierarchy

**CSS signals to check:**
- `font-size` and `font-weight` progression across `h1`–`h6`
- `color` contrast between heading and body text
- Element sizes and visual weight relative to their importance

---

## 2. Typography System

A consistent, readable type system reduces cognitive load and builds visual trust.

**Look for:**
- More than 3 font families in use (indicates lack of system)
- Inconsistent font sizes for the same semantic role (e.g., page titles varying by 4px+)
- Line height below 1.4 for body text (readability issue)
- Line length (measure) exceeding 80 characters (reduces readability)
- Missing or inconsistent font weight scale (using only 400 and 700)
- Text set entirely in uppercase without letter-spacing
- All-caps body text
- Inconsistent heading scales across pages

**CSS signals to check:**
- `font-family` values in use
- `font-size` values (should follow a modular scale: 12, 14, 16, 18, 20, 24, 28, 32...)
- `line-height` (1.4–1.7 for body, 1.1–1.3 for headings)
- `letter-spacing` (should be tight for headings, default for body)
- `font-weight` values in use

---

## 3. Color System

Colors should be intentional, consistent, and semantically meaningful.

**Look for:**
- More than 5–7 unique colors used (indicates no coherent palette)
- Primary action color used inconsistently across buttons, links, highlights
- Semantic colors violated (red used for non-error purposes; green for non-success)
- Low contrast text (WCAG AA requires 4.5:1 for normal text, 3:1 for large text)
- Background colors that vary without purpose (random grays/whites)
- Interactive elements (links, buttons) not visually distinct from static content
- Hover/active states that don't provide sufficient feedback

**WCAG contrast thresholds:**
- Normal text (< 18pt): minimum 4.5:1
- Large text (≥ 18pt or 14pt bold): minimum 3:1
- UI components and graphics: minimum 3:1
- AAA standard: 7:1 for normal text

---

## 4. Spacing & Layout System

A consistent spacing system creates rhythm, alignment, and visual order.

**Look for:**
- Arbitrary spacing values (e.g., 13px, 17px, 23px instead of multiples of 4 or 8)
- Inconsistent padding inside similar components (buttons, cards, form fields)
- Elements that lack breathing room (cramped layouts)
- Excessive whitespace that wastes screen real estate
- Misaligned elements within a grid or list
- Container widths that are inconsistent across similar pages
- Forms where labels and fields are misaligned

**Common spacing scales:**
- 4px base: 4, 8, 12, 16, 24, 32, 48, 64
- 8px base: 8, 16, 24, 32, 48, 64, 96

---

## 5. Component Consistency

UI components should look and behave the same across all pages.

**Look for:**
- Buttons with different `border-radius` values on different pages
- Form inputs styled differently across forms
- Multiple card styles that don't follow a pattern
- Tables with different cell padding, border styles, or hover colors
- Modal dialogs with inconsistent header/footer styles
- Navigation items with different active state treatments
- Badges, tags, or chips styled differently in different contexts
- Alert/notification components with inconsistent patterns

---

## 6. Responsive Design

The UI should degrade gracefully and remain usable across all screen sizes.

**Look for:**
- Horizontal scrolling on mobile (content exceeds viewport width)
- Navigation that breaks or becomes unusable on tablet/mobile
- Touch targets smaller than 44×44px on mobile
- Text too small to read on mobile (below 14px)
- Tables that overflow or lack responsive adaptations
- Forms that are hard to use on mobile (fields too small, poor tap targets)
- Images/charts that don't scale properly
- Fixed-width elements that cause layout issues
- Modals that can't be scrolled on small screens

**Viewport sizes to test:**
- Mobile: 390×844 (iPhone 14)
- Tablet: 768×1024 (iPad)
- Desktop: 1280×800 (standard laptop)
- Wide: 1920×1080 (large monitor)

---

## 7. Interactive States & Micro-interactions

Every interactive element needs clear state communication.

**Look for:**
- Buttons/links with no visible hover state
- Focus styles missing or overridden (`:focus-visible` should be visible)
- No visual feedback during loading (spinners, skeletons, progress bars)
- Form fields with no focus indicator
- Disabled states that look the same as enabled states
- No loading state on async actions (user clicks button, nothing appears to happen)
- Transitions/animations that are abrupt (no easing, zero duration)
- Animations that are too slow or distracting

---

## 8. Iconography & Visual Assets

Icons and images should enhance understanding, not create ambiguity.

**Look for:**
- Icon-only buttons with no accessible label (`aria-label` or tooltip)
- Mixed icon styles (some filled, some outlined, different sizes)
- Icons that don't match their intended meaning
- Low-quality or blurry images
- Images without `alt` text
- Icons too small to tap on mobile (below 24×24px)
- Inconsistent icon sizing across the interface
- Icons used where text would be clearer

---

## 9. Empty States & Edge Cases

Well-designed empty states guide users toward action.

**Look for:**
- Empty lists/tables with no explanation or call to action
- "No results" messages that don't suggest what to do next
- Loading states that are absent or jarring
- Error states with generic messages ("Something went wrong")
- 404 pages that don't help users navigate back
- Form fields with no placeholder or helper text
- Long loading times with no feedback

---

## 10. Density & Information Architecture

Information density should match user context and task complexity.

**Look for:**
- Dashboards showing too much data without prioritization
- Data tables with 15+ columns showing simultaneously
- Forms with too many fields on a single step
- Navigation with more than 7–8 top-level items
- Nested navigation more than 3 levels deep
- Information that requires scrolling far to reach on simple tasks
- Features buried in menus that users frequently need
- Long forms that aren't broken into logical sections
