document.addEventListener('DOMContentLoaded', function() {
    console.log('DC Management Tool initialized');

    document.body.addEventListener('htmx:beforeRequest', function(evt) {
        console.log('HTMX request starting:', evt.detail.path);
    });

    document.body.addEventListener('htmx:afterRequest', function(evt) {
        console.log('HTMX request completed:', evt.detail.path);
    });

    document.body.addEventListener('htmx:responseError', function(evt) {
        console.error('HTMX error:', evt.detail);
        showToast('An error occurred. Please try again.', 'error');
    });
});

function showToast(message, type = 'info') {
    console.log(`[${type.toUpperCase()}] ${message}`);
}

function confirmAction(message) {
    return confirm(message);
}

function formatCurrency(amount) {
    return new Intl.NumberFormat('en-IN', {
        style: 'currency',
        currency: 'INR'
    }).format(amount);
}

function formatDate(dateString) {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-IN');
}
