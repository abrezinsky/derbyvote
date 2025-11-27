// Categories page functionality (uses common.js utilities)

let categories = [];
let groups = [];
let editingId = null;
let editingGroupId = null;
let voterTypes = [];
let ranks = [];

// ===== VOTER TYPES =====
async function loadVoterTypes() {
    try {
        const data = await API.get('/api/admin/voter-types');
        voterTypes = data.voter_types || ['general', 'racer', 'Race Committee', 'Cubmaster'];
        populateVoterTypesMultiselect();
    } catch (error) {
        console.error('Error loading voter types:', error);
        voterTypes = ['general', 'racer', 'Race Committee', 'Cubmaster'];
        populateVoterTypesMultiselect();
    }
}

function populateVoterTypesMultiselect() {
    const container = $('#voter-types-multiselect');
    if (!container) return;

    container.innerHTML = voterTypes.map(type => `
        <label class="flex items-center cursor-pointer hover:bg-gray-50 p-2 rounded">
            <input type="checkbox" value="${esc(type)}" class="voter-type-checkbox mr-3">
            <span class="text-sm text-gray-700">${esc(type)}</span>
        </label>
    `).join('');
}

// ===== RANKS =====
async function loadRanks() {
    try {
        const cars = await API.get('/api/admin/cars');
        // Extract unique ranks from cars
        const uniqueRanks = [...new Set(cars.map(c => c.rank).filter(r => r))].sort();
        ranks = uniqueRanks;
        populateRanksMultiselect();
    } catch (error) {
        console.error('Error loading ranks:', error);
        ranks = [];
        populateRanksMultiselect();
    }
}

function populateRanksMultiselect() {
    const container = $('#ranks-multiselect');
    if (!container) return;

    if (ranks.length === 0) {
        container.innerHTML = '<p class="text-sm text-gray-500 p-2">No classes found. Add classes to cars first.</p>';
        return;
    }

    container.innerHTML = ranks.map(rank => `
        <label class="flex items-center cursor-pointer hover:bg-gray-50 p-2 rounded">
            <input type="checkbox" value="${esc(rank)}" class="rank-checkbox mr-3">
            <span class="text-sm text-gray-700">${esc(rank)}</span>
        </label>
    `).join('');
}

// ===== GROUP MANAGEMENT =====
async function loadGroups() {
    try {
        groups = await API.get('/api/admin/category-groups') || [];
        renderGroups();
        updateGroupDropdown();
    } catch (error) {
        console.error('Error loading groups:', error);
        groups = [];
    }
}

function renderGroups() {
    const container = $('#groups-list');

    if (groups.length === 0) {
        container.innerHTML = '<div class="bg-white rounded-lg shadow p-6 text-center text-gray-600">No groups yet. Groups help organize categories and enforce exclusivity rules.</div>';
        return;
    }

    container.innerHTML = groups.map(group => {
        let exclusivityText = 'No exclusivity';
        if (group.exclusivity_pool_id !== null) {
            exclusivityText = `Pool ${group.exclusivity_pool_id}`;
        }

        let maxWinsText = '';
        if (group.max_wins_per_car !== null && group.max_wins_per_car !== undefined) {
            maxWinsText = `<span class="inline-block bg-blue-200 text-blue-800 rounded px-2 py-1 mr-2">Max ${group.max_wins_per_car} win${group.max_wins_per_car > 1 ? 's' : ''}/car</span>`;
        }

        return `
        <div class="bg-white rounded-lg shadow p-4" data-group-id="${group.id}">
            <div class="flex items-center justify-between">
                <div class="flex-1">
                    <div class="font-semibold text-lg">${esc(group.name)}</div>
                    ${group.description ? `<div class="text-sm text-gray-600">${esc(group.description)}</div>` : ''}
                    <div class="text-xs text-gray-500 mt-1">
                        <span class="inline-block bg-gray-200 rounded px-2 py-1 mr-2">Order: ${group.display_order}</span>
                        <span class="inline-block ${group.exclusivity_pool_id ? 'bg-purple-200 text-purple-800' : 'bg-gray-200'} rounded px-2 py-1">${exclusivityText}</span>
                        ${maxWinsText}
                    </div>
                </div>
                <div class="flex space-x-2">
                    <button data-action="edit-group" class="px-4 py-2 bg-purple-600 text-white rounded hover:bg-purple-700">
                        Edit
                    </button>
                    <button data-action="delete-group" class="px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700">
                        Delete
                    </button>
                </div>
            </div>
        </div>
        `;
    }).join('');
}

