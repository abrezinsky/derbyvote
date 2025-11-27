// Cars Management JavaScript (uses common.js utilities)

let editingCarId = null;
let deletingCarId = null;
let allCars = [];

document.addEventListener('DOMContentLoaded', function() {
    // Restore filter preferences
    const hideIneligible = localStorage.getItem('hideIneligibleCars') === 'true';
    $('#hide-ineligible').checked = hideIneligible;

    const rankFilter = localStorage.getItem('rankFilter') || '';
    $('#rank-filter').value = rankFilter;

    loadCars();

    // Add car button
    $('#add-car').addEventListener('click', () => openModal());

    // Sync from DerbyNet button
    $('#sync-derbynet').addEventListener('click', syncFromDerbyNet);

    // Hide ineligible filter
    $('#hide-ineligible').addEventListener('change', handleFilterChange);

    // Rank filter
    $('#rank-filter').addEventListener('change', handleRankFilterChange);

    // Modal buttons
    $('#modal-cancel').addEventListener('click', closeModal);
    $('#modal-save').addEventListener('click', saveCar);

    // Delete modal buttons
    $('#delete-cancel').addEventListener('click', closeDeleteModal);
    $('#delete-confirm').addEventListener('click', confirmDelete);

    // Close modals on background click
    setupModalBackdropClose('car-modal', closeModal);
    setupModalBackdropClose('delete-modal', closeDeleteModal);

    // Event delegation for car list actions
    delegate('#cars-list', '[data-action]', 'click', handleCarAction);
    delegate('#cars-list', 'input[type="checkbox"]', 'change', handleEligibilityToggle);
});

function handleFilterChange(e) {
    const hideIneligible = e.target.checked;
    localStorage.setItem('hideIneligibleCars', hideIneligible);
    renderCars(allCars);
}

function handleRankFilterChange(e) {
    const rankFilter = e.target.value;
    localStorage.setItem('rankFilter', rankFilter);
    renderCars(allCars);
}

async function handleCarAction(e, target) {
    const action = target.dataset.action;
    const carId = parseInt(target.closest('[data-car-id]').dataset.carId);

    if (action === 'edit') {
        editCar(carId);
    } else if (action === 'delete') {
        deleteCar(carId);
    }
}

async function handleEligibilityToggle(e, target) {
    const carId = parseInt(target.closest('[data-car-id]').dataset.carId);
    toggleEligibility(carId, target.checked);
}

async function loadCars() {
    Loading.show('#cars-list');
    try {
        allCars = await API.get('/api/admin/cars');
        renderCars(allCars);
    } catch (error) {
        console.error('Error loading cars:', error);
        const container = $('#cars-list');
        container.innerHTML = `
            <div class="col-span-full text-center py-8 text-red-600">
                Failed to load cars. ${error.message || 'Please try refreshing the page.'}
            </div>
        `;
    } finally {
        Loading.hide('#cars-list');
    }
}

