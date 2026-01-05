/**
 * Flowra Frontend Utilities
 */

(function() {
    'use strict';

    // ===== Flash Messages Auto-hide =====
    function setupFlashMessages() {
        const flashMessages = document.querySelectorAll('.flash');
        flashMessages.forEach(function(flash) {
            setTimeout(function() {
                flash.style.opacity = '0';
                flash.style.transition = 'opacity 0.3s';
                setTimeout(function() {
                    flash.remove();
                }, 300);
            }, 5000);
        });
    }

    // ===== HTMX Event Handlers =====
    function setupHTMXHandlers() {
        // Handle 422 validation errors (swap content anyway)
        document.body.addEventListener('htmx:beforeSwap', function(evt) {
            if (evt.detail.xhr.status === 422) {
                evt.detail.shouldSwap = true;
                evt.detail.isError = false;
            }
        });

        // Handle HTMX errors
        document.body.addEventListener('htmx:responseError', function(evt) {
            console.error('HTMX Error:', evt.detail);
            showToast('An error occurred. Please try again.', 'error');
        });

        // Handle HTMX request timeout
        document.body.addEventListener('htmx:timeout', function(evt) {
            console.error('HTMX Timeout:', evt.detail);
            showToast('Request timed out. Please try again.', 'error');
        });

        // Setup indicator on send
        document.body.addEventListener('htmx:beforeSend', function(evt) {
            const target = evt.detail.elt;
            if (target.hasAttribute('data-loading-text')) {
                target.dataset.originalText = target.innerText;
                target.innerText = target.getAttribute('data-loading-text');
                target.setAttribute('aria-busy', 'true');
            }
        });

        // Reset indicator on response
        document.body.addEventListener('htmx:afterRequest', function(evt) {
            const target = evt.detail.elt;
            if (target.dataset.originalText) {
                target.innerText = target.dataset.originalText;
                delete target.dataset.originalText;
                target.removeAttribute('aria-busy');
            }
        });
    }

    // ===== Toast Notifications =====
    function showToast(message, type) {
        type = type || 'info';

        // Create toast container if it doesn't exist
        let container = document.getElementById('toast-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'toast-container';
            container.style.cssText = 'position: fixed; bottom: 1rem; right: 1rem; z-index: 9999; display: flex; flex-direction: column; gap: 0.5rem;';
            document.body.appendChild(container);
        }

        // Create toast element
        const toast = document.createElement('article');
        toast.className = 'flash flash-' + type;
        toast.style.cssText = 'margin: 0; min-width: 250px; animation: fadeIn 0.3s ease-in;';
        toast.innerHTML = '<button class="close" onclick="this.parentElement.remove()" aria-label="Close">&times;</button>' + message;

        container.appendChild(toast);

        // Auto-remove after 5 seconds
        setTimeout(function() {
            toast.style.opacity = '0';
            toast.style.transition = 'opacity 0.3s';
            setTimeout(function() {
                toast.remove();
            }, 300);
        }, 5000);
    }

    // Expose showToast globally
    window.showToast = showToast;

    // ===== Utility Functions =====

    // Scroll to bottom of an element (for chat)
    function scrollToBottom(elementId) {
        const element = document.getElementById(elementId);
        if (element) {
            element.scrollTop = element.scrollHeight;
        }
    }
    window.scrollToBottom = scrollToBottom;

    // Close modal on Escape key
    function setupModalEscapeClose() {
        document.addEventListener('keydown', function(evt) {
            if (evt.key === 'Escape') {
                const openModals = document.querySelectorAll('dialog[open]');
                openModals.forEach(function(modal) {
                    modal.close();
                });
            }
        });
    }

    // Confirm dialog for destructive actions
    function confirmAction(message) {
        return window.confirm(message || 'Are you sure you want to proceed?');
    }
    window.confirmAction = confirmAction;

    // Setup confirmation on elements with data-confirm attribute
    function setupConfirmations() {
        document.body.addEventListener('click', function(evt) {
            const target = evt.target.closest('[data-confirm]');
            if (target) {
                const message = target.getAttribute('data-confirm');
                if (!confirmAction(message)) {
                    evt.preventDefault();
                    evt.stopPropagation();
                }
            }
        }, true);
    }

    // ===== Initialize =====
    function init() {
        setupFlashMessages();
        setupHTMXHandlers();
        setupModalEscapeClose();
        setupConfirmations();
    }

    // Run on DOMContentLoaded
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    // Re-run flash message setup after HTMX content swap
    document.body.addEventListener('htmx:afterSwap', function() {
        setupFlashMessages();
    });

})();