function updateGroupDropdown() {
    const select = $('#category-group');
    const currentValue = select.value;

    select.innerHTML = '<option value="">None (Independent)</option>';
    groups.forEach(group => {
        const option = document.createElement('option');
        option.value = group.id;
        // textContent is safe, no need to escape
        option.textContent = group.name;
        if (group.exclusivity_pool_id) {
            option.textContent += ` (Pool ${group.exclusivity_pool_id})`;
        }
        select.appendChild(option);
    });

    // Restore previous selection if editing
    if (currentValue) {
        select.value = currentValue;
    }
}

function showGroupModal(title = 'Add Category Group', id = null) {
    editingGroupId = id;
    $('#group-modal-title').textContent = title;

    if (id) {
        const group = groups.find(g => g.id === id);
        $('#group-name').value = group.name;
        $('#group-description').value = group.description || '';
        $('#group-order').value = group.display_order;
        $('#group-max-wins').value = group.max_wins_per_car || '';

        // Handle exclusivity pool
        if (group.exclusivity_pool_id === null) {
            $('#group-exclusivity').value = '';
        } else if (group.exclusivity_pool_id >= 1000) {
            // Auto-generated pool for "Just This Group"
            $('#group-exclusivity').value = 'auto';
        } else {
            $('#group-exclusivity').value = group.exclusivity_pool_id;
        }
    } else {
        $('#group-name').value = '';
        $('#group-description').value = '';
        $('#group-order').value = (groups && groups.length ? groups.length : 0) + 1;
        $('#group-exclusivity').value = '';
        $('#group-max-wins').value = '';
    }

    showModal('group-modal');
}

function hideGroupModal() {
    hideModal('group-modal');
}

function editGroup(id) {
    showGroupModal('Edit Category Group', id);
}

async function deleteGroup(id) {
    const confirmed = await Confirm.danger(
        'Categories in this group will become independent.',
        'Delete this group?'
    );
    if (!confirmed) return;

    try {
        await API.delete(`/api/admin/category-groups/${id}`);
        await loadGroups();
        await loadCategories();
        Toast.success('Group deleted');
    } catch (error) {
        console.error('Error deleting group:', error);
        Toast.error('Failed to delete group');
    }
}

async function saveGroup() {
    if (!validateRequired([['#group-name', 'Group name']])) return;

    const exclusivityValue = $('#group-exclusivity').value;
    const displayOrder = parseInt($('#group-order').value);
    const maxWinsValue = $('#group-max-wins').value;

    // Handle exclusivity pool ID
    let exclusivityPoolId = null;
    if (exclusivityValue === 'auto') {
        const maxPoolId = Math.max(0, ...groups.map(g => g.exclusivity_pool_id || 0));
        exclusivityPoolId = Math.max(1000, maxPoolId + 1);
    } else if (exclusivityValue !== '') {
        exclusivityPoolId = parseInt(exclusivityValue);
    }

    // Handle max wins per car
    let maxWinsPerCar = null;
    if (maxWinsValue && maxWinsValue !== '') {
        maxWinsPerCar = parseInt(maxWinsValue);
    }

    const data = {
        name: $('#group-name').value,
        description: $('#group-description').value || null,
        exclusivity_pool_id: exclusivityPoolId,
        max_wins_per_car: maxWinsPerCar,
        display_order: displayOrder
    };

    const saveBtn = $('#group-modal-save');
    Loading.show(saveBtn);

    try {
        if (editingGroupId) {
            await API.put(`/api/admin/category-groups/${editingGroupId}`, data);
            Toast.success('Group updated');
        } else {
            await API.post('/api/admin/category-groups', data);
            Toast.success('Group created');
        }
        hideGroupModal();
        await loadGroups();
        await loadCategories();
    } catch (error) {
        console.error('Error saving group:', error);
        Toast.error('Failed to save group');
    } finally {
        Loading.hide(saveBtn);
    }
}

// ===== CATEGORY MANAGEMENT =====
async function loadCategories() {
    Loading.show('#categories-list');
    try {
        categories = await API.get('/api/admin/categories');
        renderCategories();
    } catch (error) {
        console.error('Error loading categories:', error);
        const container = $('#categories-list');
        container.innerHTML = `
            <div class="bg-white rounded-lg shadow p-8 text-center text-red-600">
                Failed to load categories. ${error.message || 'Please try refreshing the page.'}
            </div>
        `;
    } finally {
        Loading.hide('#categories-list');
    }
}