function renderCars(cars) {
    const container = $('#cars-list');
    const hideIneligible = $('#hide-ineligible').checked;
    const rankFilter = $('#rank-filter').value;

    // Populate rank filter dropdown with unique ranks
    const uniqueRanks = [...new Set(cars.map(car => car.rank).filter(rank => rank))].sort();
    const rankFilterEl = $('#rank-filter');
    const currentValue = rankFilterEl.value;
    rankFilterEl.innerHTML = '<option value="">All classes</option>' +
        uniqueRanks.map(rank => `<option value="${esc(rank)}">${esc(rank)}</option>`).join('');
    rankFilterEl.value = currentValue;

    // Apply filters
    let filteredCars = cars;
    if (hideIneligible) {
        filteredCars = filteredCars.filter(car => car.eligible !== false);
    }
    if (rankFilter) {
        filteredCars = filteredCars.filter(car => car.rank === rankFilter);
    }

    if (!cars || cars.length === 0) {
        container.innerHTML = `
            <div class="col-span-full text-center py-8 text-gray-500">
                No cars yet. Click "+ Add Car" to create one.
            </div>
        `;
        return;
    }

    if (filteredCars.length === 0) {
        let message = 'No cars found';
        if (hideIneligible && rankFilter) {
            message = `No eligible cars found with rank "${esc(rankFilter)}"`;
        } else if (hideIneligible) {
            message = 'No eligible cars found. All cars are marked as ineligible.';
        } else if (rankFilter) {
            message = `No cars found with rank "${esc(rankFilter)}"`;
        }
        container.innerHTML = `
            <div class="col-span-full text-center py-8 text-gray-500">
                ${message}
            </div>
        `;
        return;
    }

    container.innerHTML = filteredCars.map(car => {
        const isEligible = car.eligible !== false;
        return `
        <div class="bg-white rounded-lg shadow p-4 ${!isEligible ? 'opacity-60' : ''}" data-car-id="${car.id}">
            <div class="flex items-start space-x-4">
                ${car.photo_url ? `
                    <img src="${esc(car.photo_url)}" alt="${esc(car.car_name)}"
                         class="w-20 h-auto rounded-lg flex-shrink-0 ${!isEligible ? 'grayscale' : ''}"
                         onerror="this.style.display='none'">
                ` : `
                    <div class="w-20 h-20 bg-gray-200 rounded-lg flex items-center justify-center flex-shrink-0">
                        <span class="text-2xl font-bold text-gray-400">${esc(car.car_number)}</span>
                    </div>
                `}
                <div class="flex-grow min-w-0">
                    <div class="flex items-center justify-between">
                        <h3 class="font-bold text-lg truncate">#${esc(car.car_number)}</h3>
                        <div class="flex space-x-2 flex-shrink-0">
                            <button data-action="edit" class="text-blue-600 hover:text-blue-800">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"/>
                                </svg>
                            </button>
                            <button data-action="delete" class="text-red-600 hover:text-red-800">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
                                </svg>
                            </button>
                        </div>
                    </div>
                    <p class="text-gray-600 truncate">${esc(car.racer_name) || 'No racer name'}</p>
                    <p class="text-sm text-gray-400 truncate">${esc(car.car_name) || 'No car name'}</p>
                    ${car.rank ? `<p class="text-xs text-blue-600 font-medium mt-1">${esc(car.rank)}</p>` : ''}
                    <div class="mt-2 flex items-center">
                        <label class="inline-flex items-center cursor-pointer">
                            <input type="checkbox" ${isEligible ? 'checked' : ''} class="sr-only peer">
                            <div class="relative w-9 h-5 bg-gray-200 peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-blue-300 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:bg-green-500"></div>
                            <span class="ml-2 text-sm ${isEligible ? 'text-green-600' : 'text-gray-500'}">${isEligible ? 'Eligible' : 'Ineligible'}</span>
                        </label>
                    </div>
                </div>
            </div>
        </div>
    `}).join('');
}

function openModal(car = null) {
    editingCarId = car ? car.id : null;
    $('#modal-title').textContent = car ? 'Edit Car' : 'Add Car';
    $('#car-number').value = car ? car.car_number : '';
    $('#racer-name').value = car ? car.racer_name : '';
    $('#car-name').value = car ? car.car_name : '';
    $('#rank').value = car ? car.rank : '';
    $('#photo-url').value = car ? car.photo_url : '';

    // Populate rank suggestions from existing cars
    const uniqueRanks = [...new Set(allCars.map(c => c.rank).filter(r => r))].sort();
    const datalist = $('#rank-suggestions');
    datalist.innerHTML = uniqueRanks.map(rank => `<option value="${esc(rank)}">`).join('');

    showModal('car-modal');
}

function closeModal() {
    hideModal('car-modal');
    editingCarId = null;
}

