# Phase 4: Shared UI Layout & Navigation Shell

## Overview

Create the base application layout with responsive sidebar navigation, top bar with user information, HTMX configuration for smooth page transitions, Tailwind theme setup, toast notification system, and breadcrumb component. Establish template inheritance pattern for all future pages.

## Prerequisites

- Phase 1 completed (project scaffolding with HTMX and Tailwind)
- Phase 2 completed (database schema)
- Phase 3 completed (authentication with user context)

## Goals

- Create base template with sidebar navigation matching reference mockup
- Implement responsive sidebar (collapsible on mobile)
- Build top bar with user info and sign-out button
- Configure HTMX for seamless navigation (hx-boost)
- Set up Tailwind config with brand colors
- Create reusable toast notification component
- Implement breadcrumb navigation component
- Establish Go template inheritance system (base, layouts, partials)
- Add loading indicators for HTMX requests
- Create helper templates for common UI elements

## Detailed Implementation Steps

### 1. Create Base Template Structure

1.1. Create base template in `templates/base.html`
- HTML structure with sidebar and main content area
- HTMX and Tailwind CSS includes
- Custom JavaScript includes
- Template blocks for content, title, breadcrumbs

1.2. Create main layout in `templates/layouts/main.html`
- Inherits from base template
- Includes sidebar navigation
- Includes top bar
- Main content wrapper

### 2. Create Sidebar Navigation

2.1. Create sidebar partial in `templates/partials/sidebar.html`
- Logo/application name at top
- Navigation links:
  - Dashboard (/)
  - Projects (/projects)
  - All DCs (/delivery-challans)
  - Serial Search (/serial-search)
- Active link highlighting
- Responsive collapse button (mobile)
- Icons for navigation items (optional, using Heroicons via CDN)

2.2. Add sidebar toggle functionality
- JavaScript for mobile sidebar toggle
- CSS transitions for smooth open/close
- Overlay for mobile sidebar

### 3. Create Top Bar

3.1. Create top bar partial in `templates/partials/topbar.html`
- Breadcrumb navigation
- User info section (name, avatar placeholder)
- Sign out button
- Mobile menu toggle button

### 4. Create Breadcrumb Component

4.1. Create breadcrumb partial in `templates/partials/breadcrumb.html`
- Display hierarchical navigation
- Accept breadcrumb items from context
- Active page highlighting

### 5. Create Toast Notification System

5.1. Create toast container in base template
- Fixed position at top-right
- Stack multiple toasts vertically
- Auto-dismiss after 5 seconds
- Manual dismiss button

5.2. Create toast JavaScript in `static/js/toast.js`
- Show toast function (success, error, info, warning)
- Auto-dismiss timer
- Dismiss button handler
- Queue multiple toasts

5.3. Create toast styles in `static/css/toast.css`
- Toast animations (slide in, fade out)
- Color variants for different types
- Responsive positioning

### 6. Configure HTMX

6.1. Add HTMX configuration in base template
- Enable hx-boost for smooth navigation
- Configure loading indicators
- Set up HTMX event listeners

6.2. Create HTMX loading indicator
- Spinner component
- Progress bar (optional)
- Disable buttons during requests

### 7. Set Up Tailwind Configuration

7.1. Configure Tailwind in base template
- Brand color palette (blue/indigo theme)
- Custom spacing if needed
- Font configuration
- Responsive breakpoints

### 8. Create Template Helper Functions

8.1. Create template helpers in `internal/helpers/template.go`
- Active link checker
- Breadcrumb builder
- User initials generator
- Date/time formatters

### 9. Update Handlers to Use Layout

9.1. Create dashboard handler in `internal/handlers/dashboard.go`
- Use main layout template
- Pass breadcrumb data
- Pass flash messages to toast

9.2. Update main.go to use layout for protected routes

### 10. Add Icons (Optional)

10.1. Include Heroicons via CDN or create custom SVG icons
- Dashboard icon
- Projects icon
- Document icon (DCs)
- Search icon
- User icon
- Sign out icon

## Files to Create/Modify

