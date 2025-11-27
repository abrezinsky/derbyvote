// Utility functions for DerbyVote

/**
 * Format sync result details by grouping errors and skipped items by message
 * @param {Object} result - Sync result with details array
 * @param {number} successCount - Number of successful items
 * @param {string} successLabel - Label for success items (e.g., "winners pushed")
 * @returns {string} HTML formatted message
 */
function formatSyncResultDetails(result, successCount, successLabel) {
    const errorMessages = new Map();
    const skippedMessages = new Map();

    if (result.details) {
        result.details.forEach(detail => {
            if (detail.status === 'error' && detail.message) {
                const count = errorMessages.get(detail.message) || 0;
                errorMessages.set(detail.message, count + 1);
            } else if (detail.status === 'skipped' && detail.message) {
                const count = skippedMessages.get(detail.message) || 0;
                skippedMessages.set(detail.message, count + 1);
            }
        });
    }

    let msg = successCount > 0 ? `${successCount} ${successLabel}.` : '';

    if (errorMessages.size > 0) {
        msg += '<br><strong>Errors:</strong><ul>';
        errorMessages.forEach((count, message) => {
            const countStr = count > 1 ? ` (×${count})` : '';
            msg += `<li>${message}${countStr}</li>`;
        });
        msg += '</ul>';
    }

    if (skippedMessages.size > 0) {
        msg += '<br><strong>Skipped:</strong><ul>';
        skippedMessages.forEach((count, message) => {
            const countStr = count > 1 ? ` (×${count})` : '';
            msg += `<li>${message}${countStr}</li>`;
        });
        msg += '</ul>';
    }

    return msg || 'No changes made.';
}
