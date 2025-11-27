// Dashboard page functionality (uses common.js utilities)

let votingOpen = true;

// Handle WebSocket messages for dashboard
AdminWS.on('voting_status', (payload) => {
    votingOpen = payload.open;
    updateVotingStatus();
    if (!votingOpen) {
        $('#countdown-display').classList.add('hidden');
    }
});

AdminWS.on('countdown', (payload) => {
    updateCountdown(payload.seconds_remaining);
});

// Update countdown display
function updateCountdown(secondsRemaining) {
    const display = $('#countdown-display');
    if (secondsRemaining <= 0) {
        display.classList.add('hidden');
        return;
    }

    $('#countdown-time').textContent = formatTime(secondsRemaining);
    display.classList.remove('hidden');

    // Change color based on time remaining
    const countdownEl = display.querySelector('p');
    if (secondsRemaining <= 60) {
        countdownEl.className = 'countdown-urgent font-bold text-lg';
    } else if (secondsRemaining <= 300) {
        countdownEl.className = 'countdown-warning font-bold text-lg';
    } else {
        countdownEl.className = 'countdown-normal font-bold text-lg';
    }
}

// Load stats
async function loadStats() {
    try {
        const stats = await API.get('/api/admin/stats');

        $('#stat-total-voters').textContent = stats.total_voters || 0;
        $('#stat-voters-voted').textContent = stats.voters_who_voted || 0;
        $('#stat-total-votes').textContent = stats.total_votes || 0;
        $('#stat-total-cars').textContent = stats.total_cars || 0;

        votingOpen = stats.voting_open;
        updateVotingStatus();
    } catch (error) {
        console.error('Error loading stats:', error);
    }
}

// Update voting status display
function updateVotingStatus() {
    const statusEl = $('#voting-status');
    const buttonEl = $('#toggle-voting');

    if (votingOpen) {
        statusEl.textContent = 'Open';
        statusEl.className = 'text-2xl font-bold status-open';
        buttonEl.textContent = 'Close Voting';
        buttonEl.className = 'bg-red-600 text-white px-6 py-3 rounded-lg font-semibold hover:bg-red-700';
    } else {
        statusEl.textContent = 'Closed';
        statusEl.className = 'text-2xl font-bold status-closed';
        buttonEl.textContent = 'Open Voting';
        buttonEl.className = 'bg-green-600 text-white px-6 py-3 rounded-lg font-semibold hover:bg-green-700';
    }
}

// Toggle voting
async function toggleVoting() {
    const newState = !votingOpen;
    const toggleBtn = $('#toggle-voting');
    Loading.show(toggleBtn);

    try {
        await API.post('/api/admin/voting-control', { open: newState });
        votingOpen = newState;
        updateVotingStatus();
        loadStats();
        Toast.success(newState ? 'Voting opened' : 'Voting closed');
    } catch (error) {
        console.error('Error toggling voting:', error);
        Toast.error('Failed to toggle voting status');
    } finally {
        Loading.hide(toggleBtn);
    }
}

// Set timer
async function setTimer(minutes) {
    const messageEl = $('#timer-message');
    messageEl.textContent = 'Setting timer...';
    messageEl.className = 'mt-2 text-sm text-blue-600';

    try {
        await API.post('/api/admin/voting-timer', { minutes });
        messageEl.textContent = `Timer set! Voting will close in ${minutes} minute${minutes > 1 ? 's' : ''}.`;
        messageEl.className = 'mt-2 text-sm text-green-600';
        loadStats();
    } catch (error) {
        console.error('Error setting timer:', error);
        messageEl.textContent = `Error: ${error.message}`;
        messageEl.className = 'mt-2 text-sm text-red-600';
    }
}

// Set custom timer
function setCustomTimer() {
    const minutes = parseInt($('#custom-timer').value);
    if (!minutes || minutes < 1 || minutes > 60) {
        Toast.warning('Please enter a number between 1 and 60');
        return;
    }
    setTimer(minutes);
}

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    $('#toggle-voting').addEventListener('click', toggleVoting);

    // Timer buttons
    $$('[data-timer]').forEach(btn => {
        btn.addEventListener('click', () => setTimer(parseInt(btn.dataset.timer)));
    });

    $('#set-custom-timer').addEventListener('click', setCustomTimer);

    loadStats();
    setInterval(loadStats, 5000);
});