### `/Users/narendhupati/Documents/ProjectManagementTool/templates/base.html`
```html
<!DOCTYPE html>
<html lang="en" class="h-full bg-gray-50">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ block "title" . }}DC Management Tool{{ end }}</title>

    <!-- Tailwind CSS CDN -->
    <script src="https://cdn.tailwindcss.com"></script>

    <!-- HTMX -->
    <script src="https://unpkg.com/htmx.org@2.0.8"></script>

    <!-- Heroicons (for icons) -->
    <script src="https://cdn.jsdelivr.net/npm/heroicons@2.0.18/outline/index.js"></script>

    <!-- Tailwind Config -->
    <script>
        tailwind.config = {
            theme: {
                extend: {
                    colors: {
                        brand: {
                            50: '#eff6ff',
                            100: '#dbeafe',
                            200: '#bfdbfe',
                            300: '#93c5fd',
                            400: '#60a5fa',
                            500: '#3b82f6',
                            600: '#2563eb',
                            700: '#1d4ed8',
                            800: '#1e40af',
                            900: '#1e3a8a',
                            950: '#172554',
                        }
                    }
                }
            }
        }
    </script>

    <!-- Custom CSS -->
    <link rel="stylesheet" href="/static/css/custom.css">
    <link rel="stylesheet" href="/static/css/toast.css">

    {{ block "head" . }}{{ end }}
</head>
<body class="h-full">
    <!-- Toast Container -->
    <div id="toast-container" class="fixed top-4 right-4 z-50 space-y-2"></div>

    {{ block "body" . }}{{ end }}

    <!-- HTMX Config -->
    <script>
        // HTMX configuration
        document.body.addEventListener('htmx:configRequest', function(evt) {
            // Add loading class to body
            document.body.classList.add('htmx-request');
        });

        document.body.addEventListener('htmx:afterRequest', function(evt) {
            // Remove loading class
            document.body.classList.remove('htmx-request');
        });

        // Show toasts from flash messages
        document.addEventListener('DOMContentLoaded', function() {
            {{ if .flashes }}
                {{ range .flashes }}
                    showToast('{{ .Message }}', '{{ .Type }}');
                {{ end }}
            {{ end }}
        });
    </script>

    <!-- Custom JavaScript -->
    <script src="/static/js/app.js"></script>
    <script src="/static/js/toast.js"></script>

    {{ block "scripts" . }}{{ end }}
</body>
</html>
```

### `/Users/narendhupati/Documents/ProjectManagementTool/templates/layouts/main.html`
```html
{{ template "base.html" . }}

{{ define "body" }}
<div class="h-full flex">
    <!-- Sidebar -->
    {{ template "partials/sidebar.html" . }}

    <!-- Main Content Area -->
    <div class="flex-1 flex flex-col lg:pl-64">
        <!-- Top Bar -->
        {{ template "partials/topbar.html" . }}

        <!-- Page Content -->
        <main class="flex-1 overflow-y-auto bg-gray-50">
            <div class="py-6">
                <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
                    <!-- Breadcrumb -->
                    {{ if .breadcrumbs }}
                        {{ template "partials/breadcrumb.html" . }}
                    {{ end }}

                    <!-- Page Content -->
                    <div class="mt-4">
                        {{ block "content" . }}{{ end }}
                    </div>
                </div>
            </div>
        </main>
    </div>
</div>

<!-- Mobile Sidebar Overlay -->
<div id="sidebar-overlay" class="hidden fixed inset-0 bg-gray-600 bg-opacity-75 z-20 lg:hidden"></div>
{{ end }}

{{ define "scripts" }}
<script src="/static/js/sidebar.js"></script>
{{ end }}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/templates/partials/sidebar.html`
```html
<div id="sidebar" class="fixed inset-y-0 left-0 z-30 w-64 bg-white border-r border-gray-200 transform -translate-x-full lg:translate-x-0 transition-transform duration-200 ease-in-out">
    <div class="h-full flex flex-col">
        <!-- Logo -->
        <div class="flex items-center justify-between h-16 px-6 border-b border-gray-200">
            <div class="flex items-center">
                <h1 class="text-xl font-bold text-brand-600">DC Manager</h1>
            </div>
            <button id="sidebar-close" class="lg:hidden text-gray-500 hover:text-gray-700">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
                </svg>
            </button>
        </div>

        <!-- Navigation -->
        <nav class="flex-1 px-4 py-6 space-y-1 overflow-y-auto">
            <a href="/" class="nav-link {{ if eq .currentPath "/" }}active{{ end }}">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 12l2-2m0 0l7-7 7 7M5 10v10a1 1 0 001 1h3m10-11l2 2m-2-2v10a1 1 0 01-1 1h-3m-6 0a1 1 0 001-1v-4a1 1 0 011-1h2a1 1 0 011 1v4a1 1 0 001 1m-6 0h6"/>
                </svg>
                <span>Dashboard</span>
            </a>

            <a href="/projects" class="nav-link {{ if hasPrefix .currentPath "/projects" }}active{{ end }}">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
                </svg>
                <span>Projects</span>
            </a>

            <a href="/delivery-challans" class="nav-link {{ if hasPrefix .currentPath "/delivery-challans" }}active{{ end }}">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                </svg>
                <span>All DCs</span>
            </a>

            <a href="/serial-search" class="nav-link {{ if hasPrefix .currentPath "/serial-search" }}active{{ end }}">
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"/>
                </svg>
                <span>Serial Search</span>
            </a>
        </nav>

        <!-- Footer -->
        <div class="p-4 border-t border-gray-200">
            <p class="text-xs text-gray-500 text-center">v1.0.0</p>
        </div>
    </div>
</div>
```

