// Results page functionality (uses common.js utilities)

let conflictsData = null;
let resultsData = null;
let votingOpen = true; // Default to true (safe default - disables conflict resolution)

// Display preferences
let showDetails = localStorage.getItem('results_show_details') !== 'false'; // default true
let showOnlyConflicts = localStorage.getItem('results_show_only_conflicts') === 'true'; // default false

function updatePushButtonState() {
    const pushBtn = $('#push-derbynet');
    if (!pushBtn) return; // Button not loaded yet

    const tieCount = conflictsData && conflictsData.ties ? conflictsData.ties.length : 0;
    const multiWinCount = conflictsData && conflictsData.multi_wins ? conflictsData.multi_wins.length : 0;
    const hasConflicts = tieCount > 0 || multiWinCount > 0;

    if (hasConflicts) {
        pushBtn.disabled = true;
        pushBtn.classList.add('opacity-50', 'cursor-not-allowed');
        pushBtn.classList.remove('hover:bg-green-700');
        pushBtn.title = 'Resolve all conflicts before pushing to DerbyNet';
    } else {
        pushBtn.disabled = false;
        pushBtn.classList.remove('opacity-50', 'cursor-not-allowed');
        pushBtn.classList.add('hover:bg-green-700');
        pushBtn.title = '';
    }
}

async function loadConflicts() {
    try {
        const data = await API.get('/api/admin/results/conflicts');
        conflictsData = data;

        const tieCount = data.ties ? data.ties.length : 0;
        const multiWinCount = data.multi_wins ? data.multi_wins.length : 0;

        if (tieCount === 0 && multiWinCount === 0) {
            // No conflicts - hide panel
            $('#conflicts-panel').classList.add('hidden');
        } else {
            // Show panel with summary
            $('#conflicts-panel').classList.remove('hidden');

            const parts = [];
            if (tieCount > 0) {
                parts.push(`${tieCount} tie${tieCount > 1 ? 's' : ''} requiring resolution`);
            }
            if (multiWinCount > 0) {
                parts.push(`${multiWinCount} car${multiWinCount > 1 ? 's' : ''} won multiple awards`);
            }

            $('#conflicts-summary').textContent = parts.join(' • ');
        }

        // Update push button state based on conflicts
        updatePushButtonState();
    } catch (error) {
        console.error('Error loading conflicts:', error);
        $('#conflicts-panel').classList.add('hidden');
        // On error, allow pushing (fail open)
        updatePushButtonState();
    }
}

async function loadVotingStatus() {
    try {
        const stats = await API.get('/api/admin/stats');
        votingOpen = stats.voting_open !== false; // Default to true for safety
    } catch (error) {
        console.error('Error loading voting status:', error);
        votingOpen = true; // Fail safe - disable conflict resolution
    }
}

