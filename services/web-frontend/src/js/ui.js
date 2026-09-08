// SimpleCloud Web UI - DOM Rendering & Display Helpers

/**
 * Format raw byte counts into human-readable strings.
 * @param {number} bytes
 * @returns {string}
 */
function formatBytes(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

/**
 * Format ISO date string to localized date string.
 * @param {string|null} dateStr
 * @returns {string}
 */
function formatDate(dateStr) {
  if (!dateStr) return '-';
  const d = new Date(dateStr);
  return isNaN(d.getTime()) ? '-' : d.toLocaleDateString();
}

/**
 * Escape HTML special characters for safe rendering.
 * @param {string} str
 * @returns {string}
 */
function escapeHtml(str) {
  return String(str).replace(/[&<>"']/g, match => {
    const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };
    return map[match];
  });
}

/**
 * Helper to test if a filename corresponds to an image format.
 * @param {string} filename
 * @returns {boolean}
 */
function isImage(filename) {
  return /\.(jpg|jpeg|png|gif|webp|svg)$/i.test(filename || '');
}

/**
 * Helper to test if a filename corresponds to a video format.
 * @param {string} filename
 * @returns {boolean}
 */
function isVideo(filename) {
  return /\.(mp4|webm|ogg|mov)$/i.test(filename || '');
}

/**
 * Helper to test if a filename corresponds to a plain text format.
 * @param {string} filename
 * @returns {boolean}
 */
function isText(filename) {
  return /\.(txt|md|json|js|html|css|go|py|c|cpp|sh|yaml|yml|log)$/i.test(filename || '');
}

/**
 * Return appropriate SVG icon for a file or folder item.
 * @param {object} file
 * @returns {string} SVG markup
 */
function getFileIcon(file) {
  if (file.isFolder) {
    return `<svg width="100%" height="100%" viewBox="0 0 24 24" fill="#0077FF"><path d="M10 4H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2h-8l-2-2z"/></svg>`;
  }
  if (isImage(file.filename)) {
    return `<svg width="100%" height="100%" viewBox="0 0 24 24" fill="none" stroke="#0077FF" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"></rect><circle cx="8.5" cy="8.5" r="1.5"></circle><polyline points="21 15 16 10 5 21"></polyline></svg>`;
  }
  if (isVideo(file.filename)) {
    return `<svg width="100%" height="100%" viewBox="0 0 24 24" fill="none" stroke="#0077FF" stroke-width="1.5"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"></rect><line x1="7" y1="2" x2="7" y2="22"></line><line x1="17" y1="2" x2="17" y2="22"></line><line x1="2" y1="12" x2="22" y2="12"></line><line x1="2" y1="7" x2="7" y2="7"></line><line x1="2" y1="17" x2="7" y2="17"></line><line x1="17" y1="17" x2="22" y2="17"></line><line x1="17" y1="7" x2="22" y2="7"></line></svg>`;
  }
  if (isText(file.filename)) {
    return `<svg width="100%" height="100%" viewBox="0 0 24 24" fill="none" stroke="#0077FF" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path><polyline points="14 2 14 8 20 8"></polyline><line x1="16" y1="13" x2="8" y2="13"></line><line x1="16" y1="17" x2="8" y2="17"></line><polyline points="10 9 9 9 8 9"></polyline></svg>`;
  }
  return `<svg width="100%" height="100%" viewBox="0 0 24 24" fill="none" stroke="#818c99" stroke-width="1.5"><path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"></path><polyline points="13 2 13 9 20 9"></polyline></svg>`;
}

const getFileIconSVG = getFileIcon;

/**
 * Display a temporary floating toast notification.
 * @param {string} message
 * @param {'info'|'success'|'danger'} [type='info']
 */
function showToast(message, type = 'info') {
  const toastContainer = document.getElementById('toast-container');
  if (!toastContainer) return;
  const toast = document.createElement('div');
  toast.className = `toast toast-${type}`;
  toast.textContent = message;
  toastContainer.appendChild(toast);
  setTimeout(() => {
    toast.remove();
  }, 3000);
}

/**
 * Recalculate and update the storage quota progress bar.
 */
