// DC Lifecycle Management (Issue & Delete)

function issueDC(projectID, dcID, csrfToken) {
    if (!confirm('Issue this DC? Once issued, it cannot be edited.')) {
        return;
    }

    fetch(`/projects/${projectID}/dcs/${dcID}/issue`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': csrfToken,
        },
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            showToast('DC issued successfully!', 'success');
            setTimeout(() => location.reload(), 500);
        } else {
            showToast('Error: ' + (data.error || 'Failed to issue DC'), 'error');
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showToast('Failed to issue DC. Please try again.', 'error');
    });
}

let pendingDelete = null;

function deleteDC(projectID, dcID, csrfToken, isIssued) {
    const modal = document.getElementById('deleteModal');
    const message = document.getElementById('deleteMessage');

    if (isIssued) {
        message.textContent = 'This is an ISSUED DC. Are you sure you want to delete it? This action cannot be undone. All serial numbers will be freed for reuse. The DC number will NOT be reused.';
    } else {
        message.textContent = 'Are you sure you want to delete this draft DC? This action cannot be undone. All serial numbers will be freed for reuse.';
    }

    pendingDelete = { projectID, dcID, csrfToken };
    modal.classList.remove('hidden');
    modal.classList.add('flex');
}

function closeDeleteModal() {
    const modal = document.getElementById('deleteModal');
    modal.classList.add('hidden');
    modal.classList.remove('flex');
    pendingDelete = null;
}

function confirmDelete() {
    if (!pendingDelete) return;

    const { projectID, dcID, csrfToken } = pendingDelete;

    fetch(`/projects/${projectID}/dcs/${dcID}`, {
        method: 'DELETE',
        headers: {
            'Content-Type': 'application/json',
            'X-CSRF-Token': csrfToken,
        },
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            showToast('DC deleted successfully.', 'success');
            setTimeout(() => {
                window.location.href = `/projects/${projectID}`;
            }, 500);
        } else {
            showToast('Error: ' + (data.error || 'Failed to delete DC'), 'error');
            closeDeleteModal();
        }
    })
    .catch(error => {
        console.error('Error:', error);
        showToast('Failed to delete DC. Please try again.', 'error');
        closeDeleteModal();
    });
}

// Close modal on Escape key
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
        closeDeleteModal();
    }
});

// Close modal on outside click
document.addEventListener('click', function(e) {
    const modal = document.getElementById('deleteModal');
    if (modal && e.target === modal) {
        closeDeleteModal();
    }
});