### `/Users/narendhupati/Documents/ProjectManagementTool/templates/partials/topbar.html`
```html
<div class="sticky top-0 z-10 bg-white border-b border-gray-200">
    <div class="flex items-center justify-between h-16 px-4 sm:px-6 lg:px-8">
        <!-- Mobile Menu Button -->
        <button id="sidebar-toggle" class="lg:hidden text-gray-500 hover:text-gray-700">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"/>
            </svg>
        </button>

        <!-- Breadcrumb (Desktop) -->
        <div class="hidden lg:block">
            <!-- Breadcrumb will be rendered here by page template -->
        </div>

        <!-- User Menu -->
        <div class="flex items-center space-x-4">
            <!-- User Info -->
            <div class="flex items-center space-x-3">
                <div class="hidden sm:block text-right">
                    <p class="text-sm font-medium text-gray-700">{{ .user.FullName }}</p>
                    <p class="text-xs text-gray-500">{{ .user.Username }}</p>
                </div>
                <div class="h-10 w-10 rounded-full bg-brand-100 flex items-center justify-center">
                    <span class="text-brand-700 font-semibold text-sm">
                        {{ userInitials .user.FullName }}
                    </span>
                </div>
            </div>

            <!-- Sign Out Button -->
            <a href="/logout" class="inline-flex items-center px-3 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 transition-colors">
                <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1"/>
                </svg>
                Sign Out
            </a>
        </div>
    </div>
</div>
```

### `/Users/narendhupati/Documents/ProjectManagementTool/templates/partials/breadcrumb.html`
```html
<nav class="flex" aria-label="Breadcrumb">
    <ol class="flex items-center space-x-2">
        {{ range $index, $item := .breadcrumbs }}
            {{ if $index }}
                <li>
                    <svg class="w-4 h-4 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z" clip-rule="evenodd"/>
                    </svg>
                </li>
            {{ end }}
            <li>
                {{ if .URL }}
                    <a href="{{ .URL }}" class="text-sm font-medium text-gray-500 hover:text-gray-700">
                        {{ .Title }}
                    </a>
                {{ else }}
                    <span class="text-sm font-medium text-gray-900">{{ .Title }}</span>
                {{ end }}
            </li>
        {{ end }}
    </ol>
</nav>
```

