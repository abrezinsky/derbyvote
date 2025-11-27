// Settings page functionality (uses common.js utilities)

// Load saved settings
let voterTypes = [];

async function loadSettings() {
    try {
        const settings = await API.get('/api/admin/settings');
        if (settings.derbynet_url) {
            $('#derbynet-url').value = settings.derbynet_url;
        }
        if (settings.base_url) {
            $('#base-url').value = settings.base_url;
        }
        if (settings.voting_instructions) {
            $('#voting-instructions').value = settings.voting_instructions;
        }
        if (settings.derbynet_role) {
            $('#derbynet-role').value = settings.derbynet_role;
        }
        $('#require-registered-qr').checked = settings.require_registered_qr === true;

        // Load voter types
        if (settings.voter_types) {
            voterTypes = settings.voter_types;
            renderVoterTypes();
        }

        // Update dynamic QR section visibility
        updateDynamicQRSection();
    } catch (error) {
        console.error('Error loading settings:', error);
        Toast.error('Failed to load settings');
    }
}

// Toggle Require Registered QR
async function toggleRequireRegisteredQR() {
    const checked = $('#require-registered-qr').checked;
    const messageEl = $('#require-qr-message');

    try {
        await API.post('/api/admin/settings', {require_registered_qr: checked});
        messageEl.textContent = checked ?
            'Enabled - Only registered QR codes can vote' :
            'Disabled - Any QR code can be used to vote';
        messageEl.className = 'mt-2 text-sm text-green-600';
        setTimeout(() => { messageEl.textContent = ''; }, 3000);

        // Update dynamic QR section visibility
        updateDynamicQRSection();
    } catch (error) {
        $('#require-registered-qr').checked = !checked;
        console.error('Error saving setting:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
    }
}

// Update dynamic QR code section visibility and generate QR
async function updateDynamicQRSection() {
    const requireRegistered = $('#require-registered-qr').checked;
    const section = $('#dynamic-qr-section');

    if (!requireRegistered) {
        section.classList.remove('hidden');
        await generateDynamicQR();
    } else {
        section.classList.add('hidden');
    }
}

// Generate the dynamic QR code for /vote/new using backend API
async function generateDynamicQR() {
    const displayEl = $('#dynamic-qr-display');

    try {
        // Fetch QR code image from backend
        const response = await fetch('/api/admin/open-voting-qr');

        if (!response.ok) {
            throw new Error('Failed to generate QR code');
        }

        const blob = await response.blob();
        const imageUrl = URL.createObjectURL(blob);

        displayEl.innerHTML = `<img src="${imageUrl}" alt="Open Voting QR Code" class="w-full h-auto">`;
    } catch (error) {
        console.error('Error generating QR code:', error);
        displayEl.innerHTML = '<div class="text-center text-red-600 text-sm p-4">Error generating QR code</div>';
    }
}

// Download the dynamic QR code
async function downloadDynamicQR() {
    const img = $('#dynamic-qr-display').querySelector('img');
    if (!img) {
        Toast.error('QR code not generated yet');
        return;
    }

    try {
        const response = await fetch('/api/admin/open-voting-qr');
        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = 'open-voting-qr.png';
        a.click();
        URL.revokeObjectURL(url);
        Toast.success('QR code downloaded');
    } catch (error) {
        Toast.error('Failed to download QR code');
    }
}

// Print the dynamic QR code
function printDynamicQR() {
    const img = $('#dynamic-qr-display').querySelector('img');
    if (!img) {
        Toast.error('QR code not generated yet');
        return;
    }

    const printWindow = window.open('', '_blank');
    printWindow.document.write(`
        <html>
        <head>
            <title>Open Voting QR Code</title>
            <style>
                body {
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    justify-content: center;
                    min-height: 100vh;
                    margin: 0;
                    font-family: Arial, sans-serif;
                }
                img {
                    border: 2px solid #000;
                    padding: 20px;
                    background: white;
                }
                h1 {
                    font-size: 24px;
                    margin-bottom: 10px;
                }
                p {
                    font-size: 18px;
                    margin: 10px 0;
                    text-align: center;
                }
                @media print {
                    body {
                        min-height: 0;
                    }
                }
            </style>
            <script>
                // Close window after print dialog is closed (print or cancel)
                window.onafterprint = function() {
                    window.close();
                };
            </script>
        </head>
        <body>
            <h1>Scan to Vote!</h1>
            <p>Everyone scans the same code</p>
            <img src="${img.src}" alt="Voting QR Code">
            <p style="font-size: 14px; color: #666; margin-top: 20px;">
                Each scan creates a unique voting session
            </p>
        </body>
        </html>
    `);
    printWindow.document.close();
    setTimeout(() => {
        printWindow.print();
    }, 250);
}

// Save Base URL
async function saveBaseURL() {
    if (!validateRequired([['#base-url', 'Base URL']])) return;

    const url = $('#base-url').value;
    const messageEl = $('#base-url-message');
    const saveBtn = $('#save-base-url');

    messageEl.textContent = 'Saving...';
    messageEl.className = 'mt-2 text-sm text-blue-600';
    Loading.show(saveBtn);

    try {
        await API.post('/api/admin/settings', {base_url: url});
        messageEl.textContent = 'Base URL saved successfully! QR codes will now use this URL.';
        messageEl.className = 'mt-2 text-sm text-green-600';
    } catch (error) {
        console.error('Error saving base URL:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
    } finally {
        Loading.hide(saveBtn);
    }
}

// Save Voting Instructions
async function saveInstructions() {
    const instructions = $('#voting-instructions').value;
    const messageEl = $('#instructions-message');
    const saveBtn = $('#save-instructions');

    messageEl.textContent = 'Saving...';
    messageEl.className = 'mt-2 text-sm text-blue-600';
    Loading.show(saveBtn);

    try {
        await API.post('/api/admin/settings', {voting_instructions: instructions});
        messageEl.textContent = 'Instructions saved successfully!';
        messageEl.className = 'mt-2 text-sm text-green-600';
    } catch (error) {
        console.error('Error saving instructions:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
    } finally {
        Loading.hide(saveBtn);
    }
}

// Save DerbyNet Settings (URL and Credentials)
async function saveDerbyNetSettings() {
    if (!validateRequired([['#derbynet-url', 'DerbyNet URL']])) return;

    const url = $('#derbynet-url').value;
    const role = $('#derbynet-role').value;
    const password = $('#derbynet-password').value;
    const messageEl = $('#derbynet-message');
    const saveBtn = $('#save-derbynet');

    if (role && !password) {
        messageEl.textContent = 'Please enter a password for the selected role';
        messageEl.className = 'text-sm text-orange-600';
        return;
    }

    messageEl.textContent = 'Saving...';
    messageEl.className = 'text-sm text-blue-600';
    Loading.show(saveBtn);

    try {
        await API.post('/api/admin/settings', {
            derbynet_url: url,
            derbynet_role: role,
            derbynet_password: password
        });

        let message = 'DerbyNet settings saved successfully!';
        if (role) {
            message += ` Authentication configured as ${role}.`;
        }

        messageEl.textContent = message;
        messageEl.className = 'text-sm text-green-600';
        $('#derbynet-password').value = '';
    } catch (error) {
        console.error('Error saving DerbyNet settings:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'text-sm text-red-600';
    } finally {
        Loading.hide(saveBtn);
    }
}

// Test DerbyNet Connection
async function testDerbyNet() {
    if (!validateRequired([['#derbynet-url', 'DerbyNet URL']])) return;

    const url = $('#derbynet-url').value;
    const messageEl = $('#test-message');
    const testBtn = $('#test-derbynet');

    messageEl.textContent = 'Testing connection...';
    messageEl.className = 'mt-2 text-sm text-blue-600';
    Loading.show(testBtn);

    try {
        const result = await API.post('/api/admin/test-derbynet', {derbynet_url: url});

        let message = `✓ Connection successful! Found ${result.total_racers} racers`;
        if (result.total_awards > 0) {
            message += `, ${result.total_awards} awards`;
        }
        if (result.authenticated) {
            message += ` (authenticated as ${result.role})`;
        }

        messageEl.textContent = message;
        messageEl.className = 'mt-2 text-sm text-green-600';
    } catch (error) {
        console.error('Error testing connection:', error);
        messageEl.textContent = `✗ Connection failed: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
    } finally {
        Loading.hide(testBtn);
    }
}

// Seed Categories
async function seedCategories() {
    const messageEl = $('#seed-categories-message');
    const seedBtn = $('#seed-categories');

    messageEl.textContent = 'Seeding categories...';
    messageEl.className = 'mt-2 text-sm text-blue-600';
    Loading.show(seedBtn);

    try {
        const result = await API.post('/api/admin/seed-mock-data', {seed_type: 'categories'});
        messageEl.textContent = `Success! ${result.message}`;
        messageEl.className = 'mt-2 text-sm text-green-600';
    } catch (error) {
        console.error('Error seeding categories:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
    } finally {
        Loading.hide(seedBtn);
    }
}

// Seed Cars
async function seedCars() {
    const messageEl = $('#seed-cars-message');
    const seedBtn = $('#seed-cars');

    messageEl.textContent = 'Seeding cars...';
    messageEl.className = 'mt-2 text-sm text-blue-600';
    Loading.show(seedBtn);

    try {
        const result = await API.post('/api/admin/seed-mock-data', {seed_type: 'cars'});
        messageEl.textContent = `Success! ${result.message}`;
        messageEl.className = 'mt-2 text-sm text-green-600';
    } catch (error) {
        console.error('Error seeding cars:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
    } finally {
        Loading.hide(seedBtn);
    }
}

// Handle automatic vote clearing dependencies
function updateResetDependencies() {
    const votesCheckbox = $('#reset-votes');
    const votersCheckbox = $('#reset-voters');
    const carsCheckbox = $('#reset-cars');
    const categoriesCheckbox = $('#reset-categories');
    const votesNote = $('#votes-auto-note');

    const needsVotes = votersCheckbox.checked || carsCheckbox.checked || categoriesCheckbox.checked;

    if (needsVotes) {
        votesCheckbox.checked = true;
        votesCheckbox.disabled = true;
        votesNote.classList.remove('hidden');
        votesCheckbox.parentElement.parentElement.classList.add('bg-orange-50', 'border', 'border-orange-200', 'rounded', 'p-2', '-m-2');
    } else {
        votesCheckbox.disabled = false;
        votesNote.classList.add('hidden');
        votesCheckbox.parentElement.parentElement.classList.remove('bg-orange-50', 'border', 'border-orange-200', 'rounded', 'p-2', '-m-2');
    }
}

// Reset Selected Data
async function resetSelected() {
    const messageEl = $('#reset-message');
    const checkedBoxes = $$('input[name="reset-item"]:checked');
    const tables = Array.from(checkedBoxes).map(cb => cb.value);

    if (tables.length === 0) {
        Toast.warning('Please select at least one item to reset');
        return;
    }

    const itemsList = tables.join(', ').toUpperCase();

    // First confirmation
    const firstConfirm = await Confirm.danger(
        `This will DELETE the following data: ${itemsList}`,
        'Warning: Data Deletion'
    );
    if (!firstConfirm) return;

    // Second confirmation
    const secondConfirm = await Confirm.danger(
        `FINAL WARNING! The following will be permanently deleted: ${tables.join(', ')}. This cannot be undone!`,
        'Final Confirmation'
    );
    if (!secondConfirm) return;

    messageEl.textContent = 'Resetting database...';
    messageEl.className = 'mt-2 text-sm text-blue-600';

    const resetBtn = $('#reset-db');
    Loading.show(resetBtn);

    try {
        const result = await API.post('/api/admin/reset-database', {tables: tables});
        messageEl.textContent = result.message;
        messageEl.className = 'mt-2 text-sm text-green-600';
        checkedBoxes.forEach(cb => {
            if (!cb.disabled) cb.checked = false;
        });
        updateResetDependencies();
        Toast.success('Data reset complete');
    } catch (error) {
        console.error('Error resetting database:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
    } finally {
        Loading.hide(resetBtn);
    }
}

// Reset ALL Data
async function resetAll() {
    const messageEl = $('#reset-message');

    // First confirmation
    const firstConfirm = await Confirm.danger(
        'This will DELETE ALL DATA including votes, voters, cars, categories, and settings.',
        'Warning: Delete Everything?'
    );
    if (!firstConfirm) return;

    // Second confirmation
    const secondConfirm = await Confirm.danger(
        'FINAL WARNING! ALL DATA will be permanently deleted. This cannot be undone!',
        'Final Confirmation'
    );
    if (!secondConfirm) return;

    messageEl.textContent = 'Resetting ALL data...';
    messageEl.className = 'mt-2 text-sm text-blue-600';

    const resetBtn = $('#reset-all');
    Loading.show(resetBtn);

    try {
        const result = await API.post('/api/admin/reset-database', {tables: ['votes', 'voters', 'cars', 'categories', 'settings']});
        messageEl.textContent = result.message;
        messageEl.className = 'mt-2 text-sm text-green-600';
        $$('input[name="reset-item"]').forEach(cb => {
            cb.checked = false;
            cb.disabled = false;
        });
        updateResetDependencies();
        Toast.success('All data has been reset');
    } catch (error) {
        console.error('Error resetting database:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
    } finally {
        Loading.hide(resetBtn);
    }
}

// ===== VOTER TYPES =====

function renderVoterTypes() {
    const container = $('#voter-types-list');
    if (!container) return;

    container.innerHTML = voterTypes.map((type, index) => {
        const isRequired = type === 'general' || type === 'racer';
        return `
            <div class="flex items-center justify-between p-3 bg-gray-50 rounded-lg border ${isRequired ? 'border-blue-200' : 'border-gray-200'}">
                <div class="flex items-center gap-2">
                    <span class="font-medium text-gray-800">${esc(type)}</span>
                    ${isRequired ? '<span class="text-xs bg-blue-100 text-blue-700 px-2 py-1 rounded">Required</span>' : ''}
                </div>
                ${!isRequired ? `
                    <button onclick="removeVoterType(${index})" class="text-red-600 hover:text-red-800 text-sm font-medium">
                        Remove
                    </button>
                ` : '<span class="text-xs text-gray-500">Cannot remove</span>'}
            </div>
        `;
    }).join('');
}

function addVoterType() {
    const input = $('#new-voter-type');
    const newType = input.value.trim();

    if (!newType) {
        Toast.error('Please enter a voter type name');
        return;
    }

    // Check if already exists (case insensitive)
    if (voterTypes.some(t => t.toLowerCase() === newType.toLowerCase())) {
        Toast.error('This voter type already exists');
        return;
    }

    voterTypes.push(newType);
    renderVoterTypes();
    input.value = '';
    Toast.success(`Added voter type: ${newType}`);
}

function removeVoterType(index) {
    const type = voterTypes[index];
    if (type === 'general' || type === 'racer') {
        Toast.error('Cannot remove required voter types');
        return;
    }

    voterTypes.splice(index, 1);
    renderVoterTypes();
    Toast.success(`Removed voter type: ${type}`);
}

async function saveVoterTypes() {
    const messageEl = $('#voter-types-message');
    const saveBtn = $('#save-voter-types');

    try {
        Loading.show(saveBtn);
        await API.post('/api/admin/settings', { voter_types: voterTypes });
        messageEl.textContent = 'Voter types saved successfully';
        messageEl.className = 'mt-2 text-sm text-green-600';
        setTimeout(() => { messageEl.textContent = ''; }, 3000);
        Toast.success('Voter types saved');
    } catch (error) {
        console.error('Error saving voter types:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
        Toast.error('Failed to save voter types');
    } finally {
        Loading.hide(saveBtn);
    }
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    $('#save-base-url').addEventListener('click', saveBaseURL);
    $('#save-instructions').addEventListener('click', saveInstructions);
    $('#save-derbynet').addEventListener('click', saveDerbyNetSettings);
    $('#test-derbynet').addEventListener('click', testDerbyNet);
    $('#seed-categories').addEventListener('click', seedCategories);
    $('#seed-cars').addEventListener('click', seedCars);
    $('#reset-db').addEventListener('click', resetSelected);
    $('#reset-all').addEventListener('click', resetAll);
    $('#require-registered-qr').addEventListener('change', toggleRequireRegisteredQR);

    // Dynamic QR code buttons
    $('#download-dynamic-qr').addEventListener('click', downloadDynamicQR);
    $('#print-dynamic-qr').addEventListener('click', printDynamicQR);

    // Voter types
    $('#add-voter-type').addEventListener('click', addVoterType);
    $('#save-voter-types').addEventListener('click', saveVoterTypes);
    $('#new-voter-type').addEventListener('keypress', (e) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            addVoterType();
        }
    });

    // Reset dependencies
    $('#reset-voters').addEventListener('change', updateResetDependencies);
    $('#reset-cars').addEventListener('change', updateResetDependencies);
    $('#reset-categories').addEventListener('change', updateResetDependencies);
    $('#reset-votes').addEventListener('change', updateResetDependencies);

    loadSettings();
});