function buildConflictsModalContent() {
    if (!conflictsData) return '';

    let html = '';

    // Show ties
    if (conflictsData.ties && conflictsData.ties.length > 0) {
        html += '<div class="mb-6"><h4 class="text-lg font-semibold text-gray-800 mb-3">Tied Categories</h4>';
        if (votingOpen) {
            html += '<div class="bg-blue-50 border border-blue-200 rounded p-3 mb-4 text-sm text-blue-800"><strong>Note:</strong> Conflict resolution is disabled while voting is open. Close voting first to resolve ties.</div>';
        }
        conflictsData.ties.forEach(tie => {
            html += `
                <div class="border rounded-lg p-4 mb-4 bg-yellow-50">
                    <div class="font-semibold text-gray-900 mb-2">${esc(tie.category_name)}</div>
                    <div class="text-sm text-gray-600 mb-3">${tie.tied_cars.length} cars tied with ${tie.tied_cars[0].vote_count} votes each</div>
                    <div class="space-y-2">
                        ${tie.tied_cars.map(car => `
                            <div class="flex items-center justify-between bg-white p-2 rounded">
                                <div>
                                    <span class="font-medium">Car #${esc(car.car_number)}</span>
                                    <span class="text-gray-600">- ${esc(car.racer_name)}</span>
                                </div>
                                <button ${votingOpen ? 'disabled' : ''}
                                        onclick="setManualWinner(${tie.category_id}, ${car.car_id}, '${escapeJs(car.car_number)}', '${escapeJs(car.racer_name)}', '${escapeJs(tie.category_name)}')"
                                        class="px-3 py-1 text-white text-sm rounded ${votingOpen ? 'bg-gray-400 cursor-not-allowed' : 'bg-blue-600 hover:bg-blue-700'}"
                                        ${votingOpen ? 'title="Close voting first to resolve conflicts"' : ''}>
                                    Select as Winner
                                </button>
                            </div>
                        `).join('')}
                    </div>
                </div>
            `;
        });
        html += '</div>';
    }

    // Show multiple wins
    if (conflictsData.multi_wins && conflictsData.multi_wins.length > 0) {
        html += '<div class="mb-6"><h4 class="text-lg font-semibold text-gray-800 mb-3">Multiple Award Winners</h4>';
        if (votingOpen) {
            html += '<div class="bg-blue-50 border border-blue-200 rounded p-3 mb-4 text-sm text-blue-800"><strong>Note:</strong> Conflict resolution is disabled while voting is open. Close voting first to resolve multiple wins.</div>';
        }
        conflictsData.multi_wins.forEach(mw => {
            html += `
                <div class="border rounded-lg p-4 mb-4 bg-blue-50">
                    <div class="font-semibold text-gray-900 mb-2">
                        Car #${esc(mw.car_number)} - ${esc(mw.racer_name)}
                    </div>
                    <div class="text-sm text-gray-600 mb-2">
                        Won ${mw.awards_won.length} award${mw.awards_won.length > 1 ? 's' : ''}
                        ${mw.group_name ? `in "${esc(mw.group_name)}" group` : ''}
                        (limit: ${mw.max_wins_per_car})
                    </div>
                    <div class="text-xs text-gray-500 mb-3">
                        Select an alternative winner for one or more categories to resolve this conflict.
                    </div>
                    <div class="space-y-2">
                        ${mw.category_ids.map((catId, idx) => {
                            const categoryName = mw.awards_won[idx];
                            // Find this category in resultsData to get next highest vote getters
                            const catResult = resultsData ? resultsData.find(c => c.category_id === catId) : null;

                            // Get all cars except the multi-winner
                            const otherCars = catResult && catResult.votes ? catResult.votes.filter(v => v.car_id !== mw.car_id) : [];

                            // Find the highest vote count among remaining cars
                            let alternatives = [];
                            if (otherCars.length > 0) {
                                const maxVotes = otherCars[0].vote_count; // votes are already sorted by vote_count desc
                                // Get ALL cars with the highest vote count (handles ties)
                                alternatives = otherCars.filter(v => v.vote_count === maxVotes);
                            }

                            return `
                                <div class="bg-white p-3 rounded border border-blue-200">
                                    <div class="font-medium text-sm text-gray-800 mb-2">${esc(categoryName)}</div>
                                    ${alternatives.length > 0 ? `
                                        <div class="text-xs text-gray-600 mb-2">Select alternative winner${alternatives.length > 1 ? ' (tied for next highest)' : ''}:</div>
                                        <div class="space-y-1">
                                            ${alternatives.map(alt => `
                                                <button ${votingOpen ? 'disabled' : ''}
                                                        onclick="setManualWinner(${catId}, ${alt.car_id}, '${escapeJs(alt.car_number)}', '${escapeJs(alt.racer_name)}', '${escapeJs(categoryName)}')"
                                                        class="w-full text-left px-2 py-1 rounded text-xs flex items-center justify-between ${votingOpen ? 'bg-gray-200 cursor-not-allowed' : 'bg-gray-50 hover:bg-blue-100'}"
                                                        ${votingOpen ? 'title="Close voting first to resolve conflicts"' : ''}>
                                                    <span>Car #${esc(alt.car_number)} - ${esc(alt.racer_name)} (${alt.vote_count} votes)</span>
                                                    <span class="${votingOpen ? 'text-gray-400' : 'text-blue-600'}">Select</span>
                                                </button>
                                            `).join('')}
                                        </div>
                                    ` : '<div class="text-xs text-gray-500 italic">No other cars with votes</div>'}
                                </div>
                            `;
                        }).join('')}
                    </div>
                </div>
            `;
        });
        html += '</div>';
    }

    return html;
}

