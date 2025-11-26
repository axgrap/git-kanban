// Git Kanban Web UI
let currentBoard = null;
let draggedCard = null;
let draggedFromColumn = null;

// Show status message
function showStatus(message, type = 'info') {
    const statusEl = document.getElementById('status-message');
    statusEl.textContent = message;
    statusEl.className = `status-message ${type}`;
    
    setTimeout(() => {
        statusEl.style.display = 'none';
    }, 5000);
}

// Load board from server
async function loadBoard() {
    try {
        const response = await fetch('/api/load');
        if (!response.ok) {
            throw new Error(`Failed to load board: ${response.statusText}`);
        }
        currentBoard = await response.json();
        renderBoard();
        showStatus('Board loaded successfully', 'success');
    } catch (error) {
        console.error('Error loading board:', error);
        showStatus(`Error: ${error.message}`, 'error');
    }
}

// Save board to server
async function saveBoard() {
    if (!currentBoard) {
        showStatus('No board to save', 'error');
        return;
    }
    
    try {
        showStatus('Saving changes...', 'info');
        const response = await fetch('/api/save', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify(currentBoard)
        });
        
        if (!response.ok) {
            throw new Error(`Failed to save board: ${response.statusText}`);
        }
        
        showStatus('Board saved and committed successfully!', 'success');
        
        // Reload to get updated owner info
        setTimeout(() => loadBoard(), 500);
    } catch (error) {
        console.error('Error saving board:', error);
        showStatus(`Error: ${error.message}`, 'error');
    }
}

// Claim or unclaim a card
async function claimCard(laneIndex, ticketIndex, currentOwner) {
    const action = currentOwner ? 'unclaim' : 'claim';
    let owner = '';
    
    if (action === 'claim') {
        owner = prompt('Enter your name to claim this ticket:');
        if (!owner) {
            return; // User cancelled
        }
    }
    
    try {
        showStatus(`${action === 'claim' ? 'Claiming' : 'Unclaiming'} ticket...`, 'info');
        const response = await fetch('/api/claim', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                lane_index: laneIndex,
                ticket_index: ticketIndex,
                owner: owner,
                action: action
            })
        });
        
        if (!response.ok) {
            throw new Error(`Failed to ${action} ticket: ${response.statusText}`);
        }
        
        currentBoard = await response.json();
        renderBoard();
        showStatus(`Ticket ${action}ed successfully!`, 'success');
    } catch (error) {
        console.error(`Error ${action}ing ticket:`, error);
        showStatus(`Error: ${error.message}`, 'error');
    }
}

// Render board UI
function renderBoard() {
    const boardEl = document.getElementById('board');
    boardEl.innerHTML = '';
    
    if (!currentBoard || !currentBoard.columns) {
        boardEl.innerHTML = '<div class="loading">Loading board</div>';
        return;
    }
    
    currentBoard.columns.forEach((column, columnIndex) => {
        const columnEl = document.createElement('div');
        columnEl.className = 'column';
        columnEl.dataset.columnIndex = columnIndex;
        
        // Column header
        const headerEl = document.createElement('div');
        headerEl.className = 'column-header';
        headerEl.textContent = `${column.name} (${column.cards.length})`;
        columnEl.appendChild(headerEl);
        
        // Cards container
        const cardsEl = document.createElement('div');
        cardsEl.className = 'cards';
        
        column.cards.forEach((card, cardIndex) => {
            const cardEl = createCardElement(card, columnIndex, cardIndex);
            cardsEl.appendChild(cardEl);
        });
        
        columnEl.appendChild(cardsEl);
        
        // Drag and drop events for column
        columnEl.addEventListener('dragover', handleDragOver);
        columnEl.addEventListener('drop', handleDrop);
        columnEl.addEventListener('dragleave', handleDragLeave);
        
        boardEl.appendChild(columnEl);
    });
}

