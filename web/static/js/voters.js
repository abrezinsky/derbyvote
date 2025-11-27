// Voters page functionality (uses common.js utilities)

let voters = [];
let cars = [];
let editingVoter = null;
let voterTypes = [];

async function loadVoters() {
    Loading.show('#voters-table');
    try {
        voters = await API.get('/api/admin/voters') || [];
        renderVoters();
    } catch (error) {
        console.error('Error loading voters:', error);
        Toast.error('Failed to load voters');
    } finally {
        Loading.hide('#voters-table');
    }
}

async function loadCars() {
    try {
        cars = await API.get('/api/admin/cars') || [];
        populateCarDropdown();
    } catch (error) {
        console.error('Error loading cars:', error);
    }
}

async function loadVoterTypes() {
    try {
        const data = await API.get('/api/admin/voter-types');
        voterTypes = data.voter_types || ['general', 'racer', 'Race Committee', 'Cubmaster'];
        populateVoterTypeDropdowns();
    } catch (error) {
        console.error('Error loading voter types:', error);
        voterTypes = ['general', 'racer', 'Race Committee', 'Cubmaster'];
        populateVoterTypeDropdowns();
    }
}

function populateVoterTypeDropdowns() {
    // Populate the filter dropdown
    const filterSelect = $('#filter-type');
    if (filterSelect) {
        const currentValue = filterSelect.value;
        filterSelect.innerHTML = '<option value="">All Types</option>' +
            voterTypes.map(type => `<option value="${esc(type)}">${esc(type)}</option>`).join('');
        filterSelect.value = currentValue;
    }

    // Populate the modal voter type dropdown
    const voterTypeSelect = $('#voter-type');
    if (voterTypeSelect) {
        const currentValue = voterTypeSelect.value;
        voterTypeSelect.innerHTML = voterTypes.map(type =>
            `<option value="${esc(type)}">${esc(type)}</option>`
        ).join('');
        if (currentValue && voterTypes.includes(currentValue)) {
            voterTypeSelect.value = currentValue;
        }
    }
}

function populateCarDropdown() {
    const select = $('#voter-car');
    // Keep the first "None" option and clear the rest
    select.innerHTML = '<option value="">None</option>';

    cars.forEach(car => {
        const option = document.createElement('option');
        option.value = car.id;
        option.textContent = `#${car.car_number} - ${car.racer_name || ''} (${car.car_name || 'Unnamed'})`;
        select.appendChild(option);
    });
}

