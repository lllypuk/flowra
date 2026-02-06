/**
 * Chat-specific JavaScript functionality
 * Provides typing indicators, tag autocomplete, and message utilities
 */

// ============================================================
// Auto-resize textarea
// ============================================================

/**
 * Auto-resize textarea based on content
 * @param {HTMLTextAreaElement} textarea - The textarea element to resize
 */
function autoResize(textarea) {
    if (!textarea) return;

    // Reset height to calculate proper scroll height
    textarea.style.height = 'auto';

    // Set height based on content, max 160px
    const maxHeight = 160;
    const newHeight = Math.min(textarea.scrollHeight, maxHeight);
    textarea.style.height = newHeight + 'px';
}

// ============================================================
// Scroll utilities
// ============================================================

/**
 * Scroll container to bottom
 * @param {string} elementId - ID of the element to scroll
 */
function scrollToBottom(elementId) {
    var element = document.getElementById(elementId);
    if (element) {
        // Use smooth scrolling for better UX
        element.scrollTo({
            top: element.scrollHeight,
            behavior: 'smooth'
        });
    }
}

/**
 * Scroll container to bottom instantly (no animation)
 * @param {string} elementId - ID of the element to scroll
 */
function scrollToBottomInstant(elementId) {
    var element = document.getElementById(elementId);
    if (element) {
        element.scrollTop = element.scrollHeight;
    }
}

// ============================================================
// Typing indicator
// ============================================================

var typingTimeouts = {};
var typingHideTimeouts = {};

/**
 * Handle typing event - sends typing indicator via WebSocket
 * @param {string} chatId - The chat ID
 */
function handleTyping(chatId) {
    // Clear existing timeout for this chat
    if (typingTimeouts[chatId]) {
        clearTimeout(typingTimeouts[chatId]);
    }

    // Debounce typing indicator sends
    typingTimeouts[chatId] = setTimeout(function() {
        sendTypingIndicator(chatId);
    }, 300);
}

/**
 * Send typing indicator via WebSocket
 * @param {string} chatId - The chat ID
 */
function sendTypingIndicator(chatId) {
    // Find WebSocket connection from HTMX
    var wsElements = document.querySelectorAll('[hx-ext="ws"]');

    for (var i = 0; i < wsElements.length; i++) {
        var ws = wsElements[i].__htmx_ws;
        if (ws && ws.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({
                type: 'chat.typing',
                chat_id: chatId
            }));
            break;
        }
    }
}

/**
 * Show typing indicator for a user
 * @param {string} username - The username who is typing
 * @param {string} chatId - The chat ID
 */
function showTypingIndicator(username, chatId) {
    var indicator = document.getElementById('typing-indicator-' + chatId);
    var usersSpan = indicator ? indicator.querySelector('.typing-users, #typing-users') : null;

    if (indicator && usersSpan) {
        usersSpan.textContent = username;
        indicator.classList.remove('hidden');

        // Clear existing hide timeout
        if (typingHideTimeouts[chatId]) {
            clearTimeout(typingHideTimeouts[chatId]);
        }

        // Hide after 3 seconds of no updates
        typingHideTimeouts[chatId] = setTimeout(function() {
            indicator.classList.add('hidden');
        }, 3000);
    }
}

// ============================================================
// Tag autocomplete
// ============================================================

var activeAutocompleteInput = null;

/**
 * Initialize tag autocomplete on all message inputs
 */
function initTagAutocomplete() {
    var inputs = document.querySelectorAll('.message-form textarea');

    inputs.forEach(function(input) {
        // Remove existing listeners to prevent duplicates
        input.removeEventListener('input', handleTagAutocompleteEvent);
        input.removeEventListener('keydown', handleAutocompleteNavigationEvent);

        // Add listeners
        input.addEventListener('input', handleTagAutocompleteEvent);
        input.addEventListener('keydown', handleAutocompleteNavigationEvent);
    });

    // Close dropdown when clicking outside
    document.addEventListener('click', function(e) {
        if (!e.target.closest('.message-input-wrapper')) {
            closeAllAutocompleteDropdowns();
        }
    });

    // Reposition dropdown on scroll
    var messagesContainers = document.querySelectorAll('.messages-container');
    messagesContainers.forEach(function(container) {
        container.addEventListener('scroll', function() {
            if (activeAutocompleteInput) {
                var wrapper = activeAutocompleteInput.closest('.message-input-wrapper');
                var dropdown = wrapper ? wrapper.querySelector('.autocomplete-dropdown') : null;
                if (dropdown && !dropdown.classList.contains('hidden')) {
                    positionDropdown(activeAutocompleteInput, dropdown);
                }
            }
        });
    });

    // Reposition dropdown on window resize
    window.addEventListener('resize', function() {
        if (activeAutocompleteInput) {
            var wrapper = activeAutocompleteInput.closest('.message-input-wrapper');
            var dropdown = wrapper ? wrapper.querySelector('.autocomplete-dropdown') : null;
            if (dropdown && !dropdown.classList.contains('hidden')) {
                positionDropdown(activeAutocompleteInput, dropdown);
            }
        }
    });
}

