// Toast Notification System

function showToast(message, type, duration) {
    type = type || 'info';
    duration = duration || 5000;
    var container = document.getElementById('toast-container');
    if (!container) return;

    var toast = document.createElement('div');
    toast.className = 'toast toast-' + type;

    toast.innerHTML =
        '<div class="toast-message">' + escapeHtml(message) + '</div>' +
        '<button class="toast-close" aria-label="Close">' +
            '<svg fill="none" stroke="currentColor" viewBox="0 0 24 24">' +
                '<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>' +
            '</svg>' +
        '</button>';

    container.appendChild(toast);

    var closeBtn = toast.querySelector('.toast-close');
    closeBtn.addEventListener('click', function() {
        removeToast(toast);
    });

    if (duration > 0) {
        setTimeout(function() {
            removeToast(toast);
        }, duration);
    }
}

function removeToast(toast) {
    toast.classList.add('removing');
    setTimeout(function() {
        if (toast.parentNode) {
            toast.parentNode.removeChild(toast);
        }
    }, 300);
}

function escapeHtml(text) {
    var div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

window.showToast = showToast;