### `/Users/narendhupati/Documents/ProjectManagementTool/static/css/toast.css`
```css
/* Toast Notification Styles */

.toast {
    min-width: 300px;
    max-width: 400px;
    padding: 1rem 1.25rem;
    border-radius: 0.5rem;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
    display: flex;
    align-items: center;
    justify-content: space-between;
    animation: slideIn 0.3s ease-out;
}

.toast.removing {
    animation: slideOut 0.3s ease-in;
}

@keyframes slideIn {
    from {
        transform: translateX(100%);
        opacity: 0;
    }
    to {
        transform: translateX(0);
        opacity: 1;
    }
}

@keyframes slideOut {
    from {
        transform: translateX(0);
        opacity: 1;
    }
    to {
        transform: translateX(100%);
        opacity: 0;
    }
}

.toast-success {
    background-color: #10b981;
    color: white;
}

.toast-error {
    background-color: #ef4444;
    color: white;
}

.toast-info {
    background-color: #3b82f6;
    color: white;
}

.toast-warning {
    background-color: #f59e0b;
    color: white;
}

.toast-message {
    flex: 1;
    font-size: 0.875rem;
    font-weight: 500;
}

.toast-close {
    margin-left: 1rem;
    background: none;
    border: none;
    color: inherit;
    cursor: pointer;
    opacity: 0.7;
    transition: opacity 0.2s;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
}

.toast-close:hover {
    opacity: 1;
}

.toast-close svg {
    width: 1.25rem;
    height: 1.25rem;
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/static/js/toast.js`
```javascript
// Toast Notification System

function showToast(message, type = 'info', duration = 5000) {
    const container = document.getElementById('toast-container');
    if (!container) return;

    // Create toast element
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;

    // Toast content
    toast.innerHTML = `
        <div class="toast-message">${escapeHtml(message)}</div>
        <button class="toast-close" aria-label="Close">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
        </button>
    `;

    // Add to container
    container.appendChild(toast);

    // Close button handler
    const closeBtn = toast.querySelector('.toast-close');
    closeBtn.addEventListener('click', () => {
        removeToast(toast);
    });

    // Auto-dismiss
    if (duration > 0) {
        setTimeout(() => {
            removeToast(toast);
        }, duration);
    }
}

function removeToast(toast) {
    toast.classList.add('removing');
    setTimeout(() => {
        if (toast.parentNode) {
            toast.parentNode.removeChild(toast);
        }
    }, 300);
}

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// Expose globally
window.showToast = showToast;
```

### `/Users/narendhupati/Documents/ProjectManagementTool/static/js/sidebar.js`
```javascript
// Sidebar Toggle for Mobile

document.addEventListener('DOMContentLoaded', function() {
    const sidebar = document.getElementById('sidebar');
    const sidebarToggle = document.getElementById('sidebar-toggle');
    const sidebarClose = document.getElementById('sidebar-close');
    const sidebarOverlay = document.getElementById('sidebar-overlay');

    if (!sidebar || !sidebarToggle) return;

    function openSidebar() {
        sidebar.classList.remove('-translate-x-full');
        if (sidebarOverlay) {
            sidebarOverlay.classList.remove('hidden');
        }
    }

    function closeSidebar() {
        sidebar.classList.add('-translate-x-full');
        if (sidebarOverlay) {
            sidebarOverlay.classList.add('hidden');
        }
    }

    sidebarToggle.addEventListener('click', function() {
        if (sidebar.classList.contains('-translate-x-full')) {
            openSidebar();
        } else {
            closeSidebar();
        }
    });

    if (sidebarClose) {
        sidebarClose.addEventListener('click', closeSidebar);
    }

    if (sidebarOverlay) {
        sidebarOverlay.addEventListener('click', closeSidebar);
    }

    // Close sidebar on window resize to desktop
    window.addEventListener('resize', function() {
        if (window.innerWidth >= 1024) {
            closeSidebar();
        }
    });
});
```

