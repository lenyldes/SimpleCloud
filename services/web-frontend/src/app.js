// SimpleCloud Web UI Application Logic
document.addEventListener('DOMContentLoaded', () => {
  // Application State
  const state = {
    currentPath: '/',
    files: [],
    viewMode: 'grid', // 'grid' | 'list'
    sortBy: 'name',   // 'name' | 'size' | 'date'
    sortOrder: 'asc',
    searchQuery: '',
    quota: {
      used: 0,
      total: 5 * 1024 * 1024 * 1024 // 5 GB default
    },
    user: null
  };

  // DOM Element References
  const workspace = document.getElementById('workspace');
  const breadcrumbsBar = document.getElementById('breadcrumbs-bar');
  const searchInput = document.getElementById('search-input');
  const sortSelect = document.getElementById('sort-select');
  const btnViewGrid = document.getElementById('btn-view-grid');
  const btnViewList = document.getElementById('btn-view-list');
  const btnUpload = document.getElementById('btn-upload');
  const btnNewFolder = document.getElementById('btn-new-folder');
  const fileUploadInput = document.getElementById('file-upload-input');
  const dropzoneOverlay = document.getElementById('dropzone-overlay');
  
  // Quota Elements
  const quotaFill = document.getElementById('quota-fill');
  const quotaPercent = document.getElementById('quota-percent');
  const quotaDetails = document.getElementById('quota-details');

  // Modal Elements
  const modalLightbox = document.getElementById('modal-lightbox');
  const lightboxImg = document.getElementById('lightbox-img');
  const lightboxDownload = document.getElementById('lightbox-download');
  const lightboxClose = document.getElementById('lightbox-close');

  const modalText = document.getElementById('modal-text');
  const textModalTitle = document.getElementById('text-modal-title');
  const textModalContent = document.getElementById('text-modal-content');
  const textModalDownload = document.getElementById('text-modal-download');
  const textModalClose = document.getElementById('text-modal-close');
  const textModalCloseBtn = document.getElementById('text-modal-close-btn');

  const modalVideo = document.getElementById('modal-video');
  const videoModalTitle = document.getElementById('video-modal-title');
  const videoPlayer = document.getElementById('video-player');
  const videoModalDownload = document.getElementById('video-modal-download');
  const videoModalClose = document.getElementById('video-modal-close');
  const videoModalCloseBtn = document.getElementById('video-modal-close-btn');

  const modalNewFolder = document.getElementById('modal-new-folder');
  const folderNameInput = document.getElementById('folder-name-input');
  const folderModalCreate = document.getElementById('folder-modal-create');
  const folderModalCancel = document.getElementById('folder-modal-cancel');
  const folderModalClose = document.getElementById('folder-modal-close');

  // Initialize App
  init();

  async function init() {
    setupEventListeners();
    await checkAuth();
    await loadFiles();
  }

  // --- Auth Check ---
  async function checkAuth() {
    try {
      const res = await fetch('/api/v1/auth/me', { credentials: 'include' });
      if (res.ok) {
        state.user = await res.json();
        const avatar = document.getElementById('user-avatar');
        if (avatar && state.user && state.user.email) {
          avatar.textContent = state.user.email[0].toUpperCase();
        }
      }
    } catch (err) {
      console.warn('Auth check warning:', err);
    }
  }

  // --- API Operations ---
  async function loadFiles() {
    try {
      const res = await fetch('/api/v1/files', { credentials: 'include' });
      if (!res.ok) {
        if (res.status === 401) {
          showToast('Session expired or unauthorized', 'danger');
        }
        throw new Error('Failed to load files');
      }
      const data = await res.json();
      state.files = Array.isArray(data) ? data : [];
      updateQuotaDisplay();
      renderWorkspace();
    } catch (err) {
      console.error('Error loading files:', err);
      showToast('Error loading files', 'danger');
    }
  }

  async function handleFileUpload(files) {
    if (!files || files.length === 0) return;

    for (const file of files) {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('path', state.currentPath);

      try {
        showToast(`Uploading ${file.name}...`);
        const res = await fetch('/api/v1/files/upload', {
          method: 'POST',
          credentials: 'include',
          body: formData
        });

        if (res.ok) {
          showToast(`Successfully uploaded ${file.name}`, 'success');
        } else {
          const errData = await res.json().catch(() => ({}));
          showToast(errData.error || `Upload failed for ${file.name}`, 'danger');
        }
      } catch (err) {
        console.error('Upload error:', err);
        showToast(`Failed to upload ${file.name}`, 'danger');
      }
    }

    await loadFiles();
  }

  async function handleCreateFolder(folderName) {
    if (!folderName.trim()) return;

    // Simulate folder metadata item
    const newFolderObj = {
      id: `folder-${Date.now()}`,
      filename: folderName.trim(),
      size: 0,
      isFolder: true,
      created_at: new Date().toISOString()
    };

    state.files.push(newFolderObj);
    renderWorkspace();
    closeModal(modalNewFolder);
    folderNameInput.value = '';
    showToast(`Folder "${folderName}" created`, 'success');
  }

  // --- Quota Calculation & Display ---
  function updateQuotaDisplay() {
    let totalUsed = 0;
    state.files.forEach(file => {
      totalUsed += (file.size || 0);
    });

    state.quota.used = totalUsed;
    const percentage = Math.min(100, Math.round((state.quota.used / state.quota.total) * 100));

    quotaFill.style.width = `${percentage}%`;
    quotaPercent.textContent = `${percentage}%`;
    quotaDetails.textContent = `${formatBytes(state.quota.used)} of ${formatBytes(state.quota.total)} used`;

    quotaFill.classList.remove('warning', 'danger');
    if (percentage > 85) {
      quotaFill.classList.add('danger');
    } else if (percentage > 70) {
      quotaFill.classList.add('warning');
    }
  }

  // --- Workspace Rendering ---
  function renderWorkspace() {
    let filteredFiles = state.files.filter(file => {
      if (state.searchQuery) {
        return file.filename.toLowerCase().includes(state.searchQuery.toLowerCase());
      }
      return true;
    });

    // Sort files
    filteredFiles.sort((a, b) => {
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

    if (filteredFiles.length === 0) {
      workspace.innerHTML = `
        <div class="empty-state">
          <svg class="empty-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
            <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
          </svg>
          <div class="empty-title">${state.searchQuery ? 'No matching files' : 'Folder is empty'}</div>
          <div>${state.searchQuery ? 'Try adjusting your search query' : 'Drag and drop files here or click Upload'}</div>
        </div>
      `;
      return;
    }

    if (state.viewMode === 'grid') {
      renderGridView(filteredFiles);
    } else {
      renderListView(filteredFiles);
    }
  }

  function renderGridView(files) {
    let html = '<div class="file-grid">';
    files.forEach(file => {
      const isImg = isImage(file.filename);
      const isVid = isVideo(file.filename);
      const downloadUrl = `/api/v1/files/download/${file.id}`;

      html += `
        <div class="grid-card" data-id="${file.id}" data-filename="${escapeHtml(file.filename)}">
          <div class="card-actions">
            <a href="${downloadUrl}" class="btn-icon" download title="Download">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                <polyline points="7 10 12 15 17 10"></polyline>
                <line x1="12" y1="15" x2="12" y2="3"></line>
              </svg>
            </a>
          </div>
          <div class="grid-card-icon">
            ${getFileIconSVG(file)}
          </div>
          <div class="grid-card-name">${escapeHtml(file.filename)}</div>
          <div class="grid-card-meta">${file.isFolder ? 'Folder' : formatBytes(file.size)}</div>
        </div>
      `;
    });
    html += '</div>';
    workspace.innerHTML = html;

    // Attach click handlers
    workspace.querySelectorAll('.grid-card').forEach(card => {
      card.addEventListener('click', (e) => {
        if (e.target.closest('.card-actions')) return;
        const fileId = card.dataset.id;
        const file = state.files.find(f => f.id === fileId);
        if (file) handleFileClick(file);
      });
    });
  }

  function renderListView(files) {
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

    files.forEach(file => {
      const downloadUrl = `/api/v1/files/download/${file.id}`;
      const createdDate = file.created_at ? new Date(file.created_at).toLocaleDateString() : '-';

      html += `
        <tr data-id="${file.id}">
          <td>
            <div class="list-name-col">
              <div class="list-icon">${getFileIconSVG(file)}</div>
              <span>${escapeHtml(file.filename)}</span>
            </div>
          </td>
          <td>${file.isFolder ? '-' : formatBytes(file.size)}</td>
          <td>${createdDate}</td>
          <td>
            <a href="${downloadUrl}" class="btn-icon" download title="Download">
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"></path>
                <polyline points="7 10 12 15 17 10"></polyline>
                <line x1="12" y1="15" x2="12" y2="3"></line>
              </svg>
            </a>
          </td>
        </tr>
      `;
    });

    html += '</tbody></table>';
    workspace.innerHTML = html;

    workspace.querySelectorAll('.file-list tbody tr').forEach(row => {
      row.addEventListener('click', (e) => {
        if (e.target.closest('a')) return;
        const fileId = row.dataset.id;
        const file = state.files.find(f => f.id === fileId);
        if (file) handleFileClick(file);
      });
    });
  }

  // --- Preview & Interaction Modals ---
  async function handleFileClick(file) {
    if (file.isFolder) {
      showToast(`Opening folder ${file.filename}`);
      return;
    }

    const downloadUrl = `/api/v1/files/download/${file.id}`;

    if (isImage(file.filename)) {
      lightboxImg.src = downloadUrl;
      lightboxDownload.href = downloadUrl;
      openModal(modalLightbox);
    } else if (isVideo(file.filename)) {
      videoModalTitle.textContent = file.filename;
      videoPlayer.src = downloadUrl;
      videoModalDownload.href = downloadUrl;
      openModal(modalVideo);
    } else if (isText(file.filename)) {
      textModalTitle.textContent = file.filename;
      textModalDownload.href = downloadUrl;
      textModalContent.textContent = 'Loading file content...';
      openModal(modalText);

      try {
        const res = await fetch(downloadUrl, { credentials: 'include' });
        if (res.ok) {
          const text = await res.text();
          textModalContent.textContent = text;
        } else {
          textModalContent.textContent = 'Failed to load text content.';
        }
      } catch (err) {
        textModalContent.textContent = 'Error reading file content.';
      }
    } else {
      // Default download for non-previewable files
      window.open(downloadUrl, '_blank');
    }
  }

  // --- Drag and Drop Setup ---
  function setupDragAndDrop() {
    let dragCounter = 0;

    window.addEventListener('dragenter', (e) => {
      e.preventDefault();
      dragCounter++;
      if (dragCounter === 1) {
        dropzoneOverlay.classList.add('active');
        dropzoneOverlay.classList.remove('hidden');
      }
    });

    window.addEventListener('dragleave', (e) => {
      e.preventDefault();
      dragCounter--;
      if (dragCounter === 0) {
        dropzoneOverlay.classList.remove('active');
        dropzoneOverlay.classList.add('hidden');
      }
    });

    window.addEventListener('dragover', (e) => {
      e.preventDefault();
    });

    window.addEventListener('drop', async (e) => {
      e.preventDefault();
      dragCounter = 0;
      dropzoneOverlay.classList.remove('active');
      dropzoneOverlay.classList.add('hidden');

      if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
        await handleFileUpload(e.dataTransfer.files);
      }
    });
  }

  // --- Event Listeners Setup ---
  function setupEventListeners() {
    setupDragAndDrop();

    // Search input
    searchInput.addEventListener('input', (e) => {
      state.searchQuery = e.target.value;
      renderWorkspace();
    });

    // View toggle
    btnViewGrid.addEventListener('click', () => {
      state.viewMode = 'grid';
      btnViewGrid.classList.add('active');
      btnViewList.classList.remove('active');
      renderWorkspace();
    });

    btnViewList.addEventListener('click', () => {
      state.viewMode = 'list';
      btnViewList.classList.add('active');
      btnViewGrid.classList.remove('active');
      renderWorkspace();
    });

    // Sorting
    sortSelect.addEventListener('change', (e) => {
      state.sortBy = e.target.value;
      renderWorkspace();
    });

    // Upload button
    btnUpload.addEventListener('click', () => {
      fileUploadInput.click();
    });

    fileUploadInput.addEventListener('change', (e) => {
      if (e.target.files.length > 0) {
        handleFileUpload(e.target.files);
      }
    });

    // New folder modal
    btnNewFolder.addEventListener('click', () => {
      openModal(modalNewFolder);
      folderNameInput.focus();
    });

    folderModalCancel.addEventListener('click', () => closeModal(modalNewFolder));
    folderModalClose.addEventListener('click', () => closeModal(modalNewFolder));
    folderModalCreate.addEventListener('click', () => {
      handleCreateFolder(folderNameInput.value);
    });

    // Lightbox close
    lightboxClose.addEventListener('click', () => closeModal(modalLightbox));

    // Text modal close
    textModalClose.addEventListener('click', () => closeModal(modalText));
    textModalCloseBtn.addEventListener('click', () => closeModal(modalText));

    // Video modal close
    videoModalClose.addEventListener('click', () => {
      videoPlayer.pause();
      closeModal(modalVideo);
    });
    videoModalCloseBtn.addEventListener('click', () => {
      videoPlayer.pause();
      closeModal(modalVideo);
    });
  }

  // --- Helper Functions ---
  function openModal(modalEl) {
    modalEl.classList.remove('hidden');
    setTimeout(() => modalEl.classList.add('open'), 10);
  }

  function closeModal(modalEl) {
    modalEl.classList.remove('open');
    setTimeout(() => modalEl.classList.add('hidden'), 150);
  }

  function showToast(message, type = 'info') {
    const toastContainer = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    toastContainer.appendChild(toast);
    setTimeout(() => {
      toast.remove();
    }, 3000);
  }

  function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }

  function escapeHtml(str) {
    return String(str).replace(/[&<>"']/g, match => {
      const map = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' };
      return map[match];
    });
  }

  function isImage(filename) {
    return /\.(jpg|jpeg|png|gif|webp|svg)$/i.test(filename);
  }

  function isVideo(filename) {
    return /\.(mp4|webm|ogg|mov)$/i.test(filename);
  }

  function isText(filename) {
    return /\.(txt|md|json|js|html|css|go|py|c|cpp|sh|yaml|yml|log)$/i.test(filename);
  }

  function getFileIconSVG(file) {
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
});