function updateQuotaDisplay() {
  const quotaFill = document.getElementById('quota-fill');
  const quotaPercent = document.getElementById('quota-percent');
  const quotaDetails = document.getElementById('quota-details');

  if (state.user && typeof state.user.used_bytes === 'number') {
    state.quota.used = state.user.used_bytes;
    if (typeof state.user.quota_bytes === 'number') {
      state.quota.total = state.user.quota_bytes;
    }
  } else {
    let totalUsed = 0;
    if (Array.isArray(state.files)) {
      state.files.forEach(file => {
        totalUsed += (file.size || 0);
      });
    }
    state.quota.used = totalUsed;
  }

  const percentage = Math.min(100, Math.round((state.quota.used / state.quota.total) * 100));

  if (quotaFill) quotaFill.style.width = `${percentage}%`;
  if (quotaPercent) quotaPercent.textContent = `${percentage}%`;
  if (quotaDetails) quotaDetails.textContent = `${formatBytes(state.quota.used)} of ${formatBytes(state.quota.total)} used`;

  if (quotaFill) {
    quotaFill.classList.remove('warning', 'danger', 'quota-warning', 'quota-danger');
    if (percentage > 85) {
      quotaFill.classList.add('danger', 'quota-danger');
    } else if (percentage > 70) {
      quotaFill.classList.add('warning', 'quota-warning');
    }
  }
}

/**
 * Render the breadcrumbs navigation bar.
 */
function renderBreadcrumbs() {
  const breadcrumbsBar = document.getElementById('breadcrumbs-bar');
  if (!breadcrumbsBar) return;
  let html = '';
  state.breadcrumbs.forEach((crumb, idx) => {
    const isLast = idx === state.breadcrumbs.length - 1;
    if (idx > 0) {
      html += `<span class="breadcrumb-separator">/</span>`;
    }
    if (isLast) {
      html += `<span class="breadcrumb-item active">${escapeHtml(crumb.name)}</span>`;
    } else {
      html += `<span class="breadcrumb-item" data-id="${crumb.id || ''}">${escapeHtml(crumb.name)}</span>`;
    }
  });
  breadcrumbsBar.innerHTML = html;

  breadcrumbsBar.querySelectorAll('.breadcrumb-item:not(.active)').forEach(item => {
    item.addEventListener('click', () => {
      const folderId = item.dataset.id || null;
      navigateToBreadcrumb(folderId);
    });
  });
}

function navigateToBreadcrumb(folderId) {
  const targetIdx = state.breadcrumbs.findIndex(b => b.id === folderId);
  if (targetIdx !== -1) {
    state.breadcrumbs = state.breadcrumbs.slice(0, targetIdx + 1);
    state.currentFolderId = folderId;
    renderBreadcrumbs();
    renderWorkspace();
  }
}

function navigateToFolder(folder) {
  state.currentFolderId = folder.id;
  state.breadcrumbs.push({ id: folder.id, name: folder.name || folder.filename });
  renderBreadcrumbs();
  renderWorkspace();
}

/**
 * Render main workspace contents based on state filters and current viewMode.
 */
function renderWorkspace() {
  const workspace = document.getElementById('workspace');
  if (!workspace) return;

  const currentFolders = (state.folders || []).filter(f => {
    if (state.currentFolderId) return f.parent_id === state.currentFolderId;
    return !f.parent_id;
  }).map(f => ({
    id: f.id,
    filename: f.name,
    isFolder: true,
    size: 0,
    created_at: f.created_at
  }));

  const currentFiles = (state.files || []).filter(f => {
    if (state.currentFolderId) return f.folder_id === state.currentFolderId;
    return !f.folder_id;
  }).map(f => ({
    ...f,
    filename: f.filename || f.name,
    isFolder: false
  }));

  let allItems = [...currentFolders, ...currentFiles];

  if (state.searchQuery) {
    allItems = allItems.filter(item => item.filename.toLowerCase().includes(state.searchQuery.toLowerCase()));
  }

  allItems.sort((a, b) => {
    if (a.isFolder && !b.isFolder) return -1;
    if (!a.isFolder && b.isFolder) return 1;

    let valA = a[state.sortBy] || a.filename;
    let valB = b[state.sortBy] || b.filename;

    if (state.sortBy === 'name') {
      return valA.localeCompare(valB) * (state.sortOrder === 'asc' ? 1 : -1);
    } else {
      return (valA > valB ? 1 : -1) * (state.sortOrder === 'asc' ? 1 : -1);
    }
  });

  if (allItems.length === 0) {
    workspace.innerHTML = `
      <div class="empty-state">
        <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
        </svg>
        <div class="empty-title">${state.searchQuery ? 'No matching items' : 'Folder is empty'}</div>
        <div>${state.searchQuery ? 'Try adjusting your search query' : 'Drag and drop files here or click Upload'}</div>
      </div>
    `;
    return;
  }

  if (state.viewMode === 'grid') {
    renderGrid(allItems);
  } else {
    renderList(allItems);
  }
}

