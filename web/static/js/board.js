/**
 * Kanban Board Drag and Drop
 * Implements HTML5 native drag-and-drop for task cards
 */

(function () {
  "use strict";

  // Prevent double-initialization (can happen with HTMX boost).
  if (window.boardJsInitialized) {
    return;
  }
  window.boardJsInitialized = true;

  // State - stored on window to survive re-initialization
  var draggedTask = null;
  var draggedTaskOriginalStatus = null;

  /**
   * Handle drag start event
   * @param {DragEvent} event
   */
  function handleDragStart(event) {
    draggedTask = event.target;
    event.target.classList.add("dragging");

    // Save original status before the card moves
    var originalColumn = event.target.closest(".column-cards");
    draggedTaskOriginalStatus = originalColumn
      ? originalColumn.dataset.status
      : null;

    // Set data for the drag operation
    event.dataTransfer.effectAllowed = "move";
    event.dataTransfer.setData("text/plain", event.target.dataset.taskId);

    // Create a custom drag image
    var ghost = event.target.cloneNode(true);
    ghost.style.opacity = "0.8";
    ghost.style.position = "absolute";
    ghost.style.top = "-1000px";
    ghost.style.pointerEvents = "none";
    document.body.appendChild(ghost);
    event.dataTransfer.setDragImage(ghost, 0, 0);

    // Remove ghost after drag starts
    setTimeout(function () {
      ghost.remove();
    }, 0);
  }

  /**
   * Handle drag end event
   * @param {DragEvent} event
   */
  function handleDragEnd(event) {
    event.target.classList.remove("dragging");
    draggedTask = null;
    draggedTaskOriginalStatus = null;

    // Remove all drag-over states
    document
      .querySelectorAll(".column-cards.drag-over")
      .forEach(function (col) {
        col.classList.remove("drag-over");
      });
  }

  /**
   * Handle drag over event - determines where to insert the dragged element
   * @param {DragEvent} event
   */
  function handleDragOver(event) {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";

    var column = event.currentTarget;
    column.classList.add("drag-over");

    if (!draggedTask) return;

    // Find insertion point based on mouse position
    var afterElement = getDragAfterElement(column, event.clientY);

    if (afterElement) {
      column.insertBefore(draggedTask, afterElement);
    } else {
      // Insert before load-more button if exists, otherwise append
      var loadMore = column.querySelector(".load-more");
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
    var rect = event.currentTarget.getBoundingClientRect();
    if (
      event.clientX < rect.left ||
      event.clientX > rect.right ||
      event.clientY < rect.top ||
      event.clientY > rect.bottom
    ) {
      event.currentTarget.classList.remove("drag-over");
    }
  }

  /**
   * Handle drop event - updates task status via API
   * @param {DragEvent} event
   */
  function handleDrop(event) {
    event.preventDefault();

    var column = event.currentTarget;
    column.classList.remove("drag-over");

    var taskId = event.dataTransfer.getData("text/plain");
    var newStatus = column.dataset.status;
    var taskCard = document.getElementById("task-" + taskId);

    if (!taskCard) return;

    // Use the saved original status (card already moved during dragover)
    var oldStatus = draggedTaskOriginalStatus;

    // If status changed, update via API
    if (oldStatus && oldStatus !== newStatus) {
      updateTaskStatus(taskId, newStatus, taskCard, oldStatus);
    }
  }

  /**
   * Find the element after which to insert the dragged element
   * @param {HTMLElement} container - The column cards container
   * @param {number} y - Mouse Y position
   * @returns {HTMLElement|null} - Element to insert before, or null to append
   */
  function getDragAfterElement(container, y) {
    var draggableElements = Array.prototype.slice.call(
      container.querySelectorAll(".task-card:not(.dragging)"),
    );

    var result = draggableElements.reduce(
      function (closest, child) {
        var box = child.getBoundingClientRect();
        var offset = y - box.top - box.height / 2;

        if (offset < 0 && offset > closest.offset) {
          return { offset: offset, element: child };
        } else {
          return closest;
        }
      },
      { offset: Number.NEGATIVE_INFINITY },
    );

    return result.element;
  }

  /**
   * Update task status via HTMX AJAX call
   * @param {string} taskId - Task ID
   * @param {string} newStatus - New status value
   * @param {HTMLElement} taskCard - Task card element
   */
  function updateTaskStatus(taskId, newStatus, taskCard, oldStatus) {
    // Show loading state
    taskCard.style.opacity = "0.5";
    taskCard.style.pointerEvents = "none";

    // Extract workspace ID from current URL (/workspaces/{id}/board)
    var pathMatch = window.location.pathname.match(/\/workspaces\/([^/]+)/);
    var workspaceId = pathMatch ? pathMatch[1] : "";

    if (!workspaceId) {
      console.error("Could not determine workspace ID from URL");
      taskCard.style.opacity = "1";
      taskCard.style.pointerEvents = "";
      return;
    }

    // Update via fetch (no swap needed - card already moved by drag)
    fetch(
      "/api/v1/workspaces/" + workspaceId + "/tasks/" + taskId + "/status",
      {
        method: "PUT",
        headers: {
          "Content-Type": "application/x-www-form-urlencoded",
        },
        body: "status=" + encodeURIComponent(newStatus),
      },
    )
      .then(function (response) {
        if (!response.ok) {
          throw new Error(
            "Status update failed: " +
              response.status +
              " " +
              response.statusText,
          );
        }
        // Restore card visibility and update counts
        taskCard.style.opacity = "1";
        taskCard.style.pointerEvents = "";
        updateColumnCounts();
      })
      .catch(function (err) {
        console.error("Failed to update task status:", err);
        // Revert visual state on error
        taskCard.style.opacity = "1";
        taskCard.style.pointerEvents = "";

        // Revert card to original column if we know the old status
        if (oldStatus) {
          var originalColumn = document.querySelector(
            '.column-cards[data-status="' + oldStatus + '"]',
          );
          if (originalColumn && taskCard.parentElement !== originalColumn) {
            originalColumn.appendChild(taskCard);
            updateColumnCounts();
          }
        }

        // Show error notification if toast system exists
        if (typeof showToast === "function") {
          showToast("Failed to update task status", "error");
        }
      });
  }

  /**
   * Update column count badges after card movement
   */
  function updateColumnCounts() {
    document.querySelectorAll(".board-column").forEach(function (column) {
      var count = column.querySelectorAll(".task-card").length;
      var countEl = column.querySelector(".column-count");
      if (countEl) {
        countEl.textContent = count;
      }
    });

    // Update total task count in header
    var totalCount = document.querySelectorAll(".task-card").length;
    var totalCountEl = document.querySelector(".task-count");
    if (totalCountEl) {
      var taskWord = totalCount === 1 ? "task" : "tasks";
      totalCountEl.textContent = totalCount + " " + taskWord;
    }
  }

  /**
   * Add keyboard accessibility to task cards
   * @param {HTMLElement} container - Container to search for cards
   */
  function setupCardAccessibility(container) {
    var cards = container.querySelectorAll(".task-card:not([data-a11y-setup])");
    cards.forEach(function (card) {
      card.setAttribute("data-a11y-setup", "true");
      card.setAttribute("tabindex", "0");
      card.addEventListener("keydown", function (e) {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          card.click();
        }
      });
    });
  }

  // Expose handlers globally for inline event attributes
  window.handleDragStart = handleDragStart;
  window.handleDragEnd = handleDragEnd;
  window.handleDragOver = handleDragOver;
  window.handleDragLeave = handleDragLeave;
  window.handleDrop = handleDrop;

  /**
   * Handle real-time task updates via WebSocket
   */
  document.body.addEventListener("task.updated", function (evt) {
    var data = evt.detail;

    // If status changed, move the card to the new column
    if (data.changes && data.changes.status) {
      var taskCard = document.getElementById("task-" + data.task_id);
      if (taskCard) {
        var newColumn = document.querySelector(
          '.column-cards[data-status="' + data.changes.status.new + '"]',
        );
        if (newColumn) {
          // Move card to new column
          var loadMore = newColumn.querySelector(".load-more");
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
      htmx.ajax("GET", "/partials/tasks/" + data.task_id + "/card", {
        target: "#task-" + data.task_id,
        swap: "outerHTML",
      });
    }
  });

  /**
   * Handle task created event
   */
  document.body.addEventListener("task.created", function (evt) {
    var data = evt.detail;

    // Add new task card to the appropriate column (usually "todo")
    var column = document.querySelector('.column-cards[data-status="todo"]');
    if (column) {
      htmx
        .ajax("GET", "/partials/tasks/" + data.task_id + "/card", {
          target: column,
          swap: "afterbegin",
        })
        .then(function () {
          updateColumnCounts();
        });
    }
  });

  /**
   * Handle task deleted event
   */
  document.body.addEventListener("task.deleted", function (evt) {
    var data = evt.detail;

    var taskCard = document.getElementById("task-" + data.task_id);
    if (taskCard) {
      taskCard.remove();
      updateColumnCounts();
    }
  });

  /**
   * Initialize board on page load
   */
  document.addEventListener("DOMContentLoaded", function () {
    setupCardAccessibility(document);
  });

  /**
   * Re-initialize after HTMX content swap
   */
  document.body.addEventListener("htmx:afterSwap", function (evt) {
    setupCardAccessibility(evt.detail.target);
  });
})();