function renderCategories() {
    const container = $('#categories-list');

    if (categories.length === 0) {
        container.innerHTML = '<div class="bg-white rounded-lg shadow p-8 text-center text-gray-600">No categories yet</div>';
        return;
    }

    container.innerHTML = categories.map(cat => {
        let groupBadge = '';
        if (cat.group_name) {
            groupBadge = `<span class="inline-block bg-purple-100 text-purple-800 text-xs rounded px-2 py-1 mr-2">${esc(cat.group_name)}</span>`;
        }

        let voterTypesBadges = '';
        if (cat.allowed_voter_types && cat.allowed_voter_types.length > 0) {
            // Show each voter type as a separate badge
            voterTypesBadges = cat.allowed_voter_types.map(type =>
                `<span class="inline-block bg-blue-100 text-blue-800 text-xs rounded px-2 py-1 mr-1">${esc(type)}</span>`
            ).join('');
        } else {
            // Show "All Voters" when no restrictions
            voterTypesBadges = '<span class="inline-block bg-gray-100 text-gray-600 text-xs rounded px-2 py-1 mr-1">All Voters</span>';
        }

        let ranksBadges = '';
        if (cat.allowed_ranks && cat.allowed_ranks.length > 0) {
            // Show each rank as a separate badge
            ranksBadges = cat.allowed_ranks.map(rank =>
                `<span class="inline-block bg-green-100 text-green-800 text-xs rounded px-2 py-1 mr-1">${esc(rank)}</span>`
            ).join('');
        } else {
            // Show "All Classes" when no restrictions
            ranksBadges = '<span class="inline-block bg-gray-100 text-gray-600 text-xs rounded px-2 py-1 mr-1">All Classes</span>';
        }

        return `
        <div class="bg-white rounded-lg shadow p-4 flex items-center justify-between" data-category-id="${cat.id}">
            <div class="flex-1">
                <div class="font-semibold text-lg">${esc(cat.name)}</div>
                <div class="text-sm text-gray-600">
                    ${groupBadge}
                    ${voterTypesBadges}
                    ${ranksBadges}
                    <span class="text-gray-500">Order: ${cat.display_order}</span>
                    ${cat.active ? '' : '<span class="text-red-600">(Inactive)</span>'}
                </div>
            </div>
            <div class="flex space-x-2">
                <button data-action="edit" class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700">
                    Edit
                </button>
                <button data-action="toggle" data-active="${cat.active}" class="px-4 py-2 ${cat.active ? 'bg-yellow-600' : 'bg-green-600'} text-white rounded hover:opacity-80">
                    ${cat.active ? 'Deactivate' : 'Activate'}
                </button>
            </div>
        </div>
        `;
    }).join('');
}

function showCategoryModal(title = 'Add Category', id = null) {
    editingId = id;
    $('#modal-title').textContent = title;

    if (id) {
        const cat = categories.find(c => c.id === id);
        $('#category-name').value = cat.name;
        $('#category-order').value = cat.display_order;
        $('#category-group').value = cat.group_id || '';

        // Set voter type checkboxes
        const allowedTypes = cat.allowed_voter_types || [];
        document.querySelectorAll('.voter-type-checkbox').forEach(checkbox => {
            checkbox.checked = allowedTypes.includes(checkbox.value);
        });

        // Set rank checkboxes
        const allowedRanks = cat.allowed_ranks || [];
        document.querySelectorAll('.rank-checkbox').forEach(checkbox => {
            checkbox.checked = allowedRanks.includes(checkbox.value);
        });
    } else {
        $('#category-name').value = '';
        $('#category-order').value = categories.length + 1;
        $('#category-group').value = '';

        // Clear all voter type checkboxes for new category
        document.querySelectorAll('.voter-type-checkbox').forEach(checkbox => {
            checkbox.checked = false;
        });

        // Clear all rank checkboxes for new category
        document.querySelectorAll('.rank-checkbox').forEach(checkbox => {
            checkbox.checked = false;
        });
    }

    showModal('category-modal');
}

function hideCategoryModal() {
    hideModal('category-modal');
    editingId = null;
}

function editCategory(id) {
    showCategoryModal('Edit Category', id);
}

async function toggleCategory(id, active) {
    const cat = categories.find(c => c.id === id);
    try {
        await API.put(`/api/admin/categories/${id}`, {
            name: cat.name,
            display_order: cat.display_order,
            group_id: cat.group_id || null,
            active: active,
            allowed_voter_types: cat.allowed_voter_types || null,
            allowed_ranks: cat.allowed_ranks || null
        });
        loadCategories();
        Toast.success(active ? 'Category activated' : 'Category deactivated');
    } catch (error) {
        console.error('Error toggling category:', error);
        Toast.error('Failed to update category');
    }
}

