# Nielsen's 10 Usability Heuristics — Reference Guide

## 1. Visibility of System Status
The system should always keep users informed about what is going on, through appropriate feedback within reasonable time.

**Look for:**
- Loading indicators during async operations
- Progress bars for multi-step processes
- Active state indication on current page/tab/menu item
- Success/error feedback after form submissions
- Timestamps showing data freshness
- Empty states that explain why content is missing

## 2. Match Between System and Real World
The system should speak the users' language, with words, phrases, and concepts familiar to the user, rather than system-oriented terms.

**Look for:**
- Jargon or technical terms users wouldn't understand
- Icons that match real-world metaphors
- Logical ordering of information (chronological, alphabetical, priority)
- Date/number formats matching user locale
- Domain-appropriate terminology

## 3. User Control and Freedom
Users often choose system functions by mistake and need a clearly marked "emergency exit" to leave the unwanted state without going through an extended dialogue.

**Look for:**
- Undo/redo capabilities
- Cancel buttons on forms and dialogs
- Back navigation that works correctly
- Ability to dismiss modals/popups easily
- Confirmation before destructive actions
- Ability to deselect or clear filters

## 4. Consistency and Standards
Users should not have to wonder whether different words, situations, or actions mean the same thing. Follow platform conventions.

**Look for:**
- Consistent button styles, sizes, and placement
- Same terminology for same concepts throughout
- Consistent page layouts and navigation patterns
- Standard web conventions (links look like links, etc.)
- Consistent form validation behavior
- Uniform spacing, typography, and color usage

## 5. Error Prevention
Even better than good error messages is a careful design which prevents a problem from occurring in the first place.

**Look for:**
- Input validation before submission (inline validation)
- Constraints on inputs (date pickers vs free text for dates)
- Confirmation dialogs for irreversible actions
- Disabling submit buttons until forms are valid
- Helpful defaults and autofill
- Clear instructions near complex inputs

## 6. Recognition Rather Than Recall
Minimize the user's memory load by making objects, actions, and options visible.

**Look for:**
- Visible navigation (not hidden behind menus)
- Labels on icons (not icon-only buttons)
- Breadcrumbs showing location in hierarchy
- Recent items / search history
- Contextual help near complex features
- Visible filters showing active criteria

## 7. Flexibility and Efficiency of Use
Accelerators — unseen by the novice user — may often speed up the interaction for the expert user.

**Look for:**
- Keyboard shortcuts for frequent actions
- Search functionality
- Bulk actions (select all, batch delete)
- Customizable views or preferences
- Quick actions / shortcuts from lists
- Efficient form tab order

## 8. Aesthetic and Minimalist Design
Dialogues should not contain information which is irrelevant or rarely needed.

**Look for:**
- Visual clutter or information overload
- Clear visual hierarchy (headings, spacing, grouping)
- Appropriate use of whitespace
- Only essential information shown by default
- Progressive disclosure for advanced options
- Consistent and purposeful use of color

## 9. Help Users Recognize, Diagnose, and Recover from Errors
Error messages should be expressed in plain language (no codes), precisely indicate the problem, and constructively suggest a solution.

**Look for:**
- Error messages in plain language (not HTTP codes or stack traces)
- Error messages that explain what went wrong
- Error messages that suggest how to fix the problem
- Visual indication of which field has the error
- Errors displayed near the source (not just at top of page)
- Graceful handling of edge cases (empty results, timeouts)

## 10. Help and Documentation
Even though it is better if the system can be used without documentation, it may be necessary to provide help and documentation.

**Look for:**
- Tooltips on complex or ambiguous UI elements
- Onboarding for first-time users
- Contextual help (? icons, info text)
- FAQ or help section
- Placeholder text in inputs showing expected format
- Documentation accessible from within the app
