/**
 * Flowra Frontend Utilities
 * Provides core functionality: flash messages, HTMX handlers, toasts, modals, keyboard shortcuts
 */

(function() {
    'use strict';

    // ===== Configuration =====
    var config = {
        flashAutoHideDelay: 5000,
        toastAutoHideDelay: 5000,
        wsReconnectDelay: 3000,
        wsMaxReconnectAttempts: 5,
        typingIndicatorDelay: 300,
        typingIndicatorHideDelay: 3000
    };

    // ===== State =====
    var state = {
        wsReconnectAttempts: 0,
        undoStack: [],
        formStates: new Map()
    };

    // ===== Flash Messages =====
    function setupFlashMessages() {
        var flashMessages = document.querySelectorAll('.flash:not([data-flash-setup])');
        flashMessages.forEach(function(flash) {
            flash.setAttribute('data-flash-setup', 'true');

            // Add role for screen readers
            flash.setAttribute('role', 'alert');
            flash.setAttribute('aria-live', 'polite');

            // Auto-hide after delay
            setTimeout(function() {
                hideFlash(flash);
            }, config.flashAutoHideDelay);

            // Setup close button
            var closeBtn = flash.querySelector('.close');
            if (closeBtn) {
                closeBtn.addEventListener('click', function(e) {
                    e.preventDefault();
                    hideFlash(flash);
                });
            }
        });
    }

    function hideFlash(flash) {
        flash.style.opacity = '0';
        flash.style.transition = 'opacity 0.3s';
        setTimeout(function() {
            flash.remove();
        }, 300);
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
            var xhr = evt.detail.xhr;

            // Check for HX-Redirect header (used for auth redirects)
            var redirectUrl = xhr.getResponseHeader('Hx-Redirect') || xhr.getResponseHeader('HX-Redirect');
            if (redirectUrl) {
                window.location.href = redirectUrl;
                return;
            }

            // For 401 errors without redirect, go to login
            if (xhr.status === 401) {
                window.location.href = '/login';
                return;
            }

            var message = getErrorMessage(xhr);
            showToast(message, 'error');
        });

        // Handle HTMX request timeout
        document.body.addEventListener('htmx:timeout', function(evt) {
            console.error('HTMX Timeout:', evt.detail);
            showToast('Request timed out. Please try again.', 'error');
        });

        // Setup loading indicator on send
        document.body.addEventListener('htmx:beforeSend', function(evt) {
            var target = evt.detail.elt;
            if (target.hasAttribute('data-loading-text')) {
                target.dataset.originalText = target.innerText;
                target.innerText = target.getAttribute('data-loading-text');
                target.setAttribute('aria-busy', 'true');
                target.disabled = true;
            }
        });

        // Reset loading indicator on response
        document.body.addEventListener('htmx:afterRequest', function(evt) {
            var target = evt.detail.elt;
            if (target.dataset.originalText) {
                target.innerText = target.dataset.originalText;
                delete target.dataset.originalText;
                target.removeAttribute('aria-busy');
                target.disabled = false;
            }
        });

        // Re-initialize components after HTMX swap
        document.body.addEventListener('htmx:afterSwap', function() {
            setupFlashMessages();
            setupFocusTraps();
            restoreScrollPosition();
        });

        // Save scroll position before navigation
        document.body.addEventListener('htmx:beforeRequest', function(evt) {
            if (evt.detail.boosted) {
                saveScrollPosition();
            }
        });
    }

    // ===== Error Message Extraction =====
    function getErrorMessage(xhr) {
        if (xhr.status === 0) {
            return 'Network error. Please check your connection.';
        }
        if (xhr.status === 401) {
            return 'Session expired. Please log in again.';
        }
        if (xhr.status === 403) {
            return 'You don\'t have permission to perform this action.';
        }
        if (xhr.status === 404) {
            return 'Resource not found.';
        }
        if (xhr.status >= 500) {
            return 'Server error. Please try again later.';
        }

        try {
            var response = JSON.parse(xhr.responseText);
            return response.message || response.error || 'An error occurred.';
        } catch (e) {
            return 'An error occurred. Please try again.';
        }
    }

    // ===== Toast Notifications =====
    function showToast(message, type) {
        type = type || 'info';

        // Create toast container if it doesn't exist
        var container = document.getElementById('toast-container');
        if (!container) {
            container = document.createElement('div');
            container.id = 'toast-container';
            container.setAttribute('role', 'region');
            container.setAttribute('aria-label', 'Notifications');
            container.setAttribute('aria-live', 'polite');
            document.body.appendChild(container);
        }

        // Create toast element
        var toast = document.createElement('article');
        toast.className = 'flash flash-' + type + ' fade-in';
        toast.setAttribute('role', 'alert');
        toast.style.margin = '0';
        toast.style.minWidth = '250px';

        var closeBtn = document.createElement('button');
        closeBtn.className = 'close';
        closeBtn.setAttribute('aria-label', 'Dismiss notification');
        closeBtn.innerHTML = '&times;';
        closeBtn.onclick = function() {
            hideToast(toast);
        };

        var messageSpan = document.createElement('span');
        messageSpan.textContent = message;

        toast.appendChild(closeBtn);
        toast.appendChild(messageSpan);
        container.appendChild(toast);

        // Auto-remove after delay
        setTimeout(function() {
            hideToast(toast);
        }, config.toastAutoHideDelay);

        return toast;
    }

    function hideToast(toast) {
        toast.classList.remove('fade-in');
        toast.classList.add('fade-out');
        setTimeout(function() {
            toast.remove();
        }, 300);
    }

    // Expose showToast globally
    window.showToast = showToast;

    // ===== Scroll Position Management =====
    function saveScrollPosition() {
        sessionStorage.setItem('scrollPosition', window.scrollY.toString());
    }

    function restoreScrollPosition() {
        var position = sessionStorage.getItem('scrollPosition');
        if (position) {
            window.scrollTo(0, parseInt(position, 10));
            sessionStorage.removeItem('scrollPosition');
        }
    }

    // Scroll to bottom of an element (for chat)
    function scrollToBottom(elementId) {
        var element = document.getElementById(elementId);
        if (element) {
            element.scrollTop = element.scrollHeight;
        }
    }
    window.scrollToBottom = scrollToBottom;

    // ===== Modal / Dialog Management =====
    function setupModalEscapeClose() {
        document.addEventListener('keydown', function(evt) {
            if (evt.key === 'Escape') {
                var openModals = document.querySelectorAll('dialog[open]');
                openModals.forEach(function(modal) {
                    modal.close();
                });

                // Also close dropdown menus
                var openDropdowns = document.querySelectorAll('details.dropdown[open]');
                openDropdowns.forEach(function(dropdown) {
                    dropdown.removeAttribute('open');
                });
            }
        });
    }

    // Focus trap for modals
    function setupFocusTraps() {
        var modals = document.querySelectorAll('dialog');
        modals.forEach(function(modal) {
            if (modal.hasAttribute('data-focus-trap-setup')) return;
            modal.setAttribute('data-focus-trap-setup', 'true');

            modal.addEventListener('keydown', function(evt) {
                if (evt.key !== 'Tab') return;

                var focusableEls = modal.querySelectorAll(
                    'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
                );
                var firstFocusable = focusableEls[0];
                var lastFocusable = focusableEls[focusableEls.length - 1];

                if (evt.shiftKey && document.activeElement === firstFocusable) {
                    evt.preventDefault();
                    lastFocusable.focus();
                } else if (!evt.shiftKey && document.activeElement === lastFocusable) {
                    evt.preventDefault();
                    firstFocusable.focus();
                }
            });
        });
    }

    // ===== Confirmation Dialogs =====
    function confirmAction(message) {
        return window.confirm(message || 'Are you sure you want to proceed?');
    }
    window.confirmAction = confirmAction;

    function setupConfirmations() {
        document.body.addEventListener('click', function(evt) {
            var target = evt.target.closest('[data-confirm]');
            if (target) {
                var message = target.getAttribute('data-confirm');
                if (!confirmAction(message)) {
                    evt.preventDefault();
                    evt.stopPropagation();
                }
            }
        }, true);
    }

    // ===== Undo System =====
    function pushUndo(action) {
        state.undoStack.push({
            action: action,
            timestamp: Date.now()
        });

        // Keep only last 10 actions
        if (state.undoStack.length > 10) {
            state.undoStack.shift();
        }
    }

    function showUndoToast(message, undoCallback) {
        var toast = showToast(message + ' ', 'info');

        var undoBtn = document.createElement('button');
        undoBtn.textContent = 'Undo';
        undoBtn.className = 'secondary outline';
        undoBtn.style.marginLeft = '0.5rem';
        undoBtn.style.padding = '0.25rem 0.5rem';
        undoBtn.onclick = function(e) {
            e.stopPropagation();
            undoCallback();
            hideToast(toast);
        };

        toast.querySelector('span').appendChild(undoBtn);
    }
    window.showUndoToast = showUndoToast;

    // ===== Form State Preservation =====
    function saveFormState(form) {
        if (!form.id) return;

        var formData = new FormData(form);
        var data = {};
        formData.forEach(function(value, key) {
            data[key] = value;
        });

        state.formStates.set(form.id, data);
    }

    function restoreFormState(form) {
        if (!form.id) return;

        var data = state.formStates.get(form.id);
        if (!data) return;

        Object.keys(data).forEach(function(key) {
            var input = form.elements[key];
            if (input) {
                input.value = data[key];
            }
        });
    }

    function setupFormStatePreservation() {
        document.body.addEventListener('input', function(evt) {
            var form = evt.target.closest('form[data-preserve-state]');
            if (form) {
                saveFormState(form);
            }
        });

        document.body.addEventListener('htmx:afterSwap', function(evt) {
            var forms = evt.detail.target.querySelectorAll('form[data-preserve-state]');
            forms.forEach(restoreFormState);
        });
    }

    // ===== Keyboard Shortcuts =====
    function setupKeyboardShortcuts() {
        document.addEventListener('keydown', function(evt) {
            // Skip if user is typing in an input
            if (isTypingInInput()) return;

            // Ctrl+K or Cmd+K - Quick search
            if ((evt.ctrlKey || evt.metaKey) && evt.key === 'k') {
                evt.preventDefault();
                var searchInput = document.querySelector('[data-quick-search]');
                if (searchInput) {
                    searchInput.focus();
                }
            }

            // Ctrl+Enter or Cmd+Enter - Submit form
            if ((evt.ctrlKey || evt.metaKey) && evt.key === 'Enter') {
                var form = document.activeElement.closest('form');
                if (form) {
                    evt.preventDefault();
                    var submitBtn = form.querySelector('[type="submit"]');
                    if (submitBtn) {
                        submitBtn.click();
                    }
                }
            }

            // ? - Show keyboard shortcuts help
            if (evt.key === '?' && !evt.ctrlKey && !evt.metaKey) {
                showKeyboardShortcutsHelp();
            }
        });
    }

    function isTypingInInput() {
        var active = document.activeElement;
        if (!active) return false;
        var tagName = active.tagName.toLowerCase();
        return tagName === 'input' || tagName === 'textarea' || active.isContentEditable;
    }

    function showKeyboardShortcutsHelp() {
        var existingHelp = document.getElementById('keyboard-shortcuts-help');
        if (existingHelp) {
            existingHelp.close();
            return;
        }

        var dialog = document.createElement('dialog');
        dialog.id = 'keyboard-shortcuts-help';
        dialog.innerHTML = '<article>' +
            '<header><strong>Keyboard Shortcuts</strong></header>' +
            '<table>' +
            '<tr><td><kbd class="kbd">Ctrl</kbd>+<kbd class="kbd">K</kbd></td><td>Quick search</td></tr>' +
            '<tr><td><kbd class="kbd">Ctrl</kbd>+<kbd class="kbd">Enter</kbd></td><td>Submit form</td></tr>' +
            '<tr><td><kbd class="kbd">Esc</kbd></td><td>Close modal/dropdown</td></tr>' +
            '<tr><td><kbd class="kbd">?</kbd></td><td>Show this help</td></tr>' +
            '</table>' +
            '<footer><button onclick="this.closest(\'dialog\').close()">Close</button></footer>' +
            '</article>';

        document.body.appendChild(dialog);
        dialog.showModal();

        dialog.addEventListener('close', function() {
            dialog.remove();
        });
    }

    // ===== WebSocket Reconnection =====
    function setupWebSocketReconnection() {
        document.body.addEventListener('htmx:wsError', function(evt) {
            console.error('WebSocket error:', evt.detail);

            if (state.wsReconnectAttempts < config.wsMaxReconnectAttempts) {
                state.wsReconnectAttempts++;
                showToast('Connection lost. Reconnecting...', 'warning');

                setTimeout(function() {
                    var wsElement = evt.detail.elt;
                    if (wsElement && wsElement.hasAttribute('ws-connect')) {
                        htmx.trigger(wsElement, 'reconnect');
                    }
                }, config.wsReconnectDelay);
            } else {
                showToast('Connection lost. Please refresh the page.', 'error');
            }
        });

        document.body.addEventListener('htmx:wsOpen', function() {
            if (state.wsReconnectAttempts > 0) {
                showToast('Connection restored', 'success');
                state.wsReconnectAttempts = 0;
            }
        });
    }

    // ===== Live Region for Announcements =====
    function announce(message, priority) {
        priority = priority || 'polite';

        var region = document.getElementById('live-announcer');
        if (!region) {
            region = document.createElement('div');
            region.id = 'live-announcer';
            region.className = 'sr-only';
            region.setAttribute('aria-live', priority);
            region.setAttribute('aria-atomic', 'true');
            document.body.appendChild(region);
        }

        region.setAttribute('aria-live', priority);
        region.textContent = '';

        // Use setTimeout to ensure the change is announced
        setTimeout(function() {
            region.textContent = message;
        }, 100);
    }
    window.announce = announce;

    // ===== Progress Indicators =====
    function showProgress(containerId, message) {
        var container = document.getElementById(containerId);
        if (!container) return;

        var overlay = document.createElement('div');
        overlay.className = 'loading-overlay';
        overlay.innerHTML = '<div class="loading-spinner">' +
            '<div class="spinner"></div>' +
            (message ? '<span>' + message + '</span>' : '') +
            '</div>';

        container.style.position = 'relative';
        container.appendChild(overlay);

        return function hideProgress() {
            overlay.remove();
        };
    }
    window.showProgress = showProgress;

    // ===== Initialize =====
    function init() {
        setupFlashMessages();
        setupHTMXHandlers();
        setupModalEscapeClose();
        setupConfirmations();
        setupFocusTraps();
        setupKeyboardShortcuts();
        setupFormStatePreservation();
        setupWebSocketReconnection();
    }

    // Run on DOMContentLoaded
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();