async function saveCategory() {
    if (!validateRequired([['#category-name', 'Category name']])) return;

    const order = parseInt($('#category-order').value);
    const groupValue = $('#category-group').value;
    const groupId = groupValue ? parseInt(groupValue) : null;

    // Collect selected voter types
    const selectedVoterTypes = Array.from(document.querySelectorAll('.voter-type-checkbox:checked'))
        .map(checkbox => checkbox.value);

    // Collect selected ranks
    const selectedRanks = Array.from(document.querySelectorAll('.rank-checkbox:checked'))
        .map(checkbox => checkbox.value);

    const saveBtn = $('#modal-save');
    Loading.show(saveBtn);

    try {
        if (editingId) {
            const cat = categories.find(c => c.id === editingId);
            await API.put(`/api/admin/categories/${editingId}`, {
                name: $('#category-name').value.trim(),
                display_order: order,
                group_id: groupId,
                active: cat.active,
                allowed_voter_types: selectedVoterTypes.length > 0 ? selectedVoterTypes : null,
                allowed_ranks: selectedRanks.length > 0 ? selectedRanks : null
            });
            Toast.success('Category updated');
        } else {
            await API.post('/api/admin/categories', {
                name: $('#category-name').value.trim(),
                display_order: order,
                group_id: groupId,
                allowed_voter_types: selectedVoterTypes.length > 0 ? selectedVoterTypes : null,
                allowed_ranks: selectedRanks.length > 0 ? selectedRanks : null
            });
            Toast.success('Category created');
        }
        hideCategoryModal();
        loadCategories();
    } catch (error) {
        console.error('Error saving category:', error);
        Toast.error('Failed to save category');
    } finally {
        Loading.hide(saveBtn);
    }
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    // Group event listeners
    $('#add-group').addEventListener('click', () => showGroupModal());
    $('#group-modal-cancel').addEventListener('click', hideGroupModal);
    $('#group-modal-save').addEventListener('click', saveGroup);

    // Category event listeners
    $('#add-category').addEventListener('click', () => showCategoryModal());
    $('#modal-cancel').addEventListener('click', hideCategoryModal);
    $('#modal-save').addEventListener('click', saveCategory);
    $('#clear-voter-types').addEventListener('click', () => {
        document.querySelectorAll('.voter-type-checkbox').forEach(cb => cb.checked = false);
    });
    $('#clear-ranks').addEventListener('click', () => {
        document.querySelectorAll('.rank-checkbox').forEach(cb => cb.checked = false);
    });

    // Sync from DerbyNet
    $('#sync-derbynet').addEventListener('click', syncFromDerbyNet);

    // Modal backdrop close
    setupModalBackdropClose('group-modal', hideGroupModal);
    setupModalBackdropClose('category-modal', hideCategoryModal);

    // Event delegation for groups
    delegate('#groups-list', '[data-action]', 'click', (e, target) => {
        const groupId = parseInt(target.closest('[data-group-id]').dataset.groupId);
        const action = target.dataset.action;

        if (action === 'edit-group') {
            editGroup(groupId);
        } else if (action === 'delete-group') {
            deleteGroup(groupId);
        }
    });

    // Event delegation for categories
    delegate('#categories-list', '[data-action]', 'click', (e, target) => {
        const categoryId = parseInt(target.closest('[data-category-id]').dataset.categoryId);
        const action = target.dataset.action;
        const cat = categories.find(c => c.id === categoryId);

        if (action === 'edit') {
            editCategory(categoryId);
        } else if (action === 'toggle') {
            toggleCategory(categoryId, !cat.active);
        }
    });

    // Initial load
    async function init() {
        await loadVoterTypes();
        await loadRanks();
        await loadGroups();
        await loadCategories();
    }
    init();
});

// ===== DERBYNET SYNC =====
function showSyncStatus(message, isError = false) {
    const statusEl = $('#sync-status');
    const messageEl = $('#sync-message');
    messageEl.innerHTML = message;
    messageEl.className = isError ? 'text-sm text-red-600' : 'text-sm text-blue-600';
    statusEl.classList.remove('hidden');

    if (!isError && !message.includes('Syncing')) {
        setTimeout(() => {
            statusEl.classList.add('hidden');
        }, 5000);
    }
}

async function syncFromDerbyNet() {
    const syncBtn = $('#sync-derbynet');
    Loading.show(syncBtn);
    showSyncStatus('Fetching settings...');

    try {
        const settings = await API.get('/api/admin/settings');

        if (!settings.derbynet_url) {
            showSyncStatus('DerbyNet URL not configured. Please set it in Settings first.', true);
            return;
        }

        showSyncStatus('Syncing categories with DerbyNet...');

        const result = await API.post('/api/admin/sync-categories-derbynet', {derbynet_url: settings.derbynet_url});

        let message = `Synced! Categories: ${result.categories_created} created, ${result.categories_updated} updated.`;
        if (result.awards_created > 0) {
            message += ` Awards pushed to DerbyNet: ${result.awards_created}.`;
        }
        if (result.auth_error) {
            message += `<br><strong>WARNING:</strong> DerbyNet auth failed - ${result.auth_error}`;
            showSyncStatus(message, true);
        } else {
            showSyncStatus(message);
        }
        loadCategories();
    } catch (error) {
        console.error('Error syncing from DerbyNet:', error);
        showSyncStatus(`Error: ${error.message}`, true);
    } finally {
        Loading.hide(syncBtn);
    }
}