### Update `/Users/narendhupati/Documents/ProjectManagementTool/static/css/custom.css`
```css
/* Custom styles for DC Management Tool */

/* Loading indicator for HTMX requests */
.htmx-indicator {
    display: none;
}

.htmx-request .htmx-indicator {
    display: inline-block;
}

.htmx-request.htmx-indicator {
    display: inline-block;
}

/* Loading state for body */
body.htmx-request {
    cursor: wait;
}

/* Smooth transitions */
* {
    transition-property: background-color, border-color, color, fill, stroke;
    transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
    transition-duration: 150ms;
}

/* Navigation Links */
.nav-link {
    @apply flex items-center space-x-3 px-3 py-2 text-sm font-medium text-gray-700 rounded-lg hover:bg-gray-100 hover:text-brand-600 transition-colors;
}

.nav-link.active {
    @apply bg-brand-50 text-brand-600;
}

.nav-link svg {
    @apply flex-shrink-0;
}

/* Print styles */
@media print {
    .no-print {
        display: none !important;
    }

    body {
        background: white;
    }

    #sidebar,
    #sidebar-overlay,
    .sticky {
        display: none !important;
    }
}

/* Custom scrollbar */
::-webkit-scrollbar {
    width: 8px;
    height: 8px;
}

::-webkit-scrollbar-track {
    background: #f1f1f1;
}

::-webkit-scrollbar-thumb {
    background: #888;
    border-radius: 4px;
}

::-webkit-scrollbar-thumb:hover {
    background: #555;
}

/* Card component */
.card {
    @apply bg-white rounded-lg shadow-sm border border-gray-200 p-6;
}

.card-header {
    @apply border-b border-gray-200 pb-4 mb-4;
}

.card-title {
    @apply text-lg font-semibold text-gray-900;
}

/* Button variants */
.btn {
    @apply inline-flex items-center justify-center px-4 py-2 border rounded-lg font-medium focus:outline-none focus:ring-2 focus:ring-offset-2 transition-colors;
}

.btn-primary {
    @apply btn border-transparent bg-brand-600 text-white hover:bg-brand-700 focus:ring-brand-500;
}

.btn-secondary {
    @apply btn border-gray-300 bg-white text-gray-700 hover:bg-gray-50 focus:ring-brand-500;
}

.btn-danger {
    @apply btn border-transparent bg-red-600 text-white hover:bg-red-700 focus:ring-red-500;
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/helpers/template.go`
```go
package helpers

import (
	"html/template"
	"strings"
)

// TemplateFuncs returns a map of template helper functions
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"userInitials": UserInitials,
		"hasPrefix":    strings.HasPrefix,
		"formatDate":   FormatDate,
	}
}

// UserInitials extracts initials from full name
func UserInitials(fullName string) string {
	parts := strings.Fields(fullName)
	if len(parts) == 0 {
		return "?"
	}
	if len(parts) == 1 {
		return strings.ToUpper(string(parts[0][0]))
	}
	return strings.ToUpper(string(parts[0][0]) + string(parts[len(parts)-1][0]))
}

// FormatDate formats a date string
func FormatDate(date string) string {
	// Will be expanded in future phases
	return date
}

// Breadcrumb represents a breadcrumb item
type Breadcrumb struct {
	Title string
	URL   string
}

// BuildBreadcrumbs creates breadcrumb trail
func BuildBreadcrumbs(items ...Breadcrumb) []Breadcrumb {
	return items
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/internal/handlers/dashboard.go`
```go
package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
)

// ShowDashboard displays the dashboard page
func ShowDashboard(c *gin.Context) {
	user := auth.GetCurrentUser(c)

	// Get flash messages
	session, _ := auth.GetStore().Get(c.Request, "dc_management_session")
	flashes := auth.GetFlash(session)
	session.Save(c.Request, c.Writer)

	// Build breadcrumbs
	breadcrumbs := helpers.BuildBreadcrumbs(
		helpers.Breadcrumb{Title: "Dashboard", URL: ""},
	)

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"user":        user,
		"currentPath": c.Request.URL.Path,
		"breadcrumbs": breadcrumbs,
		"flashes":     flashes,
	})
}
```

