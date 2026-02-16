// Serial Number Real-time Validation
// Validates serial numbers against the project's existing serials via API.

(function() {
    'use strict';

    var DEBOUNCE_MS = 500;
    var validationTimers = {};

    function getProjectID() {
        var el = document.querySelector('input[name="dc_number"]');
        if (!el) return null;
        // Extract from form action URL: /projects/{id}/dcs/...
        var form = el.closest('form');
        if (!form) return null;
        var match = form.action.match(/\/projects\/(\d+)\//);
        return match ? parseInt(match[1]) : null;
    }

    function getCSRFToken() {
        var el = document.querySelector('input[name="gorilla.csrf.Token"]');
        return el ? el.value : '';
    }

    function getExcludeDCID() {
        var el = document.querySelector('input[name="dc_id"]');
        return el ? parseInt(el.value) || null : null;
    }

    function initSerialValidation() {
        document.addEventListener('input', function(e) {
            if (!e.target.classList.contains('serial-textarea')) return;

            var textarea = e.target;
            var card = textarea.closest('.product-card');
            if (!card) return;
            var index = card.getAttribute('data-index');

            // Clear previous timer
            if (validationTimers[index]) {
                clearTimeout(validationTimers[index]);
            }

            // Clear validation state immediately
            clearValidationUI(textarea);

            // Debounce the validation call
            validationTimers[index] = setTimeout(function() {
                validateTextarea(textarea, index);
            }, DEBOUNCE_MS);
        });

        // Prevent form submission with validation errors
        document.addEventListener('submit', function(e) {
            var form = e.target;
            if (form.querySelector('.serial-textarea.border-red-500')) {
                e.preventDefault();
                // Show toast if available
                if (typeof showToast === 'function') {
                    showToast('Please fix serial number validation errors before saving.', 'error');
                }
                return false;
            }
        });

        // Barcode scanner: convert Enter to newline in serial textareas
        document.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' && e.target.classList.contains('serial-textarea')) {
                // Allow default Enter behavior (adds newline in textarea)
                // But trigger input event for validation
                setTimeout(function() {
                    e.target.dispatchEvent(new Event('input', { bubbles: true }));
                }, 10);
            }
        });
    }

    function validateTextarea(textarea, index) {
        var projectID = getProjectID();
        if (!projectID) return;

        var serialNumbers = textarea.value.trim();
        if (!serialNumbers) {
            clearValidationUI(textarea);
            return;
        }

        var body = {
            project_id: projectID,
            serial_numbers: serialNumbers,
            exclude_dc_id: getExcludeDCID()
        };

        fetch('/api/serial-numbers/validate', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': getCSRFToken()
            },
            body: JSON.stringify(body)
        })
        .then(function(resp) { return resp.json(); })
        .then(function(result) {
            if (!result.valid) {
                showValidationErrors(textarea, result);
            } else {
                showValidationSuccess(textarea);
            }
        })
        .catch(function(err) {
            console.error('Serial validation error:', err);
        });
    }

    function showValidationErrors(textarea, result) {
        textarea.classList.add('border-red-500', 'bg-red-50');
        textarea.classList.remove('border-green-500', 'bg-green-50', 'border-gray-200', 'bg-gray-50');

        var container = getOrCreateErrorContainer(textarea);
        var html = '';

        if (result.duplicate_in_input && result.duplicate_in_input.length > 0) {
            html += '<div class="text-red-600 text-xs mb-1">' +
                '<strong>Duplicates in this list:</strong> ' +
                escapeHtml(result.duplicate_in_input.join(', ')) +
                '</div>';
        }

        if (result.duplicate_in_db && result.duplicate_in_db.length > 0) {
            html += '<div class="text-red-600 text-xs">' +
                '<strong>Already used in other DCs:</strong>' +
                '<ul class="list-disc list-inside mt-1">';
            result.duplicate_in_db.forEach(function(c) {
                html += '<li>' + escapeHtml(c.serial_number) +
                    ' &mdash; ' + escapeHtml(c.dc_number) +
                    ' (' + escapeHtml(c.dc_status) + ')</li>';
            });
            html += '</ul></div>';
        }

        container.innerHTML = html;
        container.classList.remove('hidden');
    }

    function showValidationSuccess(textarea) {
        textarea.classList.remove('border-red-500', 'bg-red-50', 'border-gray-200', 'bg-gray-50');
        textarea.classList.add('border-green-500', 'bg-green-50');

        var container = getOrCreateErrorContainer(textarea);
        container.innerHTML = '';
        container.classList.add('hidden');
    }

    function clearValidationUI(textarea) {
        textarea.classList.remove('border-red-500', 'bg-red-50', 'border-green-500', 'bg-green-50');
        textarea.classList.add('border-gray-200', 'bg-gray-50');

        var container = textarea.parentElement.querySelector('.serial-validation-errors');
        if (container) {
            container.innerHTML = '';
            container.classList.add('hidden');
        }
    }

    function getOrCreateErrorContainer(textarea) {
        var parent = textarea.parentElement;
        var container = parent.querySelector('.serial-validation-errors');
        if (!container) {
            container = document.createElement('div');
            container.className = 'serial-validation-errors mt-2 hidden';
            parent.appendChild(container);
        }
        return container;
    }

    function escapeHtml(str) {
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(str));
        return div.innerHTML;
    }

    // Initialize on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initSerialValidation);
    } else {
        initSerialValidation();
    }
})();
