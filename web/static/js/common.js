// Common utilities shared across all pages

// DOM utilities
const $ = (selector) => document.querySelector(selector);
const $$ = (selector) => document.querySelectorAll(selector);

// XSS protection - escape HTML entities in user content
function escapeHtml(str) {
    if (str === null || str === undefined) return '';
    return String(str).replace(/[&<>"']/g, c => ({
        '&': '&amp;',
        '<': '&lt;',
        '>': '&gt;',
        '"': '&quot;',
        "'": '&#39;'
    }[c]));
}

// Shorthand alias
const esc = escapeHtml;

// Escape JavaScript strings (for use in onclick attributes, etc.)
function escapeJs(str) {
    if (str === null || str === undefined) return '';
    return String(str)
        .replace(/\\/g, '\\\\')  // Backslash first
        .replace(/'/g, "\\'")    // Single quotes
        .replace(/"/g, '\\"')    // Double quotes
        .replace(/\n/g, '\\n')   // Newlines
        .replace(/\r/g, '\\r');  // Carriage returns
}

// Format utilities
function formatTime(seconds) {
    const minutes = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${minutes}:${secs.toString().padStart(2, '0')}`;
}

// Toast notification system
const Toast = {
    container: null,

    init() {
        if (this.container) return;
        this.container = document.createElement('div');
        this.container.id = 'toast-container';
        this.container.className = 'fixed top-4 right-4 z-50 flex flex-col gap-2';
        document.body.appendChild(this.container);
    },

    show(message, type = 'info', duration = 3000) {
        this.init();

        const toast = document.createElement('div');
        const colors = {
            success: 'bg-green-500',
            error: 'bg-red-500',
            info: 'bg-blue-500',
            warning: 'bg-yellow-500'
        };

        toast.className = `${colors[type] || colors.info} text-white px-4 py-3 rounded-lg shadow-lg transform transition-all duration-300 translate-x-full opacity-0`;
        toast.textContent = message;

        this.container.appendChild(toast);

        // Animate in
        requestAnimationFrame(() => {
            toast.classList.remove('translate-x-full', 'opacity-0');
        });

        // Auto-remove
        setTimeout(() => {
            toast.classList.add('translate-x-full', 'opacity-0');
            setTimeout(() => toast.remove(), 300);
        }, duration);
    },

    success(message, duration) { this.show(message, 'success', duration); },
    error(message, duration) { this.show(message, 'error', duration); },
    info(message, duration) { this.show(message, 'info', duration); },
    warning(message, duration) { this.show(message, 'warning', duration); }
};

// Global notification function (for backward compatibility)
function showNotification(message, type = 'info') {
    Toast.show(message, type);
}

// APIError class to hold error code and message
class APIError extends Error {
    constructor(message, code, status, data) {
        super(message);
        this.code = code;
        this.status = status;
        this.name = 'APIError';
        // Store all additional error data fields
        if (data) {
            Object.assign(this, data);
        }
    }
}

// Simple API utilities (no auth required)
const API = {
    async handleResponse(response) {
        if (!response.ok) {
            const errorData = await response.json().catch(() => ({
                error: 'Request failed',
                code: 'UNKNOWN_ERROR'
            }));

            // Handle unauthorized errors - redirect to login
            if (response.status === 401 || errorData.code === 'UNAUTHORIZED') {
                window.location.href = '/admin/login';
                throw new APIError('Unauthorized - redirecting to login', errorData.code, response.status, errorData);
            }

            throw new APIError(
                errorData.error || errorData.message || `HTTP ${response.status}`,
                errorData.code || 'UNKNOWN_ERROR',
                response.status,
                errorData
            );
        }
        return response.json().catch(() => ({}));
    },

    async get(url) {
        const response = await fetch(url);
        return this.handleResponse(response);
    },

    async post(url, data) {
        const response = await fetch(url, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return this.handleResponse(response);
    },

    async put(url, data) {
        const response = await fetch(url, {
            method: 'PUT',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        return this.handleResponse(response);
    },

    async delete(url) {
        const response = await fetch(url, { method: 'DELETE' });
        return this.handleResponse(response);
    }
};

// Modal utilities
function showModal(modalId) {
    const modal = typeof modalId === 'string' ? document.getElementById(modalId) : modalId;
    if (modal) modal.classList.remove('hidden');
}

function hideModal(modalId) {
    const modal = typeof modalId === 'string' ? document.getElementById(modalId) : modalId;
    if (modal) modal.classList.add('hidden');
}

// Setup modal backdrop click to close
function setupModalBackdropClose(modalId, onClose) {
    const modal = document.getElementById(modalId);
    if (modal) {
        modal.addEventListener('click', (e) => {
            if (e.target === modal) {
                if (onClose) onClose();
                else hideModal(modal);
            }
        });
    }
}

// Loading state utilities
const Loading = {
    show(element) {
        if (typeof element === 'string') element = $(element);
        if (element) element.classList.add('opacity-50', 'pointer-events-none');
    },
    hide(element) {
        if (typeof element === 'string') element = $(element);
        if (element) element.classList.remove('opacity-50', 'pointer-events-none');
    }
};

// Styled confirm dialog (replaces native confirm())
const Confirm = {
    modalId: 'confirm-modal',
    resolveFunc: null,

    init() {
        if ($('#' + this.modalId)) return;

        const modal = document.createElement('div');
        modal.id = this.modalId;
        modal.className = 'hidden fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
        modal.innerHTML = `
            <div class="bg-white rounded-lg shadow-xl max-w-md w-full mx-4 overflow-hidden">
                <div class="p-6">
                    <h3 id="confirm-title" class="text-lg font-bold text-gray-900 mb-2">Confirm</h3>
                    <p id="confirm-message" class="text-gray-600"></p>
                </div>
                <div class="bg-gray-50 px-6 py-3 flex justify-end space-x-3">
                    <button id="confirm-cancel" class="px-4 py-2 text-gray-700 bg-gray-200 rounded hover:bg-gray-300">Cancel</button>
                    <button id="confirm-ok" class="px-4 py-2 text-white bg-red-600 rounded hover:bg-red-700">Confirm</button>
                </div>
            </div>
        `;
        document.body.appendChild(modal);

        // Event listeners
        $('#confirm-cancel').addEventListener('click', () => this.resolve(false));
        $('#confirm-ok').addEventListener('click', () => this.resolve(true));
        modal.addEventListener('click', (e) => {
            if (e.target === modal) this.resolve(false);
        });
    },

    show(message, title = 'Confirm', okText = 'Confirm', okClass = 'bg-red-600 hover:bg-red-700') {
        this.init();
        $('#confirm-title').textContent = title;
        $('#confirm-message').textContent = message;
        const okBtn = $('#confirm-ok');
        okBtn.textContent = okText;
        okBtn.className = `px-4 py-2 text-white rounded ${okClass}`;
        showModal(this.modalId);

        return new Promise(resolve => {
            this.resolveFunc = resolve;
        });
    },

    resolve(result) {
        hideModal(this.modalId);
        if (this.resolveFunc) {
            this.resolveFunc(result);
            this.resolveFunc = null;
        }
    },

    // Convenience methods
    async danger(message, title = 'Are you sure?') {
        return this.show(message, title, 'Delete', 'bg-red-600 hover:bg-red-700');
    },

    async warn(message, title = 'Warning') {
        return this.show(message, title, 'Continue', 'bg-yellow-600 hover:bg-yellow-700');
    }
};

// Debounce utility
function debounce(func, wait = 300) {
    let timeout;
    return function executedFunction(...args) {
        const later = () => {
            clearTimeout(timeout);
            func(...args);
        };
        clearTimeout(timeout);
        timeout = setTimeout(later, wait);
    };
}

// Form validation helper
function validateRequired(fields) {
    for (const [selector, name] of fields) {
        const el = typeof selector === 'string' ? $(selector) : selector;
        if (!el || !el.value.trim()) {
            Toast.warning(`${name} is required`);
            if (el) el.focus();
            return false;
        }
    }
    return true;
}

// Event delegation helper
function delegate(container, selector, event, handler) {
    const el = typeof container === 'string' ? $(container) : container;
    if (!el) return;

    el.addEventListener(event, (e) => {
        const target = e.target.closest(selector);
        if (target && el.contains(target)) {
            handler(e, target);
        }
    });
}

// Global keyboard support
document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
        // Close any visible modals
        $$('.fixed.inset-0:not(.hidden)').forEach(modal => {
            modal.classList.add('hidden');
        });
    }
});

// Global error handler for unhandled promise rejections
window.addEventListener('unhandledrejection', (event) => {
    console.error('Unhandled promise rejection:', event.reason);
    Toast.error('An unexpected error occurred');
    event.preventDefault(); // Prevents console error spam
});

// Focus management for modals
const FocusTrap = {
    previouslyFocused: null,

    trap(modalEl) {
        this.previouslyFocused = document.activeElement;

        const focusable = modalEl.querySelectorAll(
            'button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])'
        );

        if (focusable.length > 0) {
            focusable[0].focus();
        }

        modalEl._focusTrapHandler = (e) => {
            if (e.key !== 'Tab') return;

            const first = focusable[0];
            const last = focusable[focusable.length - 1];

            if (e.shiftKey && document.activeElement === first) {
                e.preventDefault();
                last.focus();
            } else if (!e.shiftKey && document.activeElement === last) {
                e.preventDefault();
                first.focus();
            }
        };

        modalEl.addEventListener('keydown', modalEl._focusTrapHandler);
    },

    release(modalEl) {
        if (modalEl._focusTrapHandler) {
            modalEl.removeEventListener('keydown', modalEl._focusTrapHandler);
            delete modalEl._focusTrapHandler;
        }

        if (this.previouslyFocused) {
            this.previouslyFocused.focus();
            this.previouslyFocused = null;
        }
    }
};
