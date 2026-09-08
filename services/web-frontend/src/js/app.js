// SimpleCloud Web UI - Orchestrator & Application Bootstrap

// Global Application State
var state = window.state || {
  currentFolderId: null,
  breadcrumbs: [{ id: null, name: 'All Files' }],
  files: [],
  folders: [],
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
window.state = state;

/**
 * Bootstrap and initialize application components.
 */
async function init() {
  closeProfileDropdown();
  setupEventListeners();
  await checkAuth();
  await loadWorkspaceData();
}

document.addEventListener('DOMContentLoaded', init);

/**
 * Fetch and refresh workspace files and folders.
 */
async function loadWorkspaceData() {
  await checkAuth();
  await Promise.all([loadFiles(), loadFolders()]);
  updateQuotaDisplay();
  renderBreadcrumbs();
  renderWorkspace();
}

/**
 * Fetch user files from backend API.
 */
async function loadFiles() {
  try {
    const res = window.api && window.api.listFiles
      ? await window.api.listFiles(state.currentFolderId)
      : await fetchWithAuth('/api/v1/files');
    if (res.ok) {
      const data = await res.json();
      state.files = Array.isArray(data) ? data : [];
    }
  } catch (err) {
    console.error('Error loading files:', err);
  }
}

/**
 * Fetch user folders from backend API.
 */
async function loadFolders() {
  try {
    const res = window.api && window.api.listFolders
      ? await window.api.listFolders()
      : await fetchWithAuth('/api/v1/folders');
    if (res.ok) {
      const data = await res.json();
      state.folders = Array.isArray(data) ? data : [];
    }
  } catch (err) {
    console.error('Error loading folders:', err);
  }
}

/**
 * Upload one or more selected files.
 * @param {Array<File>|FileList} files
 */
async function handleFileUpload(files) {
  if (!files || files.length === 0) return;

  for (const file of files) {
    const formData = new FormData();
    formData.append('file', file);
    if (state.currentFolderId) {
      formData.append('folder_id', state.currentFolderId);
    }

    try {
      showToast(`Uploading ${file.name}...`);
      const res = window.api && window.api.uploadFile
        ? await window.api.uploadFile(formData)
        : await fetchWithAuth('/api/v1/files/upload', {
            method: 'POST',
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

  await loadWorkspaceData();
}

/**
 * Set up drag and drop visual overlay and drop handler.
 */
function setupDragAndDrop() {
  const dropzoneOverlay = document.getElementById('dropzone-overlay');
  let dragCounter = 0;

  window.addEventListener('dragenter', (e) => {
    e.preventDefault();
    dragCounter++;
    if (dragCounter === 1 && dropzoneOverlay) {
      dropzoneOverlay.classList.add('active');
      dropzoneOverlay.classList.remove('hidden');
    }
  });

  window.addEventListener('dragleave', (e) => {
    e.preventDefault();
    dragCounter--;
    if (dragCounter === 0 && dropzoneOverlay) {
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
    if (dropzoneOverlay) {
      dropzoneOverlay.classList.remove('active');
      dropzoneOverlay.classList.add('hidden');
    }

    if (e.dataTransfer && e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      await handleFileUpload(e.dataTransfer.files);
    }
  });
}

/**
 * Attach DOM event listeners for buttons, inputs, modals, and window actions.
 */
function setupEventListeners() {
  setupDragAndDrop();

  // Cached DOM elements
  const searchInput = document.getElementById('search-input');
  const sortSelect = document.getElementById('sort-select');
  const btnViewGrid = document.getElementById('btn-view-grid');
  const btnViewList = document.getElementById('btn-view-list');
  const btnUpload = document.getElementById('btn-upload');
  const fileUploadInput = document.getElementById('file-upload-input');
  const btnNewFolder = document.getElementById('btn-new-folder');
  const folderNameInput = document.getElementById('folder-name-input');
  const folderModalCreate = document.getElementById('folder-modal-create');
  const folderModalCancel = document.getElementById('folder-modal-cancel');
  const folderModalClose = document.getElementById('folder-modal-close');
  const lightboxClose = document.getElementById('lightbox-close');
  const textModalClose = document.getElementById('text-modal-close');
  const textModalCloseBtn = document.getElementById('text-modal-close-btn');
  const videoModalClose = document.getElementById('video-modal-close');
  const videoModalCloseBtn = document.getElementById('video-modal-close-btn');
  const userProfile = document.getElementById('user-profile');
  const btnLogout = document.getElementById('btn-logout');
  const authForm = document.getElementById('auth-form');
  const authSubmit = document.getElementById('auth-submit');

  // Auth Form listeners
  if (authForm) authForm.addEventListener('submit', handleAuthSubmit);
  if (authSubmit) authSubmit.addEventListener('click', handleAuthSubmit);

  // Search filter listener
  if (searchInput) {
    searchInput.addEventListener('input', (e) => {
      state.searchQuery = e.target.value;
      renderWorkspace();
    });
  }

  // View toggle listeners
  if (btnViewGrid) {
    btnViewGrid.addEventListener('click', () => {
      state.viewMode = 'grid';
      btnViewGrid.classList.add('active');
      if (btnViewList) btnViewList.classList.remove('active');
      renderWorkspace();
    });
  }

  if (btnViewList) {
    btnViewList.addEventListener('click', () => {
      state.viewMode = 'list';
      btnViewList.classList.add('active');
      if (btnViewGrid) btnViewGrid.classList.remove('active');
      renderWorkspace();
    });
  }

  // Sort select listener
  if (sortSelect) {
    sortSelect.addEventListener('change', (e) => {
      state.sortBy = e.target.value;
      renderWorkspace();
    });
  }

  // Upload button and input listener with snapshot
  if (btnUpload) {
    btnUpload.addEventListener('click', () => {
      if (fileUploadInput) fileUploadInput.click();
    });
  }

  if (fileUploadInput) {
    fileUploadInput.addEventListener('change', (e) => {
      if (e.target.files && e.target.files.length > 0) {
        const files = Array.from(e.target.files);
        fileUploadInput.value = '';
        handleFileUpload(files);
      }
    });
  }

  // New folder modal listeners
  if (btnNewFolder) btnNewFolder.addEventListener('click', openNewFolderModal);
  if (folderModalCancel) folderModalCancel.addEventListener('click', closeNewFolderModal);
  if (folderModalClose) folderModalClose.addEventListener('click', closeNewFolderModal);
  if (folderModalCreate) {
    folderModalCreate.addEventListener('click', () => {
      if (folderNameInput) handleCreateFolder(folderNameInput.value);
    });
  }

  // Preview modals close listeners
  if (lightboxClose) lightboxClose.addEventListener('click', closeLightbox);
  if (textModalClose) textModalClose.addEventListener('click', closeTextViewer);
  if (textModalCloseBtn) textModalCloseBtn.addEventListener('click', closeTextViewer);
  if (videoModalClose) videoModalClose.addEventListener('click', closeVideoPlayer);
  if (videoModalCloseBtn) videoModalCloseBtn.addEventListener('click', closeVideoPlayer);

  // User profile dropdown and dismissal
  if (userProfile) {
    userProfile.addEventListener('click', (e) => {
      if (e.target.closest('#profile-dropdown')) return;
      toggleProfileDropdown();
    });
  }

  document.addEventListener('click', (e) => {
    if (userProfile && !userProfile.contains(e.target)) {
      closeProfileDropdown();
    }
  });

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      closeProfileDropdown();
    }
  });

  if (btnLogout) {
    btnLogout.addEventListener('click', (e) => {
      e.stopPropagation();
      handleLogout();
    });
  }
}