function refreshConflictsModalContent() {
    const content = $('#conflicts-content');
    content.innerHTML = buildConflictsModalContent();
}

function showConflictsModal() {
    if (!conflictsData) return;

    refreshConflictsModalContent();
    $('#conflicts-modal').classList.remove('hidden');
}

function hideConflictsModal() {
    $('#conflicts-modal').classList.add('hidden');
}

// Store pending manual winner action
let pendingManualWinner = null;

function showManualWinnerModal(categoryID, carID, carNumber, racerName, categoryName) {
    pendingManualWinner = { categoryID, carID, carNumber, racerName, categoryName };

    const message = `Set Car #${carNumber} (${racerName}) as winner for ${categoryName}`;
    const defaultReason = `Selected car #${carNumber} as winner for ${categoryName}`;

    $('#manual-winner-message').textContent = message;
    $('#manual-winner-reason').value = defaultReason;
    $('#manual-winner-modal').classList.remove('hidden');
    $('#manual-winner-reason').focus();
}

function hideManualWinnerModal() {
    $('#manual-winner-modal').classList.add('hidden');
    pendingManualWinner = null;
}

async function confirmManualWinner() {
    if (!pendingManualWinner) return;

    const reason = $('#manual-winner-reason').value.trim();
    if (!reason) {
        alert('Please enter a reason');
        return;
    }

    const { categoryID, carID, carNumber, racerName, categoryName } = pendingManualWinner;

    try {
        await API.post('/api/admin/results/override-winner', {
            category_id: categoryID,
            car_id: carID,
            reason: reason
        });

        showPushStatus(`Set ${racerName} (Car #${carNumber}) as winner for ${categoryName}`, false);
        hideManualWinnerModal();

        // Check if conflicts modal is open
        const modalWasOpen = !$('#conflicts-modal').classList.contains('hidden');

        // Reload data
        await loadConflicts();
        await loadResults();

        // If modal was open, refresh its content or close if no conflicts
        if (modalWasOpen) {
            const tieCount = conflictsData.ties ? conflictsData.ties.length : 0;
            const multiWinCount = conflictsData.multi_wins ? conflictsData.multi_wins.length : 0;

            if (tieCount === 0 && multiWinCount === 0) {
                hideConflictsModal();
            } else {
                // Refresh modal content with new conflicts data
                refreshConflictsModalContent();
            }
        }
    } catch (error) {
        console.error('Error setting manual winner:', error);
        showPushStatus(`Error: ${error.message}`, true);
    }
}

// For backward compatibility - called from inline onclick
async function setManualWinner(categoryID, carID, carNumber, racerName, categoryName) {
    showManualWinnerModal(categoryID, carID, carNumber, racerName, categoryName);
}

// Store pending clear override action
let pendingClearOverride = null;

function showClearOverrideModal(categoryID, categoryName) {
    pendingClearOverride = { categoryID, categoryName };
    $('#clear-override-message').textContent = `Are you sure you want to clear the manual winner override for ${categoryName}?`;
    $('#clear-override-modal').classList.remove('hidden');
}

function hideClearOverrideModal() {
    $('#clear-override-modal').classList.add('hidden');
    pendingClearOverride = null;
}

async function confirmClearOverride() {
    if (!pendingClearOverride) return;

    const { categoryID, categoryName } = pendingClearOverride;

    try {
        await API.delete(`/api/admin/results/override-winner/${categoryID}`);
        showPushStatus(`Cleared manual override for ${categoryName}`, false);
        hideClearOverrideModal();

        // Reload data
        await loadConflicts();
        await loadResults();
    } catch (error) {
        console.error('Error clearing manual winner:', error);
        showPushStatus(`Error: ${error.message}`, true);
    }
}