async function saveCar() {
    if (!validateRequired([['#car-number', 'Car number']])) return;

    const data = {
        car_number: $('#car-number').value.trim(),
        racer_name: $('#racer-name').value.trim(),
        car_name: $('#car-name').value.trim(),
        rank: $('#rank').value.trim(),
        photo_url: $('#photo-url').value.trim()
    };

    const saveBtn = $('#modal-save');
    Loading.show(saveBtn);

    try {
        if (editingCarId) {
            await API.put(`/api/admin/cars/${editingCarId}`, data);
        } else {
            await API.post('/api/admin/cars', data);
        }

        closeModal();
        loadCars();
        Toast.success(editingCarId ? 'Car updated' : 'Car created');
    } catch (error) {
        console.error('Error saving car:', error);
        Toast.error(error.message || 'Failed to save car');
    } finally {
        Loading.hide(saveBtn);
    }
}

async function editCar(id) {
    try {
        const car = await API.get(`/api/admin/cars/${id}`);
        openModal(car);
    } catch (error) {
        console.error('Error loading car:', error);
        Toast.error('Failed to load car');
    }
}

function deleteCar(id) {
    deletingCarId = id;
    showModal('delete-modal');
}

function closeDeleteModal() {
    hideModal('delete-modal');
    deletingCarId = null;
}

async function confirmDelete() {
    if (!deletingCarId) return;

    const confirmBtn = $('#delete-confirm');
    Loading.show(confirmBtn);

    try {
        await API.delete(`/api/admin/cars/${deletingCarId}`);
        closeDeleteModal();
        loadCars();
        Toast.success('Car deleted');
    } catch (error) {
        console.error('Error deleting car:', error);

        // Check if confirmation is required
        if (error.confirmation_required) {
            Loading.hide(confirmBtn);
            if (confirm(error.message)) {
                // Retry with force parameter
                Loading.show(confirmBtn);
                try {
                    await API.delete(`/api/admin/cars/${deletingCarId}?force=true`);
                    closeDeleteModal();
                    loadCars();
                    Toast.success('Car deleted');
                } catch (retryError) {
                    console.error('Error deleting car with force:', retryError);
                    Toast.error(retryError.message || 'Failed to delete car');
                } finally {
                    Loading.hide(confirmBtn);
                }
            }
        } else {
            Toast.error(error.message || 'Failed to delete car');
            Loading.hide(confirmBtn);
        }
    } finally {
        if (!error || !error.confirmation_required) {
            Loading.hide(confirmBtn);
        }
    }
}

async function toggleEligibility(carId, eligible) {
    try {
        await API.put(`/api/admin/cars/${carId}/eligibility`, { eligible });
        loadCars();
        Toast.success(eligible ? 'Car marked as eligible' : 'Car marked as ineligible');
    } catch (error) {
        console.error('Error updating eligibility:', error);

        // Check if confirmation is required
        if (error.confirmation_required) {
            if (confirm(error.message)) {
                // Retry with force parameter
                try {
                    await API.put(`/api/admin/cars/${carId}/eligibility`, { eligible, force: true });
                    loadCars();
                    Toast.success(eligible ? 'Car marked as eligible' : 'Car marked as ineligible');
                } catch (retryError) {
                    console.error('Error updating eligibility with force:', retryError);
                    Toast.error(retryError.message || 'Failed to update eligibility');
                    loadCars(); // Reload to reset the toggle state
                }
            } else {
                loadCars(); // Reload to reset the toggle state
            }
        } else {
            Toast.error(error.message || 'Failed to update eligibility');
            loadCars(); // Reload to reset the toggle state
        }
    }
}

function showSyncStatus(message, isError = false) {
    const statusEl = $('#sync-status');
    const messageEl = $('#sync-message');
    messageEl.innerHTML = message;
    messageEl.className = isError ? 'text-sm text-red-600' : 'text-sm text-blue-600';
    statusEl.classList.remove('hidden');

    // Auto-hide after 5 seconds for success messages
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

        showSyncStatus('Syncing from DerbyNet...');

        const result = await API.post('/api/admin/sync-derbynet', {
            derbynet_url: settings.derbynet_url
        });

        showSyncStatus(`Synced! Cars: ${result.cars_created} created, ${result.cars_updated} updated.`);
        loadCars();
    } catch (error) {
        console.error('Error syncing from DerbyNet:', error);
        showSyncStatus(`Error: ${error.message}`, true);
    } finally {
        Loading.hide(syncBtn);
    }
}
