// Sidebar & Navigation

// Toggle expandable nav sections
function toggleNavSection(header) {
    var submenu = header.nextElementSibling;
    if (!submenu) return;

    var isOpen = submenu.classList.contains('open');
    submenu.classList.toggle('open');
    header.classList.toggle('expanded');
}

document.addEventListener('DOMContentLoaded', function() {
    var sidebar = document.getElementById('sidebar');
    var sidebarToggle = document.getElementById('sidebar-toggle');
    var sidebarClose = document.getElementById('sidebar-close');
    var sidebarOverlay = document.getElementById('sidebar-overlay');
    var collapseToggle = document.getElementById('sidebar-collapse-toggle');
    var mainWrapper = document.getElementById('main-content-wrapper');

    if (!sidebar) return;

    // --- Mobile open/close ---
    function openMobile() {
        sidebar.classList.add('mobile-open');
        if (sidebarOverlay) sidebarOverlay.classList.add('visible');
    }

    function closeMobile() {
        sidebar.classList.remove('mobile-open');
        if (sidebarOverlay) sidebarOverlay.classList.remove('visible');
    }

    if (sidebarToggle) sidebarToggle.addEventListener('click', function() {
        sidebar.classList.contains('mobile-open') ? closeMobile() : openMobile();
    });

    if (sidebarClose) sidebarClose.addEventListener('click', closeMobile);
    if (sidebarOverlay) sidebarOverlay.addEventListener('click', closeMobile);

    // --- Desktop collapse/expand ---
    if (collapseToggle) {
        collapseToggle.addEventListener('click', function() {
            sidebar.classList.toggle('collapsed');
            if (mainWrapper) {
                mainWrapper.classList.toggle('lg:ml-64');
                mainWrapper.classList.toggle('lg:ml-16');
            }
        });
    }

    // Close mobile sidebar on resize to desktop
    window.addEventListener('resize', function() {
        if (window.innerWidth >= 1024) closeMobile();
    });

    // --- User Menu Dropdown ---
    var userMenuToggle = document.getElementById('user-menu-toggle');
    var userMenuDropdown = document.getElementById('user-menu-dropdown');

    if (userMenuToggle && userMenuDropdown) {
        userMenuToggle.addEventListener('click', function(e) {
            e.stopPropagation();
            userMenuDropdown.classList.toggle('hidden');
        });

        document.addEventListener('click', function(e) {
            if (!userMenuDropdown.classList.contains('hidden')) {
                var container = document.getElementById('user-menu-container');
                if (container && !container.contains(e.target)) {
                    userMenuDropdown.classList.add('hidden');
                }
            }
        });
    }
});
