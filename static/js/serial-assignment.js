/**
 * Serial Number Entry & Assignment for Shipment Wizard Step 3
 */

(function() {
    'use strict';

    // State per product: { productId: { serials: [], assignments: { addrId: [serial,...] } } }
    var state = {};

    // Initialize a product's state
    function initProduct(productId, required, destinations) {
        state[productId] = {
            required: required,
            destinations: destinations, // [{id, name, qty}]
            serials: [],
            assignments: {} // addrId -> [serial]
        };
        // Initialize empty assignments for each destination
        destinations.forEach(function(d) {
            state[productId].assignments[d.id] = [];
        });
    }

    /**
     * Parse textarea into array of serial numbers, trim whitespace, remove blanks.
     */
    function parseSerials(textarea) {
        var lines = textarea.value.split('\n');
        var serials = [];
        for (var i = 0; i < lines.length; i++) {
            var s = lines[i].trim();
            if (s !== '') {
                serials.push(s);
            }
        }
        return serials;
    }

    /**
     * Validate serials against the server for duplicates in the project.
     */
    function validateSerials(productId, serials, projectId, csrfToken) {
        if (serials.length === 0) return;

        var errDiv = document.getElementById('serial-errors-' + productId);

        fetch('/api/serial-numbers/validate', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': csrfToken
            },
            body: JSON.stringify({
                project_id: projectId,
                product_id: productId,
                serial_numbers: serials.join('\n')
            })
        })
        .then(function(resp) { return resp.json(); })
        .then(function(data) {
            var html = '';
            if (data.duplicate_in_db && data.duplicate_in_db.length > 0) {
                html += '<p class="text-sm text-red-600 mt-1">Already used in project: ';
                html += data.duplicate_in_db.map(function(d) {
                    return d.serial_number + ' (DC: ' + d.dc_number + ')';
                }).join(', ');
                html += '</p>';
            }
            // Append to existing errors (don't overwrite duplicate-in-input errors)
            var existing = errDiv.querySelector('.db-errors');
            if (existing) existing.remove();
            if (html) {
                var div = document.createElement('div');
                div.className = 'db-errors';
                div.innerHTML = html;
                errDiv.appendChild(div);
            }
        })
        .catch(function(err) {
            console.error('Serial validation error:', err);
        });
    }

    /**
     * Render the assignment grid for a product's serials.
     */
    function renderAssignmentGrid(productId, serials, destinations, qtyPerSet) {
        var container = document.getElementById('assignment-' + productId);
        if (!container) return;

        if (serials.length === 0) {
            container.classList.add('hidden');
            container.innerHTML = '';
            return;
        }

        container.classList.remove('hidden');

        var html = '<div class="border-t border-gray-200 pt-4">';
        html += '<div class="flex items-center justify-between mb-3">';
        html += '<h4 class="text-sm font-medium text-gray-700">Serial Assignment</h4>';
        html += '<div class="flex gap-2">';
        html += '<button type="button" class="btn btn-secondary text-xs" onclick="SerialAssignment.autoAssign(' + productId + ')">Auto-Assign All</button>';
        html += '<button type="button" class="btn btn-secondary text-xs" onclick="SerialAssignment.clearAssignments(' + productId + ')">Clear Assignments</button>';
        html += '</div></div>';

        // Table
        html += '<div class="overflow-x-auto"><table class="min-w-full divide-y divide-gray-200 text-sm">';

        // Header
        html += '<thead class="bg-gray-50"><tr>';
        html += '<th class="px-3 py-2 text-left font-medium text-gray-500">Serial</th>';
        destinations.forEach(function(d) {
            html += '<th class="px-3 py-2 text-center font-medium text-gray-500">' + escapeHtml(d.name) + '</th>';
        });
        html += '<th class="px-3 py-2 text-center font-medium text-gray-500">Unassigned</th>';
        html += '</tr></thead>';

        // Body
        html += '<tbody class="divide-y divide-gray-100">';
        serials.forEach(function(serial, idx) {
            var rowClass = idx % 2 === 0 ? 'bg-white' : 'bg-gray-50';
            html += '<tr class="' + rowClass + '">';
            html += '<td class="px-3 py-2 font-mono text-xs">' + escapeHtml(serial) + '</td>';

            // Find current assignment
            var currentAssignment = getCurrentAssignment(productId, serial);

            destinations.forEach(function(d) {
                var checked = currentAssignment === String(d.id) ? ' checked' : '';
                html += '<td class="px-3 py-2 text-center">';
                html += '<input type="radio" name="assign_serial_' + productId + '_' + idx + '" ';
                html += 'value="' + d.id + '" data-product="' + productId + '" data-serial="' + escapeAttr(serial) + '"';
                html += ' onchange="SerialAssignment.onAssignmentChange(' + productId + ')"' + checked + '>';
                html += '</td>';
            });

            // Unassigned column
            var unChecked = currentAssignment === 'unassigned' || currentAssignment === '' ? ' checked' : '';
            html += '<td class="px-3 py-2 text-center">';
            html += '<input type="radio" name="assign_serial_' + productId + '_' + idx + '" ';
            html += 'value="unassigned" data-product="' + productId + '" data-serial="' + escapeAttr(serial) + '"';
            html += ' onchange="SerialAssignment.onAssignmentChange(' + productId + ')"' + unChecked + '>';
            html += '</td>';

            html += '</tr>';
        });
        html += '</tbody>';

        // Footer with counters
        html += '<tfoot class="bg-gray-100"><tr>';
        html += '<td class="px-3 py-2 font-medium text-gray-700">Assigned</td>';
        destinations.forEach(function(d) {
            var count = state[productId] && state[productId].assignments[d.id] ? state[productId].assignments[d.id].length : 0;
            var cls = count === qtyPerSet ? 'text-green-600' : (count > qtyPerSet ? 'text-red-600' : 'text-gray-600');
            html += '<td class="px-3 py-2 text-center font-medium ' + cls + '" id="counter-' + productId + '-' + d.id + '">';
            html += count + '/' + qtyPerSet;
            html += '</td>';
        });
        // Unassigned count
        var totalAssigned = 0;
        destinations.forEach(function(d) {
            totalAssigned += state[productId] && state[productId].assignments[d.id] ? state[productId].assignments[d.id].length : 0;
        });
        var unassignedCount = serials.length - totalAssigned;
        html += '<td class="px-3 py-2 text-center font-medium text-gray-500" id="counter-' + productId + '-unassigned">' + unassignedCount + '</td>';
        html += '</tr></tfoot>';

        html += '</table></div></div>';
        container.innerHTML = html;
    }

    function getCurrentAssignment(productId, serial) {
        if (!state[productId]) return '';
        var assignments = state[productId].assignments;
        for (var addrId in assignments) {
            if (assignments[addrId].indexOf(serial) !== -1) {
                return String(addrId);
            }
        }
        return 'unassigned';
    }

    /**
     * Auto-assign serials sequentially: first qtyPerSet → first destination, etc.
     */
    function autoAssign(productId) {
        var ps = state[productId];
        if (!ps) return;

        // Clear current assignments
        ps.destinations.forEach(function(d) {
            ps.assignments[d.id] = [];
        });

        var serials = ps.serials;
        var idx = 0;
        ps.destinations.forEach(function(d) {
            var qty = d.qty;
            for (var i = 0; i < qty && idx < serials.length; i++) {
                ps.assignments[d.id].push(serials[idx]);
                idx++;
            }
        });

        // Re-render grid and sync
        renderAssignmentGrid(productId, ps.serials, ps.destinations, ps.destinations.length > 0 ? ps.destinations[0].qty : 0);
        syncHiddenFields(productId);
    }

    /**
     * Clear all assignments for a product (all set to unassigned).
     */
    function clearAssignments(productId) {
        var ps = state[productId];
        if (!ps) return;

        ps.destinations.forEach(function(d) {
            ps.assignments[d.id] = [];
        });

        renderAssignmentGrid(productId, ps.serials, ps.destinations, ps.destinations.length > 0 ? ps.destinations[0].qty : 0);
        syncHiddenFields(productId);
    }

    /**
     * Update counters after assignment change.
     */
    function updateCounters(productId) {
        var ps = state[productId];
        if (!ps) return;

        var totalAssigned = 0;
        ps.destinations.forEach(function(d) {
            var count = ps.assignments[d.id] ? ps.assignments[d.id].length : 0;
            totalAssigned += count;
            var el = document.getElementById('counter-' + productId + '-' + d.id);
            if (el) {
                el.textContent = count + '/' + d.qty;
                el.className = 'px-3 py-2 text-center font-medium ';
                if (count === d.qty) {
                    el.className += 'text-green-600';
                } else if (count > d.qty) {
                    el.className += 'text-red-600';
                } else {
                    el.className += 'text-gray-600';
                }
            }
        });

        var unEl = document.getElementById('counter-' + productId + '-unassigned');
        if (unEl) {
            unEl.textContent = String(ps.serials.length - totalAssigned);
        }
    }

    /**
     * Write assignments to hidden inputs for form submission.
     */
    function syncHiddenFields(productId) {
        var ps = state[productId];
        if (!ps) return;

        var form = document.getElementById('wizard-step3-form');
        if (!form) return;

        // Remove existing hidden fields for this product
        var existing = form.querySelectorAll('input[data-serial-hidden="' + productId + '"]');
        existing.forEach(function(el) { el.remove(); });

        // All serials
        var allInput = document.createElement('input');
        allInput.type = 'hidden';
        allInput.name = 'serials_' + productId;
        allInput.value = ps.serials.join('\n');
        allInput.setAttribute('data-serial-hidden', productId);
        form.appendChild(allInput);

        // Per-destination assignments
        ps.destinations.forEach(function(d) {
            if (ps.assignments[d.id] && ps.assignments[d.id].length > 0) {
                var input = document.createElement('input');
                input.type = 'hidden';
                input.name = 'assign_' + productId + '_' + d.id;
                input.value = ps.assignments[d.id].join('\n');
                input.setAttribute('data-serial-hidden', productId);
                form.appendChild(input);
            }
        });
    }

    /**
     * Handle radio button change in assignment grid.
     */
    function onAssignmentChange(productId) {
        var ps = state[productId];
        if (!ps) return;

        // Clear all assignments
        ps.destinations.forEach(function(d) {
            ps.assignments[d.id] = [];
        });

        // Read from radio buttons
        ps.serials.forEach(function(serial, idx) {
            var radios = document.querySelectorAll('input[name="assign_serial_' + productId + '_' + idx + '"]');
            radios.forEach(function(radio) {
                if (radio.checked && radio.value !== 'unassigned') {
                    var addrId = radio.value;
                    if (!ps.assignments[addrId]) ps.assignments[addrId] = [];
                    ps.assignments[addrId].push(serial);
                }
            });
        });

        updateCounters(productId);
        syncHiddenFields(productId);
    }

    /**
     * Handle textarea input - parse, validate, render grid.
     */
    function handleSerialInput(textarea, projectId, csrfToken) {
        var productId = textarea.dataset.productId;
        var required = parseInt(textarea.dataset.required);
        var serials = parseSerials(textarea);

        // Update counter
        var counter = document.getElementById('counter-' + productId);
        if (counter) {
            counter.textContent = 'Entered: ' + serials.length + ' / Required: ' + required;
            if (serials.length === required) {
                counter.className = 'text-sm text-green-600 font-medium';
            } else if (serials.length > required) {
                counter.className = 'text-sm text-red-600 font-medium';
            } else {
                counter.className = 'text-sm text-gray-500';
            }
        }

        // Check for duplicates within input
        var errDiv = document.getElementById('serial-errors-' + productId);
        var seen = {};
        var dupes = [];
        for (var i = 0; i < serials.length; i++) {
            if (seen[serials[i]]) {
                if (dupes.indexOf(serials[i]) === -1) dupes.push(serials[i]);
            }
            seen[serials[i]] = true;
        }

        // Clear previous input-dupe errors
        var inputErr = errDiv.querySelector('.input-errors');
        if (inputErr) inputErr.remove();
        if (dupes.length > 0) {
            var div = document.createElement('div');
            div.className = 'input-errors';
            div.innerHTML = '<p class="text-sm text-red-600">Duplicate serials: ' + dupes.map(escapeHtml).join(', ') + '</p>';
            errDiv.insertBefore(div, errDiv.firstChild);
        }

        // Too many serials error
        var tooManyErr = errDiv.querySelector('.too-many-errors');
        if (tooManyErr) tooManyErr.remove();
        if (serials.length > required) {
            var div2 = document.createElement('div');
            div2.className = 'too-many-errors';
            div2.innerHTML = '<p class="text-sm text-red-600 font-medium">Too many serials entered. Remove ' + (serials.length - required) + ' to proceed.</p>';
            errDiv.appendChild(div2);
        }

        // Deduplicate for state
        var uniqueSerials = [];
        var seenUnique = {};
        for (var j = 0; j < serials.length; j++) {
            if (!seenUnique[serials[j]]) {
                uniqueSerials.push(serials[j]);
                seenUnique[serials[j]] = true;
            }
        }

        // Update state
        if (state[productId]) {
            state[productId].serials = uniqueSerials;
            // Reset assignments when serials change
            state[productId].destinations.forEach(function(d) {
                state[productId].assignments[d.id] = [];
            });
        }

        // Validate against DB (debounced via the caller)
        if (uniqueSerials.length > 0 && projectId) {
            validateSerials(productId, uniqueSerials, projectId, csrfToken);
        }

        // Render assignment grid
        if (state[productId]) {
            var qtyPerSet = state[productId].destinations.length > 0 ? state[productId].destinations[0].qty : 0;
            renderAssignmentGrid(productId, uniqueSerials, state[productId].destinations, qtyPerSet);
            syncHiddenFields(productId);
        }

        // Update submit button state
        updateSubmitButton();
    }

    /**
     * Enable/disable submit button based on validation.
     */
    function updateSubmitButton() {
        var btn = document.getElementById('step3-next');
        if (!btn) return;

        var hasError = false;
        for (var pid in state) {
            var ps = state[pid];
            if (ps.serials.length > ps.required) {
                hasError = true;
                break;
            }
        }
        btn.disabled = hasError;
        if (hasError) {
            btn.classList.add('opacity-50', 'cursor-not-allowed');
        } else {
            btn.classList.remove('opacity-50', 'cursor-not-allowed');
        }
    }

    // Utility
    function escapeHtml(text) {
        var div = document.createElement('div');
        div.appendChild(document.createTextNode(text));
        return div.innerHTML;
    }

    function escapeAttr(text) {
        return text.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/'/g, '&#39;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
    }

    // Public API
    window.SerialAssignment = {
        initProduct: initProduct,
        parseSerials: parseSerials,
        validateSerials: validateSerials,
        renderAssignmentGrid: renderAssignmentGrid,
        autoAssign: autoAssign,
        clearAssignments: clearAssignments,
        updateCounters: updateCounters,
        syncHiddenFields: syncHiddenFields,
        onAssignmentChange: onAssignmentChange,
        handleSerialInput: handleSerialInput
    };
})();