/**
 * Render items in grid layout.
 * @param {Array<object>} items
 */
function renderGrid(items) {
  const workspace = document.getElementById('workspace');
  if (!workspace) return;
  let html = '<div class="file-grid">';
  items.forEach(item => {
    const downloadUrl = `/api/v1/files/download/${item.id}`;

    html += `
      <div class="grid-card" data-id="${item.id}" data-isfolder="${item.isFolder}">
        ${!item.isFolder ? `
        <div class="card-actions">
          <a href="${downloadUrl}" class="btn-icon" download title="Download">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
              <polyline points="7 10 12 15 17 10"></polyline>
              <line x1="12" y1="15" x2="12" y2="3"></line>
            </svg>
          </a>
        </div>` : ''}
        <div class="grid-card-icon">
          ${getFileIcon(item)}
        </div>
        <div class="grid-card-name">${escapeHtml(item.filename)}</div>
        <div class="grid-card-meta">${item.isFolder ? 'Folder' : formatBytes(item.size)}</div>
      </div>
    `;
  });
  html += '</div>';
  workspace.innerHTML = html;

  workspace.querySelectorAll('.grid-card').forEach(card => {
    card.addEventListener('click', (e) => {
      if (e.target.closest('.card-actions')) return;
      const itemId = card.dataset.id;
      const isFolder = card.dataset.isfolder === 'true';
      if (isFolder) {
        const folder = state.folders.find(f => f.id === itemId);
        if (folder) navigateToFolder(folder);
      } else {
        const file = state.files.find(f => f.id === itemId);
        if (file && typeof handleFileClick === 'function') handleFileClick(file);
      }
    });
  });
}

const renderGridView = renderGrid;

/**
 * Render items in table list layout.
 * @param {Array<object>} items
 */
function renderList(items) {
  const workspace = document.getElementById('workspace');
  if (!workspace) return;
  let html = `
    <table class="file-list">
      <thead>
        <tr>
          <th>Name</th>
          <th>Size</th>
          <th>Created</th>
          <th>Actions</th>
        </tr>
      </thead>
      <tbody>
  `;

  items.forEach(item => {
    const downloadUrl = `/api/v1/files/download/${item.id}`;
    const createdDate = formatDate(item.created_at);

    html += `
      <tr data-id="${item.id}" data-isfolder="${item.isFolder}">
        <td>
          <div class="list-name-col">
            <div class="list-icon">${getFileIcon(item)}</div>
            <span>${escapeHtml(item.filename)}</span>
          </div>
        </td>
        <td>${item.isFolder ? '-' : formatBytes(item.size)}</td>
        <td>${createdDate}</td>
        <td>
          ${!item.isFolder ? `
          <a href="${downloadUrl}" class="btn-icon" download title="Download">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
              <polyline points="7 10 12 15 17 10"></polyline>
              <line x1="12" y1="15" x2="12" y2="3"></line>
            </svg>
          </a>` : '-'}
        </td>
      </tr>
    `;
  });

  html += '</tbody></table>';
  workspace.innerHTML = html;

  workspace.querySelectorAll('.file-list tbody tr').forEach(row => {
    row.addEventListener('click', (e) => {
      if (e.target.closest('a')) return;
      const itemId = row.dataset.id;
      const isFolder = row.dataset.isfolder === 'true';
      if (isFolder) {
        const folder = state.folders.find(f => f.id === itemId);
        if (folder) navigateToFolder(folder);
      } else {
        const file = state.files.find(f => f.id === itemId);
        if (file && typeof handleFileClick === 'function') handleFileClick(file);
      }
    });
  });
}

const renderListView = renderList;