// Create card element
function createCardElement(card, columnIndex, cardIndex) {
    const cardEl = document.createElement('div');
    cardEl.className = 'card';
    cardEl.draggable = true;
    cardEl.tabIndex = 0;
    cardEl.dataset.columnIndex = columnIndex;
    cardEl.dataset.cardIndex = cardIndex;
    
    // Card title with checkbox
    const titleEl = document.createElement('div');
    titleEl.className = 'card-title';
    
    const checkbox = document.createElement('input');
    checkbox.type = 'checkbox';
    checkbox.className = 'card-checkbox';
    checkbox.checked = card.checked;
    checkbox.addEventListener('change', (e) => {
        card.checked = e.target.checked;
        // Auto-save on checkbox change
        saveBoard();
    });
    
    const titleText = document.createTextNode(card.title);
    titleEl.appendChild(checkbox);
    titleEl.appendChild(titleText);
    cardEl.appendChild(titleEl);
    
    // Owner info
    if (card.assignee) {
        const ownerEl = document.createElement('div');
        ownerEl.className = 'card-owner';
        ownerEl.textContent = `Claimed by ${card.assignee}`;
        cardEl.appendChild(ownerEl);
    }
    
    // Drag events
    cardEl.addEventListener('dragstart', handleDragStart);
    cardEl.addEventListener('dragend', handleDragEnd);
    
    // Double-click to claim/unclaim
    cardEl.addEventListener('dblclick', () => {
        claimCard(columnIndex, cardIndex, card.assignee);
    });
    
    // Keyboard support
    cardEl.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
            claimCard(columnIndex, cardIndex, card.assignee);
        }
    });
    
    return cardEl;
}

// Drag and drop handlers
function handleDragStart(e) {
    draggedCard = e.target;
    draggedFromColumn = parseInt(e.target.dataset.columnIndex);
    e.target.classList.add('dragging');
    e.dataTransfer.effectAllowed = 'move';
    e.dataTransfer.setData('text/html', e.target.innerHTML);
}

function handleDragEnd(e) {
    e.target.classList.remove('dragging');
    
    // Remove drag-over class from all columns
    document.querySelectorAll('.column').forEach(col => {
        col.classList.remove('drag-over');
    });
}

function handleDragOver(e) {
    if (e.preventDefault) {
        e.preventDefault();
    }
    
    e.dataTransfer.dropEffect = 'move';
    
    // Find the column element
    let columnEl = e.target;
    while (columnEl && !columnEl.classList.contains('column')) {
        columnEl = columnEl.parentElement;
    }
    
    if (columnEl) {
        columnEl.classList.add('drag-over');
    }
    
    return false;
}

function handleDragLeave(e) {
    e.target.classList.remove('drag-over');
}

function handleDrop(e) {
    if (e.stopPropagation) {
        e.stopPropagation();
    }
    
    // Find the column element
    let columnEl = e.target;
    while (columnEl && !columnEl.classList.contains('column')) {
        columnEl = columnEl.parentElement;
    }
    
    if (!columnEl || !draggedCard) {
        return false;
    }
    
    columnEl.classList.remove('drag-over');
    
    const toColumnIndex = parseInt(columnEl.dataset.columnIndex);
    const fromColumnIndex = draggedFromColumn;
    const cardIndex = parseInt(draggedCard.dataset.cardIndex);
    
    if (toColumnIndex === fromColumnIndex) {
        return false; // Same column, no move
    }
    
    // Move card in data model
    const card = currentBoard.columns[fromColumnIndex].cards.splice(cardIndex, 1)[0];
    
    // If moving to "Done" column, mark as checked
    if (currentBoard.columns[toColumnIndex].name.toLowerCase() === 'done') {
        card.checked = true;
    }
    
    currentBoard.columns[toColumnIndex].cards.push(card);
    
    // Re-render board
    renderBoard();
    
    // Show status and auto-save
    showStatus(`Moved card to ${currentBoard.columns[toColumnIndex].name}. Click Save to commit.`, 'info');
    
    return false;
}

// Modal handling
function showHelpModal() {
    document.getElementById('help-modal').style.display = 'flex';
}

function hideHelpModal() {
    document.getElementById('help-modal').style.display = 'none';
}

// Event listeners
document.addEventListener('DOMContentLoaded', () => {
    // Load board on startup
    loadBoard();
    
    // Button handlers
    document.getElementById('reload-btn').addEventListener('click', loadBoard);
    document.getElementById('save-btn').addEventListener('click', saveBoard);
    document.getElementById('help-btn').addEventListener('click', showHelpModal);
    
    // Modal close
    document.querySelector('.close').addEventListener('click', hideHelpModal);
    document.getElementById('help-modal').addEventListener('click', (e) => {
        if (e.target.id === 'help-modal') {
            hideHelpModal();
        }
    });
    
    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
        // Ctrl+S to save
        if (e.ctrlKey && e.key === 's') {
            e.preventDefault();
            saveBoard();
        }
        
        // Ctrl+R to reload (let default behavior work but also trigger our reload)
        if (e.ctrlKey && e.key === 'r') {
            e.preventDefault();
            loadBoard();
        }
        
        // ? to show help
        if (e.key === '?' && !e.ctrlKey && !e.metaKey && !e.altKey) {
            showHelpModal();
        }
        
        // Escape to close modal
        if (e.key === 'Escape') {
            hideHelpModal();
        }
    });
});
