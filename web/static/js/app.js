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
        wsReconnectBaseDelay: 1000,
        wsReconnectMaxDelay: 30000,
        wsMaxReconnectAttempts: 10,
        typingIndicatorDelay: 300,
        typingIndicatorHideDelay: 3000,
        searchDebounceDelay: 300
    };

    // ===== State =====
    var state = {
        wsReconnectAttempts: 0,
        wsReconnectTimeoutId: null,
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
                openGlobalSearch();
                return;
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

    // ===== Global Search (Cmd+K) =====
    var searchState = {
        cache: {},
        debounceTimer: null,
        selectedIndex: -1,
        results: []
    };

    function getWorkspaceIdFromUrl() {
        var match = window.location.pathname.match(/\/workspaces\/([^/]+)/);
        return match ? match[1] : null;
    }

    function openGlobalSearch() {
        var existing = document.getElementById('global-search-dialog');
        if (existing) {
            existing.close();
            return;
        }

        var workspaceId = getWorkspaceIdFromUrl();

        var dialog = document.createElement('dialog');
        dialog.id = 'global-search-dialog';
        dialog.className = 'global-search-dialog';
        dialog.innerHTML =
            '<div class="search-container">' +
                '<div class="search-input-wrapper">' +
                    '<span class="search-icon" aria-hidden="true">🔍</span>' +
                    '<input type="search" id="global-search-input" ' +
                        'placeholder="Search chats and tasks..." ' +
                        'autocomplete="off" aria-label="Search" />' +
                    '<kbd class="kbd search-kbd">Esc</kbd>' +
                '</div>' +
                '<div id="global-search-results" class="search-results" role="listbox" aria-label="Search results">' +
                    '<div class="search-hint">' +
                        (workspaceId
                            ? 'Type to search chats and tasks in this workspace'
                            : 'Navigate to a workspace to search') +
                    '</div>' +
                '</div>' +
            '</div>';

        document.body.appendChild(dialog);
        dialog.showModal();

        var input = document.getElementById('global-search-input');
        input.focus();

        // Reset state
        searchState.selectedIndex = -1;
        searchState.results = [];

        input.addEventListener('input', function() {
            var query = input.value.trim();
            if (searchState.debounceTimer) {
                clearTimeout(searchState.debounceTimer);
            }
            if (!query) {
                renderSearchResults([], '', workspaceId);
                return;
            }
            searchState.debounceTimer = setTimeout(function() {
                performSearch(query, workspaceId);
            }, config.searchDebounceDelay);
        });

        // Keyboard navigation
        input.addEventListener('keydown', function(evt) {
            var resultsEl = document.getElementById('global-search-results');
            var items = resultsEl ? resultsEl.querySelectorAll('.search-result-item') : [];

            if (evt.key === 'ArrowDown') {
                evt.preventDefault();
                searchState.selectedIndex = Math.min(searchState.selectedIndex + 1, items.length - 1);
                updateSearchSelection(items);
            } else if (evt.key === 'ArrowUp') {
                evt.preventDefault();
                searchState.selectedIndex = Math.max(searchState.selectedIndex - 1, 0);
                updateSearchSelection(items);
            } else if (evt.key === 'Enter' && searchState.selectedIndex >= 0 && items.length > 0) {
                evt.preventDefault();
                items[searchState.selectedIndex].click();
            }
        });

        // Click on backdrop to close
        dialog.addEventListener('click', function(evt) {
            if (evt.target === dialog) {
                dialog.close();
            }
        });

        dialog.addEventListener('close', function() {
            if (searchState.debounceTimer) {
                clearTimeout(searchState.debounceTimer);
            }
            dialog.remove();
        });
    }

    function performSearch(query, workspaceId) {
        if (!workspaceId) {
            renderSearchResults([], query, workspaceId);
            return;
        }

        var lowerQuery = query.toLowerCase();
        var cacheKey = workspaceId;

        if (searchState.cache[cacheKey]) {
            var filtered = filterResults(searchState.cache[cacheKey], lowerQuery);
            renderSearchResults(filtered, query, workspaceId);
            return;
        }

        // Show loading
        var resultsEl = document.getElementById('global-search-results');
        if (resultsEl) {
            resultsEl.innerHTML = '<div class="search-loading" aria-busy="true">Searching...</div>';
        }

        var basePath = '/api/v1/workspaces/' + workspaceId;
        var allData = { chats: [], tasks: [] };
        var pending = 2;

        function checkDone() {
            pending--;
            if (pending === 0) {
                searchState.cache[cacheKey] = allData;
                // Clear cache after 60 seconds
                setTimeout(function() {
                    delete searchState.cache[cacheKey];
                }, 60000);
                var filtered = filterResults(allData, lowerQuery);
                renderSearchResults(filtered, query, workspaceId);
            }
        }

        fetch(basePath + '/chats?limit=100')
            .then(function(r) { return r.ok ? r.json() : { chats: [] }; })
            .then(function(data) { allData.chats = data.chats || []; })
            .catch(function() {})
            .finally(checkDone);

        fetch(basePath + '/tasks?per_page=100')
            .then(function(r) { return r.ok ? r.json() : { tasks: [] }; })
            .then(function(data) { allData.tasks = data.tasks || []; })
            .catch(function() {})
            .finally(checkDone);
    }

    function filterResults(data, lowerQuery) {
        var results = [];

        data.chats.forEach(function(chat) {
            if (chat.name && chat.name.toLowerCase().indexOf(lowerQuery) !== -1) {
                results.push({ type: 'chat', id: chat.id, name: chat.name, chatType: chat.type });
            }
        });

        data.tasks.forEach(function(task) {
            if (task.title && task.title.toLowerCase().indexOf(lowerQuery) !== -1) {
                results.push({ type: 'task', id: task.id, name: task.title, status: task.status, priority: task.priority });
            }
        });

        return results;
    }

    function renderSearchResults(results, query, workspaceId) {
        var resultsEl = document.getElementById('global-search-results');
        if (!resultsEl) return;

        searchState.selectedIndex = -1;
        searchState.results = results;

        if (!query) {
            resultsEl.innerHTML = '<div class="search-hint">' +
                (workspaceId ? 'Type to search chats and tasks in this workspace' : 'Navigate to a workspace to search') +
                '</div>';
            return;
        }

        if (results.length === 0) {
            resultsEl.innerHTML = '<div class="search-empty">No results for "<strong>' + escapeHtml(query) + '</strong>"</div>';
            return;
        }

        var chats = results.filter(function(r) { return r.type === 'chat'; });
        var tasks = results.filter(function(r) { return r.type === 'task'; });
        var html = '';

        if (chats.length > 0) {
            html += '<div class="search-group"><div class="search-group-label">Chats</div>';
            chats.forEach(function(chat) {
                var icon = chat.chatType === 'direct' ? '💬' : '📢';
                html += '<a class="search-result-item" role="option" ' +
                    'href="/workspaces/' + workspaceId + '/chats/' + chat.id + '">' +
                    '<span class="result-icon" aria-hidden="true">' + icon + '</span>' +
                    '<span class="result-name">' + highlightMatch(escapeHtml(chat.name), query) + '</span>' +
                    '<span class="result-meta">' + escapeHtml(chat.chatType || '') + '</span>' +
                    '</a>';
            });
            html += '</div>';
        }

        if (tasks.length > 0) {
            html += '<div class="search-group"><div class="search-group-label">Tasks</div>';
            tasks.forEach(function(task) {
                html += '<a class="search-result-item" role="option" ' +
                    'href="/workspaces/' + workspaceId + '/board?task=' + task.id + '">' +
                    '<span class="result-icon" aria-hidden="true">📋</span>' +
                    '<span class="result-name">' + highlightMatch(escapeHtml(task.name), query) + '</span>' +
                    '<span class="result-meta">' + escapeHtml(task.status || '') + '</span>' +
                    '</a>';
            });
            html += '</div>';
        }

        resultsEl.innerHTML = html;

        // Wire click to close dialog
        resultsEl.querySelectorAll('.search-result-item').forEach(function(item) {
            item.addEventListener('click', function() {
                var dialog = document.getElementById('global-search-dialog');
                if (dialog) dialog.close();
            });
        });
    }

    function updateSearchSelection(items) {
        items.forEach(function(item, i) {
            if (i === searchState.selectedIndex) {
                item.classList.add('selected');
                item.scrollIntoView({ block: 'nearest' });
            } else {
                item.classList.remove('selected');
            }
        });
    }

    function highlightMatch(text, query) {
        if (!query) return text;
        var escaped = escapeHtml(query).replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
        var regex = new RegExp('(' + escaped + ')', 'gi');
        return text.replace(regex, '<mark>$1</mark>');
    }

    function escapeHtml(str) {
        var div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
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

    // ===== WebSocket Connection Status =====
    var wsStatus = {
        element: null,
        dot: null,
        state: 'disconnected',

        init: function() {
            this.element = document.getElementById('ws-status');
            if (!this.element) return;

            this.dot = this.element.querySelector('.status-dot');
            if (!this.dot) return;

            // Click to reconnect when disconnected
            this.element.addEventListener('click', function() {
                if (wsStatus.state === 'disconnected') {
                    wsStatus.reconnect();
                }
            });
        },

        setState: function(newState, message) {
            this.state = newState;
            if (!this.dot || !this.element) return;

            this.dot.className = 'status-dot ' + newState;
            this.element.title = message;
        },

        setConnected: function() {
            this.setState('connected', 'Real-time updates active');
        },

        setConnecting: function(attempt, maxAttempts) {
            this.setState('connecting', 'Reconnecting... (attempt ' + attempt + '/' + maxAttempts + ')');
        },

        setDisconnected: function() {
            this.setState('disconnected', 'Offline - click to reconnect');
        },

        reconnect: function() {
            // Reset reconnection counter and trigger reconnect
            state.wsReconnectAttempts = 0;
            if (state.wsReconnectTimeoutId) {
                clearTimeout(state.wsReconnectTimeoutId);
                state.wsReconnectTimeoutId = null;
            }
            scheduleReconnect();
        }
    };

    // ===== WebSocket Reconnection with Exponential Backoff =====
    function calculateReconnectDelay() {
        var exponential = Math.min(
            config.wsReconnectBaseDelay * Math.pow(2, state.wsReconnectAttempts),
            config.wsReconnectMaxDelay
        );
        var jitter = Math.random() * 1000;
        return exponential + jitter;
    }

    function scheduleReconnect() {
        if (state.wsReconnectAttempts >= config.wsMaxReconnectAttempts) {
            wsStatus.setDisconnected();
            showToast('Connection lost. Click status indicator to retry.', 'error');
            return;
        }

        state.wsReconnectAttempts++;
        var delay = calculateReconnectDelay();

        wsStatus.setConnecting(state.wsReconnectAttempts, config.wsMaxReconnectAttempts);
        console.log('WS reconnect attempt ' + state.wsReconnectAttempts + ' in ' + Math.round(delay) + 'ms');

        state.wsReconnectTimeoutId = setTimeout(function() {
            doReconnect();
        }, delay);
    }

    function doReconnect() {
        // Find HTMX WebSocket element and trigger reconnect
        var wsElement = document.querySelector('[ws-connect]');
        if (wsElement) {
            htmx.trigger(wsElement, 'htmx:wsReconnect');
        }
    }

    function resetReconnect() {
        state.wsReconnectAttempts = 0;
        if (state.wsReconnectTimeoutId) {
            clearTimeout(state.wsReconnectTimeoutId);
            state.wsReconnectTimeoutId = null;
        }
        wsStatus.setConnected();
    }

    // ===== WebSocket Reconnection =====
    function setupWebSocketReconnection() {
        document.body.addEventListener('htmx:wsOpen', function() {
            if (state.wsReconnectAttempts > 0) {
                showToast('Connection restored', 'success');
            }
            resetReconnect();
        });

        document.body.addEventListener('htmx:wsError', function(evt) {
            console.error('WebSocket error:', evt.detail);
            scheduleReconnect();
        });

        document.body.addEventListener('htmx:wsClose', function() {
            console.log('WebSocket closed');
            scheduleReconnect();
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

    // ===== Notification Handlers =====
    function setupNotificationHandlers() {
        // Handle notification.new event from WebSocket
        document.body.addEventListener('notification.new', function(evt) {
            if (!evt.detail) return;

            var notification = evt.detail;
            
            // Update badge count
            htmx.trigger(document.body, 'notification-update');
            
            // Show toast notification
            var message = notification.Message || notification.message || 'New notification';
            if (message.length > 60) {
                message = message.substring(0, 57) + '...';
            }
            showToast(message, 'info');
            
            // If dropdown is open, reload it to show new notification
            var dropdown = document.querySelector('.notification-dropdown[open]');
            if (dropdown) {
                htmx.trigger(document.body, 'reload-notifications');
            }
            
            // If on notifications page, reload the list
            var notificationsList = document.getElementById('notifications-list');
            if (notificationsList) {
                htmx.trigger(notificationsList, 'reload-notifications');
            }
            
            // Animate badge
            var badge = document.getElementById('notification-badge');
            if (badge && !badge.classList.contains('hidden')) {
                badge.classList.add('badge-pulse');
                setTimeout(function() {
                    badge.classList.remove('badge-pulse');
                }, 600);
            }
        });
    }

    // ===== Task Detail Helpers =====

    /**
     * Close the task sidebar panel.
     */
    function closeTaskSidebar() {
        var sidebar = document.querySelector('.task-sidebar');
        if (sidebar) {
            sidebar.style.display = 'none';
            var layout = document.querySelector('.chat-layout');
            if (layout) {
                layout.classList.remove('with-task-sidebar');
            }
        }
    }
    window.closeTaskSidebar = closeTaskSidebar;

    /**
     * Handle task deletion: close sidebar and remove board card.
     * @param {string} taskId
     */
    function handleTaskDeleted(taskId) {
        var card = document.getElementById('task-' + taskId);
        if (card) {
            card.remove();
            document.querySelectorAll('.board-column').forEach(function(column) {
                var count = column.querySelectorAll('.task-card').length;
                var countEl = column.querySelector('.column-count');
                if (countEl) countEl.textContent = count;
            });
        }
        closeTaskSidebar();
        if (typeof showToast === 'function') {
            showToast('Task deleted', 'success');
        }
    }
    window.handleTaskDeleted = handleTaskDeleted;

    /**
     * Handle keyboard shortcuts for inline title editing.
     * Enter = save, Escape = cancel.
     */
    function handleEditKeydown(event, form) {
        if (event.key === 'Enter') {
            event.preventDefault();
            htmx.trigger(form, 'submit');
        } else if (event.key === 'Escape') {
            event.preventDefault();
            var cancelBtn = form.querySelector('button[type="button"]');
            if (cancelBtn) cancelBtn.click();
        }
    }
    window.handleEditKeydown = handleEditKeydown;

    /**
     * Handle keyboard shortcuts for inline description editing.
     * Ctrl+Enter = save, Escape = cancel.
     */
    function handleDescriptionKeydown(event, form) {
        if (event.key === 'Enter' && (event.ctrlKey || event.metaKey)) {
            event.preventDefault();
            htmx.trigger(form, 'submit');
        } else if (event.key === 'Escape') {
            event.preventDefault();
            var cancelBtn = form.querySelector('button[type="button"]');
            if (cancelBtn) cancelBtn.click();
        }
    }
    window.handleDescriptionKeydown = handleDescriptionKeydown;

    /**
     * Set a quick due date relative to today.
     * @param {string} taskId
     * @param {number} daysFromNow
     */
    function setQuickDate(taskId, daysFromNow) {
        var date = new Date();
        date.setDate(date.getDate() + daysFromNow);
        var formatted = date.toISOString().split('T')[0];
        var sidebar = document.getElementById('task-sidebar-' + taskId);
        var dateInput = sidebar
            ? sidebar.querySelector('.date-input')
            : document.querySelector('.date-input');
        if (dateInput) {
            dateInput.value = formatted;
            htmx.trigger(dateInput, 'change');
        }
    }
    window.setQuickDate = setQuickDate;

    // ===== Dark Mode Toggle =====
    function getPreferredTheme() {
        var stored = localStorage.getItem('flowra-theme');
        if (stored) return stored;
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }

    function applyTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
        updateThemeIcons(theme);
    }

    function updateThemeIcons(theme) {
        var icons = document.querySelectorAll('.theme-icon');
        icons.forEach(function(icon) {
            icon.textContent = theme === 'dark' ? '☀️' : '🌙';
        });
    }

    function toggleTheme() {
        var current = document.documentElement.getAttribute('data-theme') || getPreferredTheme();
        var next = current === 'dark' ? 'light' : 'dark';
        localStorage.setItem('flowra-theme', next);
        applyTheme(next);
    }
    window.toggleTheme = toggleTheme;

    // Listen for OS theme changes when no manual preference is stored
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', function(e) {
        if (!localStorage.getItem('flowra-theme')) {
            applyTheme(e.matches ? 'dark' : 'light');
        }
    });

    // ===== Initialize =====
    function init() {
        applyTheme(getPreferredTheme());
        setupFlashMessages();
        setupHTMXHandlers();
        setupModalEscapeClose();
        setupConfirmations();
        setupFocusTraps();
        setupKeyboardShortcuts();
        setupFormStatePreservation();
        setupWebSocketReconnection();
        setupNotificationHandlers();
        wsStatus.init();
    }

    // Run on DOMContentLoaded
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();