/**
 * Event handler for tag autocomplete
 * @param {Event} e - Input event
 */
function handleTagAutocompleteEvent(e) {
    handleTagAutocomplete(e.target);
}

/**
 * Event handler for autocomplete navigation
 * @param {KeyboardEvent} e - Keydown event
 */
function handleAutocompleteNavigationEvent(e) {
    handleAutocompleteNavigation(e);
}

/**
 * Position dropdown above textarea
 * @param {HTMLTextAreaElement} textarea - The textarea element
 * @param {HTMLElement} dropdown - The dropdown element
 */
function positionDropdown(textarea, dropdown) {
    var rect = textarea.getBoundingClientRect();
    var dropdownHeight = dropdown.offsetHeight || 250; // fallback to max-height

    // Position dropdown above the textarea
    dropdown.style.position = 'fixed';
    dropdown.style.left = rect.left + 'px';
    dropdown.style.width = rect.width + 'px';
    dropdown.style.bottom = (window.innerHeight - rect.top + 8) + 'px';
    dropdown.style.top = 'auto';
}

/**
 * Handle tag autocomplete on input
 * @param {HTMLTextAreaElement} textarea - The textarea element
 */
function handleTagAutocomplete(textarea) {
    var value = textarea.value;
    var cursorPos = textarea.selectionStart;
    var textBeforeCursor = value.substring(0, cursorPos);

    // Check if user just typed # (start of tag)
    var hashMatch = textBeforeCursor.match(/#(\w*)$/);

    // Find the dropdown for this textarea
    var wrapper = textarea.closest('.message-input-wrapper');
    var dropdown = wrapper ? wrapper.querySelector('.autocomplete-dropdown') : null;

    if (!dropdown) return;

    if (hashMatch) {
        activeAutocompleteInput = textarea;
        var filter = hashMatch[1].toLowerCase();
        var items = dropdown.querySelectorAll('li');
        var hasVisible = false;

        items.forEach(function(item) {
            var tag = (item.dataset.tag || '').toLowerCase();
            var label = item.textContent.toLowerCase();

            if (tag.includes(filter) || label.includes(filter) || filter === '') {
                item.style.display = '';
                hasVisible = true;
            } else {
                item.style.display = 'none';
            }
        });

        if (hasVisible) {
            dropdown.classList.remove('hidden');
            // Position dropdown
            positionDropdown(textarea, dropdown);
            // Reset active state
            items.forEach(function(item) {
                item.classList.remove('active');
            });
            // Set first visible item as active
            var firstVisible = dropdown.querySelector('li:not([style*="display: none"])');
            if (firstVisible) {
                firstVisible.classList.add('active');
            }
        } else {
            dropdown.classList.add('hidden');
        }
    } else {
        dropdown.classList.add('hidden');
        activeAutocompleteInput = null;
    }
}

/**
 * Handle keyboard navigation in autocomplete dropdown
 * @param {KeyboardEvent} e - Keydown event
 */
function handleAutocompleteNavigation(e) {
    var wrapper = e.target.closest('.message-input-wrapper');
    var dropdown = wrapper ? wrapper.querySelector('.autocomplete-dropdown') : null;

    if (!dropdown || dropdown.classList.contains('hidden')) return;

    var items = Array.from(dropdown.querySelectorAll('li:not([style*="display: none"])'));
    if (items.length === 0) return;

    var active = dropdown.querySelector('li.active');
    var index = items.indexOf(active);

    switch (e.key) {
        case 'ArrowDown':
            e.preventDefault();
            if (active) active.classList.remove('active');
            index = (index + 1) % items.length;
            items[index].classList.add('active');
            items[index].scrollIntoView({ block: 'nearest' });
            break;

        case 'ArrowUp':
            e.preventDefault();
            if (active) active.classList.remove('active');
            index = index <= 0 ? items.length - 1 : index - 1;
            items[index].classList.add('active');
            items[index].scrollIntoView({ block: 'nearest' });
            break;

        case 'Enter':
        case 'Tab':
            if (active && !dropdown.classList.contains('hidden')) {
                e.preventDefault();
                insertTag(e.target, active.dataset.tag);
                dropdown.classList.add('hidden');
                activeAutocompleteInput = null;
            }
            break;

        case 'Escape':
            e.preventDefault();
            dropdown.classList.add('hidden');
            activeAutocompleteInput = null;
            break;
    }
}

/**
 * Insert selected tag into textarea
 * @param {HTMLTextAreaElement} textarea - The textarea element
 * @param {string} tag - The tag to insert
 */
function insertTag(textarea, tag) {
    var value = textarea.value;
    var cursorPos = textarea.selectionStart;
    var textBeforeCursor = value.substring(0, cursorPos);
    var textAfterCursor = value.substring(cursorPos);

    // Replace the partial # input with the full tag
    var newText = textBeforeCursor.replace(/#\w*$/, tag + ' ') + textAfterCursor;
    textarea.value = newText;

    // Move cursor after the tag
    var newCursorPos = textBeforeCursor.replace(/#\w*$/, tag + ' ').length;
    textarea.setSelectionRange(newCursorPos, newCursorPos);
    textarea.focus();

    // Trigger resize
    autoResize(textarea);
}

/**
 * Close all autocomplete dropdowns
 */
function closeAllAutocompleteDropdowns() {
    var dropdowns = document.querySelectorAll('.autocomplete-dropdown');
    dropdowns.forEach(function(dropdown) {
        dropdown.classList.add('hidden');
    });
    activeAutocompleteInput = null;
}

// Add click handlers for autocomplete items
document.addEventListener('click', function(e) {
    var item = e.target.closest('.autocomplete-dropdown li');
    if (item && activeAutocompleteInput) {
        insertTag(activeAutocompleteInput, item.dataset.tag);
        item.closest('.autocomplete-dropdown').classList.add('hidden');
        activeAutocompleteInput = null;
    }
});

// ============================================================
// WebSocket message handling
// ============================================================

/**
 * Parse and dispatch WebSocket messages as custom events
 */
document.body.addEventListener('htmx:wsAfterMessage', function(evt) {
    try {
        var msg = JSON.parse(evt.detail.message);
        if (msg.type && msg.data) {
            // Dispatch custom event for the message type
            document.body.dispatchEvent(new CustomEvent(msg.type, {
                detail: msg.data,
                bubbles: true
            }));
        } else if (msg.type) {
            // Dispatch event without data (for presence and typing)
            document.body.dispatchEvent(new CustomEvent(msg.type, {
                detail: msg,
                bubbles: true
            }));
        }
    } catch (e) {
        console.error('Failed to parse WebSocket message:', e);
    }
});

// ============================================================
// Presence handling
// ============================================================

/**
 * Update presence indicator for a user
 * @param {string} userId - User ID
 * @param {boolean} isOnline - Whether user is online
 */
function updateUserPresence(userId, isOnline) {
    // Update all presence dots for this user
    var presenceDots = document.querySelectorAll('[data-user-id="' + userId + '"] .presence-dot');
    presenceDots.forEach(function(dot) {
        if (isOnline) {
            dot.classList.add('online');
            dot.classList.remove('offline');
        } else {
            dot.classList.add('offline');
            dot.classList.remove('online');
        }
    });
    
    // Update online count
    updateOnlineCount();
}

/**
 * Update the online user count display
 */
function updateOnlineCount() {
    var onlineDots = document.querySelectorAll('.presence-dot.online');
    var onlineCount = onlineDots.length;
    
    // Update modal count
    var modalCountEl = document.querySelector('.online-count');
    if (modalCountEl) {
        modalCountEl.textContent = onlineCount + ' online';
    }
    
    // Update header count
    var headerCountEl = document.getElementById('chat-online-count');
    if (headerCountEl) {
        if (onlineCount > 0) {
            headerCountEl.textContent = onlineCount + ' online';
        } else {
            headerCountEl.textContent = '';
        }
    }
}

/**
 * Handle presence change events from WebSocket
 */
document.body.addEventListener('presence.changed', function(evt) {
    if (evt.detail && evt.detail.user_id && typeof evt.detail.is_online === 'boolean') {
        updateUserPresence(evt.detail.user_id, evt.detail.is_online);
    }
});

/**
 * Handle typing indicator events from WebSocket
 */
document.body.addEventListener('chat.typing', function(evt) {
    if (evt.detail && evt.detail.user_id && evt.detail.chat_id) {
        showTypingIndicator(evt.detail.chat_id, evt.detail.user_id);
    }
});

// ============================================================
// Initialization
// ============================================================

/**
 * Initialize chat functionality
 */
function initChat() {
    initTagAutocomplete();

    // Auto-resize all textareas on page
    var textareas = document.querySelectorAll('.message-form textarea');
    textareas.forEach(function(textarea) {
        autoResize(textarea);
    });
}

// Initialize on DOMContentLoaded
document.addEventListener('DOMContentLoaded', initChat);

// Re-initialize after HTMX swaps (for dynamically loaded content)
document.body.addEventListener('htmx:afterSwap', function(evt) {
    // Check if chat content was swapped
    if (evt.detail.target.classList.contains('chat-main') ||
        evt.detail.target.closest('.chat-main') ||
        evt.detail.target.id.startsWith('messages-')) {
        initTagAutocomplete();
    }
});

// Scroll to bottom after messages are loaded
document.body.addEventListener('htmx:afterSettle', function(evt) {
    var target = evt.detail.target;
    if (target && target.id && target.id.startsWith('messages-')) {
        scrollToBottomInstant(target.id);
    }
});