// For backward compatibility - called from inline onclick
async function clearManualWinner(categoryID, categoryName) {
    showClearOverrideModal(categoryID, categoryName);
}

function hasConflictOrOverride(category) {
    // Check if category has manual override
    if (category.has_override) {
        return true;
    }

    // Check if category has a tie
    if (conflictsData && conflictsData.ties) {
        if (conflictsData.ties.some(tie => tie.category_id === category.category_id)) {
            return true;
        }
    }

    // Check if category has multi-win conflict
    if (conflictsData && conflictsData.multi_wins) {
        if (conflictsData.multi_wins.some(mw => mw.category_ids.includes(category.category_id))) {
            return true;
        }
    }

    return false;
}

function applyDisplayFilters() {
    const categories = document.querySelectorAll('.category-card');

    categories.forEach(card => {
        const hasConflict = card.dataset.hasConflict === 'true';

        // Show only conflicts filter
        if (showOnlyConflicts && !hasConflict) {
            card.style.display = 'none';
        } else {
            card.style.display = '';
        }

        // Show details filter
        const detailsSection = card.querySelector('.vote-details');
        if (detailsSection) {
            detailsSection.style.display = showDetails ? '' : 'none';
        }
    });
}

async function loadResults() {
    Loading.show('#results-container');
    try {
        const results = await API.get('/api/admin/results');
        resultsData = results;
        const container = $('#results-container');

        if (!results || !Array.isArray(results) || results.length === 0) {
            container.innerHTML = '<div class="bg-white rounded-lg shadow p-8 text-center text-gray-600">No categories or votes yet. Create categories first.</div>';
            return;
        }

        container.innerHTML = results.map(category => {
            const votes = category.votes || [];
            const totalVotes = votes.reduce((sum, v) => sum + v.vote_count, 0);

            // Sort by vote count descending
            votes.sort((a, b) => b.vote_count - a.vote_count);

            // Calculate max votes for highlighting
            const maxVotes = votes.length > 0 ? votes[0].vote_count : 0;

            // Determine winner(s) - check for manual override first
            let winners = [];
            const hasOverride = category.has_override && category.override_car_id;

            if (hasOverride) {
                // Find the override winner in the votes list
                const overrideWinner = votes.find(v => v.car_id === category.override_car_id);
                if (overrideWinner) {
                    winners = [overrideWinner];
                }
            } else {
                // Use vote-based winners
                winners = votes.filter(v => v.vote_count === maxVotes);
            }

            const hasConflict = hasConflictOrOverride(category);

            return `
                <div class="category-card bg-white rounded-lg shadow-lg p-6" data-has-conflict="${hasConflict}">
                    <div class="flex items-center justify-between mb-4">
                        <h2 class="text-2xl font-bold text-gray-800">${esc(category.category_name)}</h2>
                        <span class="text-sm text-gray-600">${totalVotes} total votes</span>
                    </div>

                    ${winners.length > 0 ? `
                        <div class="bg-yellow-50 border-2 border-yellow-400 rounded-lg p-4 mb-4 relative">
                            ${hasOverride ? `
                                <div class="absolute top-3 right-3">
                                    <span class="inline-flex items-center px-2 py-1 rounded bg-purple-100 text-purple-800 text-xs font-medium">
                                        <svg class="w-3 h-3 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                            <path d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-11a1 1 0 10-2 0v2H7a1 1 0 100 2h2v2a1 1 0 102 0v-2h2a1 1 0 100-2h-2V7z"/>
                                        </svg>
                                        Manual Override
                                    </span>
                                </div>
                            ` : ''}
                            <div class="text-yellow-800 font-semibold mb-2">
                                ${hasOverride ? 'Manual Winner:' : `Winner${winners.length > 1 ? 's (Tie)' : ''}:`}
                            </div>
                            <div class="space-y-4">
                                ${winners.map(w => `
                                    <div class="flex items-center space-x-4">
                                        <img src="/cars/${w.car_id}/photo" alt="${esc(w.car_name) || 'Car'}"
                                             class="w-64 h-auto object-contain rounded-lg shadow-md border-2 border-yellow-400">
                                        <div>
                                            <div class="text-xl font-bold text-yellow-900">
                                                ${esc(w.racer_name) || 'Unknown Driver'}
                                            </div>
                                            <div class="text-lg text-yellow-800">
                                                Car #${esc(w.car_number)}${w.car_name ? ` - ${esc(w.car_name)}` : ''}
                                            </div>
                                            <div class="text-sm text-yellow-700">
                                                ${w.vote_count} votes
                                            </div>
                                        </div>
                                    </div>
                                `).join('')}
                            </div>
                            ${hasOverride ? `
                                <div class="mt-3 pt-3 border-t border-yellow-300">
                                    <div class="flex items-center justify-between">
                                        ${category.override_reason ? `
                                            <div class="text-xs text-yellow-800 italic">
                                                <strong>Reason:</strong> ${esc(category.override_reason)}
                                            </div>
                                        ` : '<div></div>'}
                                        <button ${votingOpen ? 'disabled' : ''}
                                                onclick="clearManualWinner(${category.category_id}, '${escapeJs(category.category_name)}')"
                                                class="text-xs font-medium whitespace-nowrap ${votingOpen ? 'text-gray-400 cursor-not-allowed' : 'text-purple-600 hover:text-purple-800 underline'}"
                                                ${votingOpen ? 'title="Close voting first to clear overrides"' : ''}>
                                            Clear Override
                                        </button>
                                    </div>
                                </div>
                            ` : ''}
                        </div>
                    ` : ''}

                    <div class="vote-details space-y-2">
                        ${votes.map((vote, index) => {
                            const percentage = totalVotes > 0 ? (vote.vote_count / totalVotes * 100) : 0;
                            const isWinner = vote.vote_count === maxVotes;

                            return `
                                <div class="border rounded-lg p-3 ${isWinner ? 'bg-yellow-50 border-yellow-400' : 'bg-gray-50'}">
                                    <div class="flex items-center justify-between mb-2">
                                        <div class="flex items-center space-x-3">
                                            <img src="/cars/${vote.car_id}/photo" alt="${esc(vote.car_name) || 'Car'}"
                                                 class="w-32 h-auto object-contain rounded shadow">
                                            <div>
                                                <div class="font-semibold text-gray-800">
                                                    ${index + 1}. ${esc(vote.racer_name) || 'Unknown'}
                                                </div>
                                                <div class="text-sm text-gray-600">
                                                    Car #${esc(vote.car_number)}${vote.car_name ? ` - ${esc(vote.car_name)}` : ''}
                                                </div>
                                            </div>
                                        </div>
                                        <span class="font-bold text-blue-600">${vote.vote_count} votes</span>
                                    </div>
                                    <div class="w-full bg-gray-200 rounded-full h-4">
                                        <div class="bg-blue-600 h-4 rounded-full transition-all duration-300"
                                             style="width: ${percentage}%"></div>
                                    </div>
                                    <div class="text-xs text-gray-600 mt-1 text-right">${percentage.toFixed(1)}%</div>
                                </div>
                            `;
                        }).join('')}

                        ${votes.length === 0 ? `
                            <div class="text-center text-gray-500 py-4">No votes for this category yet</div>
                        ` : ''}
                    </div>
                </div>
            `;
        }).join('');

        // Apply display filters after rendering
        applyDisplayFilters();

    } catch (error) {
        console.error('Error loading results:', error);
        $('#results-container').innerHTML =
            '<div class="bg-red-50 border border-red-400 rounded-lg p-4 text-red-700">Error loading results</div>';
    } finally {
        Loading.hide('#results-container');
    }
}