### `/Users/narendhupati/Documents/ProjectManagementTool/templates/dashboard.html`
```html
{{ template "layouts/main.html" . }}

{{ define "title" }}Dashboard - DC Management Tool{{ end }}

{{ define "content" }}
<div class="space-y-6">
    <!-- Welcome Message -->
    <div class="card">
        <h2 class="text-2xl font-bold text-gray-900 mb-2">
            Welcome back, {{ .user.FullName }}!
        </h2>
        <p class="text-gray-600">
            Manage your delivery challans and projects efficiently.
        </p>
    </div>

    <!-- Stats Cards (Placeholder for Phase 18) -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        <div class="card">
            <div class="flex items-center justify-between">
                <div>
                    <p class="text-sm font-medium text-gray-600">Total Projects</p>
                    <p class="text-3xl font-bold text-gray-900 mt-2">0</p>
                </div>
                <div class="h-12 w-12 bg-brand-100 rounded-lg flex items-center justify-center">
                    <svg class="w-6 h-6 text-brand-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
                    </svg>
                </div>
            </div>
        </div>

        <div class="card">
            <div class="flex items-center justify-between">
                <div>
                    <p class="text-sm font-medium text-gray-600">Total DCs</p>
                    <p class="text-3xl font-bold text-gray-900 mt-2">0</p>
                </div>
                <div class="h-12 w-12 bg-green-100 rounded-lg flex items-center justify-center">
                    <svg class="w-6 h-6 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                    </svg>
                </div>
            </div>
        </div>

        <div class="card">
            <div class="flex items-center justify-between">
                <div>
                    <p class="text-sm font-medium text-gray-600">Issued DCs</p>
                    <p class="text-3xl font-bold text-gray-900 mt-2">0</p>
                </div>
                <div class="h-12 w-12 bg-blue-100 rounded-lg flex items-center justify-center">
                    <svg class="w-6 h-6 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"/>
                    </svg>
                </div>
            </div>
        </div>

        <div class="card">
            <div class="flex items-center justify-between">
                <div>
                    <p class="text-sm font-medium text-gray-600">Draft DCs</p>
                    <p class="text-3xl font-bold text-gray-900 mt-2">0</p>
                </div>
                <div class="h-12 w-12 bg-yellow-100 rounded-lg flex items-center justify-center">
                    <svg class="w-6 h-6 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"/>
                    </svg>
                </div>
            </div>
        </div>
    </div>

    <!-- Quick Actions -->
    <div class="card">
        <h3 class="card-title mb-4">Quick Actions</h3>
        <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            <a href="/projects/new" class="btn btn-primary">
                <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"/>
                </svg>
                Create Project
            </a>
            <a href="/projects" class="btn btn-secondary">
                <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"/>
                </svg>
                View All Projects
            </a>
            <a href="/delivery-challans" class="btn btn-secondary">
                <svg class="w-5 h-5 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
                </svg>
                View All DCs
            </a>
        </div>
    </div>
</div>
{{ end }}
```

### Update `/Users/narendhupati/Documents/ProjectManagementTool/cmd/server/main.go`
```go
package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/narendhupati/dc-management-tool/internal/auth"
	"github.com/narendhupati/dc-management-tool/internal/config"
	"github.com/narendhupati/dc-management-tool/internal/database"
	"github.com/narendhupati/dc-management-tool/internal/handlers"
	"github.com/narendhupati/dc-management-tool/internal/helpers"
	"github.com/narendhupati/dc-management-tool/internal/middleware"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
	db, err := database.Init(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run migrations
	if err := database.RunMigrations(db, "./migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Initialize session store
	auth.InitSessionStore(cfg.SessionSecret)

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.Default()

	// Set template functions
	router.SetFuncMap(helpers.TemplateFuncs())

	// Load templates
	router.LoadHTMLGlob("templates/**/*")

	// Serve static files
	router.Static("/static", "./static")

	// Public routes
	router.GET("/login", handlers.ShowLogin)
	router.POST("/login", handlers.ProcessLogin)
	router.GET("/logout", handlers.Logout)
	router.GET("/health", handlers.HealthCheck)

	// Protected routes
	protected := router.Group("/")
	protected.Use(middleware.RequireAuth())
	{
		protected.GET("/", handlers.ShowDashboard)
	}

	// Start server
	log.Printf("Starting server on %s in %s mode", cfg.ServerAddress, cfg.Environment)
	if err := router.Run(cfg.ServerAddress); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
```

## API Routes / Endpoints

| Method | Path | Handler | Auth Required | Description |
|--------|------|---------|---------------|-------------|
| GET | / | handlers.ShowDashboard | Yes | Dashboard home page |

(Authentication routes from Phase 3 continue to work)

## UI Components

### Sidebar Navigation
- Fixed left sidebar on desktop (64px from left)
- Collapsible on mobile (slide in from left)
- Navigation links with icons
- Active link highlighting
- Application logo at top
- Version number in footer

### Top Bar
- Sticky at top of page
- Mobile menu toggle button (left, mobile only)
- User info section (right)
  - Full name and username
  - Avatar with initials
  - Sign out button
