/**
 * Kanban Board Drag and Drop
 * Implements HTML5 native drag-and-drop for task cards
 */

let draggedTask = null;

/**
 * Handle drag start event
 * @param {DragEvent} event
 */
function handleDragStart(event) {
    draggedTask = event.target;
    event.target.classList.add('dragging');

    // Set data for the drag operation
    event.dataTransfer.effectAllowed = 'move';
    event.dataTransfer.setData('text/plain', event.target.dataset.taskId);

    // Create a custom drag image
    const ghost = event.target.cloneNode(true);
    ghost.style.opacity = '0.8';
    ghost.style.position = 'absolute';
    ghost.style.top = '-1000px';
    ghost.style.pointerEvents = 'none';
    document.body.appendChild(ghost);
    event.dataTransfer.setDragImage(ghost, 0, 0);

    // Remove ghost after drag starts
    setTimeout(() => ghost.remove(), 0);
}

/**
 * Handle drag end event
 * @param {DragEvent} event
 */
function handleDragEnd(event) {
    event.target.classList.remove('dragging');
    draggedTask = null;

    // Remove all drag-over states
    document.querySelectorAll('.column-cards.drag-over').forEach(col => {
        col.classList.remove('drag-over');
    });
}

/**
 * Handle drag over event - determines where to insert the dragged element
 * @param {DragEvent} event
 */
function handleDragOver(event) {
    event.preventDefault();
    event.dataTransfer.dropEffect = 'move';

    const column = event.currentTarget;
    column.classList.add('drag-over');

    if (!draggedTask) return;

    // Find insertion point based on mouse position
    const afterElement = getDragAfterElement(column, event.clientY);

    if (afterElement) {
        column.insertBefore(draggedTask, afterElement);
    } else {
        // Insert before load-more button if exists, otherwise append
        const loadMore = column.querySelector('.load-more');
        if (loadMore) {
            column.insertBefore(draggedTask, loadMore);
        } else {
            column.appendChild(draggedTask);
        }
    }
}

/**
 * Handle drag leave event
 * @param {DragEvent} event
 */
function handleDragLeave(event) {
    // Only remove drag-over if leaving the column entirely
    const rect = event.currentTarget.getBoundingClientRect();
    if (
        event.clientX < rect.left ||
        event.clientX > rect.right ||
        event.clientY < rect.top ||
        event.clientY > rect.bottom
    ) {
        event.currentTarget.classList.remove('drag-over');
    }
}

/**
 * Handle drop event - updates task status via API
 * @param {DragEvent} event
 */
function handleDrop(event) {
    event.preventDefault();

    const column = event.currentTarget;
    column.classList.remove('drag-over');

    const taskId = event.dataTransfer.getData('text/plain');
    const newStatus = column.dataset.status;
    const taskCard = document.getElementById('task-' + taskId);

    if (!taskCard) return;

    // Get old status from the previous column
    const oldColumn = document.querySelector(`.column-cards .task-card#task-${taskId}`)?.closest('.column-cards');
    const oldStatus = oldColumn?.dataset.status;

    // If status changed, update via API
    if (oldStatus && oldStatus !== newStatus) {
        updateTaskStatus(taskId, newStatus, taskCard);
    }
}

/**
 * Find the element after which to insert the dragged element
 * @param {HTMLElement} container - The column cards container
 * @param {number} y - Mouse Y position
 * @returns {HTMLElement|null} - Element to insert before, or null to append
 */
function getDragAfterElement(container, y) {
    const draggableElements = [
        ...container.querySelectorAll('.task-card:not(.dragging)')
    ];

    return draggableElements.reduce((closest, child) => {
        const box = child.getBoundingClientRect();
        const offset = y - box.top - box.height / 2;

        if (offset < 0 && offset > closest.offset) {
            return { offset: offset, element: child };
        } else {
            return closest;
        }
    }, { offset: Number.NEGATIVE_INFINITY }).element;
}