// ===== DERBYNET PUSH =====
function showPushStatus(message, isError = false, isWarning = false) {
    const statusEl = $('#push-status');
    const messageEl = $('#push-message');
    messageEl.innerHTML = message;

    statusEl.className = 'mb-4 rounded-lg shadow-lg p-4';
    if (isError) {
        statusEl.className += ' bg-red-50 border border-red-400';
        messageEl.className = 'text-sm text-red-700';
    } else if (isWarning) {
        statusEl.className += ' bg-yellow-50 border border-yellow-400';
        messageEl.className = 'text-sm text-yellow-700';
    } else {
        statusEl.className += ' bg-green-50 border border-green-400';
        messageEl.className = 'text-sm text-green-700';
    }

    statusEl.classList.remove('hidden');

    if (!isError && !message.includes('Pushing')) {
        setTimeout(() => {
            statusEl.classList.add('hidden');
        }, 10000);
    }
}

async function pushResultsToDerbyNet() {
    // Check for conflicts before pushing
    const tieCount = conflictsData && conflictsData.ties ? conflictsData.ties.length : 0;
    const multiWinCount = conflictsData && conflictsData.multi_wins ? conflictsData.multi_wins.length : 0;

    if (tieCount > 0 || multiWinCount > 0) {
        showPushStatus('Cannot push to DerbyNet: Please resolve all conflicts first', true);
        return;
    }

    const pushBtn = $('#push-derbynet');
    Loading.show(pushBtn);
    showPushStatus('Fetching settings...');

    try {
        const settings = await API.get('/api/admin/settings');

        if (!settings.derbynet_url) {
            showPushStatus('DerbyNet URL not configured. Please set it in Settings first.', true);
            return;
        }

        showPushStatus('Pushing results to DerbyNet...');

        const result = await API.post('/api/admin/push-results-derbynet', {derbynet_url: settings.derbynet_url});

        if (result.status === 'success') {
            const msg = result.message || `Pushed ${result.winners_pushed} winners to DerbyNet!`;
            showPushStatus(msg, false, result.skipped > 0);
        } else if (result.status === 'partial') {
            const msg = formatSyncResultDetails(result, result.winners_pushed, 'winners pushed');
            showPushStatus(msg, false, true);
        } else {
            showPushStatus(`Error: ${result.message || 'Failed to push results'}`, true);
        }
    } catch (error) {
        console.error('Error pushing to DerbyNet:', error);
        showPushStatus(`Error: ${error.message}`, true);
    } finally {
        Loading.hide(pushBtn);
    }
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    // Set initial checkbox states from localStorage
    $('#show-details').checked = showDetails;
    $('#show-only-conflicts').checked = showOnlyConflicts;

    // Load results, conflicts, and voting status
    loadResults();
    loadConflicts();
    loadVotingStatus();

    // Refresh every 10 seconds
    setInterval(() => {
        loadResults();
        loadConflicts();
        loadVotingStatus();
    }, 10000);

    // Wire up filter checkboxes
    $('#show-details').addEventListener('change', (e) => {
        showDetails = e.target.checked;
        localStorage.setItem('results_show_details', showDetails);
        applyDisplayFilters();
    });

    $('#show-only-conflicts').addEventListener('change', (e) => {
        showOnlyConflicts = e.target.checked;
        localStorage.setItem('results_show_only_conflicts', showOnlyConflicts);
        applyDisplayFilters();
    });

    // Wire up buttons
    $('#push-derbynet').addEventListener('click', pushResultsToDerbyNet);
    $('#review-conflicts-btn').addEventListener('click', showConflictsModal);
    $('#close-conflicts-modal').addEventListener('click', hideConflictsModal);

    // Manual winner modal buttons
    $('#cancel-manual-winner').addEventListener('click', hideManualWinnerModal);
    $('#confirm-manual-winner').addEventListener('click', confirmManualWinner);

    // Clear override modal buttons
    $('#cancel-clear-override').addEventListener('click', hideClearOverrideModal);
    $('#confirm-clear-override').addEventListener('click', confirmClearOverride);

    // Close modals when clicking outside
    $('#conflicts-modal').addEventListener('click', (e) => {
        if (e.target.id === 'conflicts-modal') {
            hideConflictsModal();
        }
    });

    $('#manual-winner-modal').addEventListener('click', (e) => {
        if (e.target.id === 'manual-winner-modal') {
            hideManualWinnerModal();
        }
    });

    $('#clear-override-modal').addEventListener('click', (e) => {
        if (e.target.id === 'clear-override-modal') {
            hideClearOverrideModal();
        }
    });
});