- Breadcrumb navigation (desktop)

### Toast Notifications
- Fixed position at top-right
- Four variants: success, error, info, warning
- Auto-dismiss after 5 seconds
- Manual dismiss button (X)
- Slide-in animation
- Stack multiple toasts vertically

### Breadcrumb
- Horizontal navigation trail
- Arrow separators
- Last item not clickable (current page)
- Responsive text size

### Dashboard Page
- Welcome message card
- Four stat cards (placeholder for Phase 18)
- Quick actions section with buttons

## Testing Checklist

### Manual Testing

- [ ] Login and see dashboard with sidebar navigation
- [ ] Sidebar displays all navigation links (Dashboard, Projects, All DCs, Serial Search)
- [ ] Active link highlighted correctly on dashboard
- [ ] Top bar shows user's full name and initials
- [ ] Sign out button works and redirects to login
- [ ] Mobile: sidebar hidden by default
- [ ] Mobile: toggle button opens sidebar
- [ ] Mobile: overlay appears when sidebar open
- [ ] Mobile: clicking overlay closes sidebar
- [ ] Mobile: close button in sidebar works
- [ ] Desktop: sidebar always visible, toggle button hidden
- [ ] Breadcrumb displays "Dashboard" on home page
- [ ] Flash messages from login appear as toasts
- [ ] Toast auto-dismisses after 5 seconds
- [ ] Toast manual dismiss button works
- [ ] Multiple toasts stack vertically
- [ ] HTMX navigation with hx-boost (when implemented in future routes)
- [ ] Loading indicator during HTMX requests
- [ ] User initials extracted correctly (first + last name)
- [ ] All stat cards show "0" (placeholder data)
- [ ] Quick action buttons styled correctly
- [ ] Responsive layout works on mobile, tablet, desktop

### Cross-Browser Testing

- [ ] Chrome: all features work
- [ ] Firefox: all features work
- [ ] Safari: all features work
- [ ] Edge: all features work

### Accessibility Testing

- [ ] Sidebar navigation keyboard accessible
- [ ] Sign out button keyboard accessible
- [ ] Toast notifications have proper ARIA labels
- [ ] Color contrast meets WCAG AA standards
- [ ] Focus indicators visible on interactive elements

## Acceptance Criteria

- [x] Base template with sidebar and main content area
- [x] Sidebar navigation with all menu items
- [x] Responsive sidebar (collapsible on mobile)
- [x] Top bar with user info and sign out button
- [x] User initials displayed in avatar
- [x] Breadcrumb component
- [x] Toast notification system (4 variants)
- [x] HTMX configuration for page transitions
- [x] Tailwind config with brand colors
- [x] Template helper functions (userInitials, hasPrefix)
- [x] Dashboard page using main layout
- [x] Active link highlighting in navigation
- [x] Mobile sidebar overlay
- [x] Sidebar toggle functionality
- [x] Flash messages converted to toasts
- [x] Custom CSS for nav links, cards, buttons
- [x] Template inheritance system (base, layouts, partials)
- [x] Loading indicators for HTMX requests
- [x] Print-friendly CSS (hide sidebar, topbar)

## Notes

- Using Heroicons via CDN for icons (can be replaced with custom SVGs)
- Tailwind classes applied via CDN config (optimize for production later)
- Toast system uses vanilla JavaScript (no dependencies)
- Sidebar state not persisted across page reloads (can be added later)
- Active link detection uses simple path prefix matching
- Template functions registered in main.go before loading templates
- Glob pattern for templates changed to `templates/**/*` to support subdirectories
- Stats on dashboard are placeholders; will be implemented in Phase 18
- HTMX hx-boost not yet used (will be used in future list/detail pages)

## Future Enhancements

- Dark mode toggle
- User avatar image upload
- Notification badge on nav items
- Keyboard shortcuts
- Sidebar width adjustment (drag to resize)
- Sidebar collapse/expand state persistence
- Advanced toast options (progress bar, actions)
- Dropdown menu for user info (profile, settings)

## Next Steps

After completing Phase 4, proceed to:
- **Phase 5**: Project CRUD - build project management interface with list, create, edit, delete