/**
 * Update task status via HTMX AJAX call
 * @param {string} taskId - Task ID
 * @param {string} newStatus - New status value
 * @param {HTMLElement} taskCard - Task card element
 */
function updateTaskStatus(taskId, newStatus, taskCard) {
    // Show loading state
    taskCard.style.opacity = '0.5';
    taskCard.style.pointerEvents = 'none';

    // Update via HTMX
    htmx.ajax('PUT', '/api/v1/tasks/' + taskId + '/status', {
        values: { status: newStatus },
        target: '#task-' + taskId,
        swap: 'outerHTML'
    }).then(function() {
        // Update column counts after successful update
        updateColumnCounts();
    }).catch(function(err) {
        console.error('Failed to update task status:', err);
        // Revert visual state on error
        taskCard.style.opacity = '1';
        taskCard.style.pointerEvents = '';

        // Show error notification if toast system exists
        if (typeof showToast === 'function') {
            showToast('Failed to update task status', 'error');
        }
    });
}

/**
 * Update column count badges after card movement
 */
function updateColumnCounts() {
    document.querySelectorAll('.board-column').forEach(column => {
        const count = column.querySelectorAll('.task-card').length;
        const countEl = column.querySelector('.column-count');
        if (countEl) {
            countEl.textContent = count;
        }
    });

    // Update total task count in header
    const totalCount = document.querySelectorAll('.task-card').length;
    const totalCountEl = document.querySelector('.task-count');
    if (totalCountEl) {
        const taskWord = totalCount === 1 ? 'task' : 'tasks';
        totalCountEl.textContent = totalCount + ' ' + taskWord;
    }
}

/**
 * Handle real-time task updates via WebSocket
 */
document.body.addEventListener('task.updated', function(evt) {
    const data = evt.detail;

    // If status changed, move the card to the new column
    if (data.changes && data.changes.status) {
        const taskCard = document.getElementById('task-' + data.task_id);
        if (taskCard) {
            const newColumn = document.querySelector(
                `.column-cards[data-status="${data.changes.status.new}"]`
            );
            if (newColumn) {
                // Move card to new column
                const loadMore = newColumn.querySelector('.load-more');
                if (loadMore) {
                    newColumn.insertBefore(taskCard, loadMore);
                } else {
                    newColumn.appendChild(taskCard);
                }
                updateColumnCounts();
            }
        }
    }

    // Refresh task card if other fields changed (not just status)
    if (data.changes && !data.changes.status) {
        htmx.ajax('GET', '/partials/tasks/' + data.task_id + '/card', {
            target: '#task-' + data.task_id,
            swap: 'outerHTML'
        });
    }
});

/**
 * Handle task created event
 */
document.body.addEventListener('task.created', function(evt) {
    const data = evt.detail;

    // Add new task card to the appropriate column (usually "todo")
    const column = document.querySelector('.column-cards[data-status="todo"]');
    if (column) {
        htmx.ajax('GET', '/partials/tasks/' + data.task_id + '/card', {
            target: column,
            swap: 'afterbegin'
        }).then(function() {
            updateColumnCounts();
        });
    }
});

/**
 * Handle task deleted event
 */
document.body.addEventListener('task.deleted', function(evt) {
    const data = evt.detail;

    const taskCard = document.getElementById('task-' + data.task_id);
    if (taskCard) {
        taskCard.remove();
        updateColumnCounts();
    }
});

/**
 * Initialize board on page load
 */
document.addEventListener('DOMContentLoaded', function() {
    // Add keyboard accessibility for task cards
    document.querySelectorAll('.task-card').forEach(card => {
        card.setAttribute('tabindex', '0');
        card.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                card.click();
            }
        });
    });
});

/**
 * Re-initialize after HTMX content swap
 */
document.body.addEventListener('htmx:afterSwap', function(evt) {
    // Re-add keyboard accessibility to new cards
    evt.detail.target.querySelectorAll('.task-card').forEach(card => {
        card.setAttribute('tabindex', '0');
        card.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' || e.key === ' ') {
                e.preventDefault();
                card.click();
            }
        });
    });
});