function renderVoters() {
    const filterType = $('#filter-type').value;
    const filteredVoters = filterType ? voters.filter(v => v.voter_type === filterType) : voters;

    const tbody = $('#voters-table');

    if (filteredVoters.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="px-6 py-8 text-center text-gray-600">No voters found. Add voters or sync from DerbyNet.</td></tr>';
        return;
    }

    tbody.innerHTML = filteredVoters.map(voter => `
        <tr data-voter-id="${voter.id}">
            <td class="px-6 py-4 whitespace-nowrap">
                <button data-action="qr" class="text-blue-600 hover:text-blue-800 font-mono">
                    ${esc(voter.qr_code)}
                </button>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">${esc(voter.name) || '<span class="text-gray-400">Not set</span>'}</td>
            <td class="px-6 py-4 whitespace-nowrap">
                <span class="px-2 py-1 text-xs rounded-full ${getTypeColor(voter.voter_type)}">
                    ${capitalizeFirst(voter.voter_type || 'general')}
                </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
                ${voter.car_number ? `#${esc(voter.car_number)} - ${esc(voter.racer_name) || ''}` : '<span class="text-gray-400">None</span>'}
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
                ${voter.has_voted ? '<span class="text-green-600">Voted</span>' : '<span class="text-gray-400">Not voted</span>'}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm">
                <button data-action="edit" class="text-blue-600 hover:text-blue-800 mr-3">Edit</button>
                <button data-action="delete" class="text-red-600 hover:text-red-800">Delete</button>
            </td>
        </tr>
    `).join('');
}

function getTypeColor(type) {
    const colors = {
        'racer': 'bg-blue-100 text-blue-800',
        'general': 'bg-gray-100 text-gray-800',
        'committee': 'bg-purple-100 text-purple-800',
        'staff': 'bg-green-100 text-green-800'
    };
    return colors[type] || colors['general'];
}

function capitalizeFirst(str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
}

function openQRModal(voterId, qrCode) {
    $('#modal-qr-title').textContent = qrCode;
    $('#modal-qr-image').src = `/api/admin/voters/${voterId}/qr`;
    showModal('qr-modal');
}

function closeQRModal() {
    hideModal('qr-modal');
}

function showVoterModal(title = 'Add Voter', voter = null) {
    editingVoter = voter;
    $('#modal-title').textContent = title;

    if (voter) {
        $('#voter-name').value = voter.name || '';
        $('#voter-email').value = voter.email || '';
        $('#voter-type').value = voter.voter_type || 'general';
        $('#voter-car').value = voter.car_id || '';
        $('#voter-notes').value = voter.notes || '';
        $('#voter-qr-code').textContent = voter.qr_code;
        $('#voter-qr-image').src = `/api/admin/voters/${voter.id}/qr`;
        $('#qr-code-display').classList.remove('hidden');
    } else {
        $('#voter-name').value = '';
        $('#voter-email').value = '';
        $('#voter-type').value = 'general';
        $('#voter-car').value = '';
        $('#voter-notes').value = '';
        $('#qr-code-display').classList.add('hidden');
    }

    showModal('voter-modal');
}

function hideVoterModal() {
    hideModal('voter-modal');
    editingVoter = null;
}

function editVoter(id) {
    const voter = voters.find(v => v.id === id);
    if (voter) {
        showVoterModal('Edit Voter', voter);
    }
}

async function deleteVoter(id) {
    const voter = voters.find(v => v.id === id);
    const confirmed = await Confirm.danger(
        `This will permanently delete voter "${voter.qr_code}" and ALL of their votes.`,
        'Delete Voter?'
    );
    if (!confirmed) return;

    try {
        await API.delete(`/api/admin/voters/${id}`);
        await loadVoters();
        Toast.success('Voter deleted');
    } catch (error) {
        console.error('Error deleting voter:', error);
        Toast.error('Failed to delete voter');
    }
}

async function saveVoter() {
    const data = {
        name: $('#voter-name').value,
        email: $('#voter-email').value,
        voter_type: $('#voter-type').value,
        car_id: $('#voter-car').value ? parseInt($('#voter-car').value) : null,
        notes: $('#voter-notes').value
    };

    const saveBtn = $('#modal-save');
    Loading.show(saveBtn);

    try {
        if (editingVoter) {
            data.id = editingVoter.id;
            await API.put('/api/admin/voters', data);
            Toast.success('Voter updated');
        } else {
            await API.post('/api/admin/voters', data);
            Toast.success('Voter created');
        }
        hideVoterModal();
        await loadVoters();
    } catch (error) {
        console.error('Error saving voter:', error);
        Toast.error(error.message || 'Failed to save voter');
    } finally {
        Loading.hide(saveBtn);
    }
}

async function printQRCodes() {
    if (voters.length === 0) {
        Toast.warning('No voters to print');
        return;
    }

    const grid = $('#qr-grid');
    grid.innerHTML = voters.map(voter => `
        <div class="qr-card">
            <img src="/api/admin/voters/${voter.id}/qr" alt="${esc(voter.qr_code)}">
            <div class="font-bold text-sm mt-2">${esc(voter.qr_code)}</div>
            <div class="text-xs text-gray-600">${esc(voter.name) || ''}</div>
        </div>
    `).join('');

    // Wait for all QR images to load before printing
    const images = grid.querySelectorAll('img');
    const loadPromises = Array.from(images).map(img => {
        if (img.complete) return Promise.resolve();
        return new Promise((resolve, reject) => {
            img.onload = resolve;
            img.onerror = reject;
        });
    });

    try {
        Toast.info('Loading QR codes...');
        await Promise.all(loadPromises);
        window.print();
    } catch (error) {
        console.error('Error loading QR images:', error);
        Toast.error('Failed to load some QR codes');
    }
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    $('#add-voter').addEventListener('click', () => showVoterModal());
    $('#modal-cancel').addEventListener('click', hideVoterModal);
    $('#modal-save').addEventListener('click', saveVoter);
    $('#filter-type').addEventListener('change', renderVoters);
    $('#export-qr').addEventListener('click', printQRCodes);

    // Close QR modal on backdrop click
    setupModalBackdropClose('qr-modal', closeQRModal);
    setupModalBackdropClose('voter-modal', hideVoterModal);

    // Event delegation for voter table actions
    delegate('#voters-table', '[data-action]', 'click', (e, target) => {
        const row = target.closest('[data-voter-id]');
        const voterId = parseInt(row.dataset.voterId);
        const action = target.dataset.action;
        const voter = voters.find(v => v.id === voterId);

        if (action === 'qr') {
            openQRModal(voterId, voter.qr_code);
        } else if (action === 'edit') {
            editVoter(voterId);
        } else if (action === 'delete') {
            deleteVoter(voterId);
        }
    });

    loadVoters();
    loadCars();
    loadVoterTypes();
});
