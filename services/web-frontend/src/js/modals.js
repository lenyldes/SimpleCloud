// SimpleCloud Web UI - Modal Dialog Handlers

/**
 * Open modal element with transition class.
 * @param {HTMLElement} modalEl
 */
function openModal(modalEl) {
  if (!modalEl) return;
  modalEl.classList.remove('hidden');
  setTimeout(() => modalEl.classList.add('open'), 10);
}

/**
 * Close modal element with transition class.
 * @param {HTMLElement} modalEl
 */
function closeModal(modalEl) {
  if (!modalEl) return;
  modalEl.classList.remove('open');
  setTimeout(() => modalEl.classList.add('hidden'), 150);
}

/**
 * Open image lightbox preview modal.
 * @param {object} file
 */
function openLightbox(file) {
  const modalLightbox = document.getElementById('modal-lightbox');
  const lightboxImg = document.getElementById('lightbox-img');
  const lightboxDownload = document.getElementById('lightbox-download');
  const downloadUrl = `/api/v1/files/download/${file.id}`;

  if (lightboxImg) lightboxImg.src = downloadUrl;
  if (lightboxDownload) lightboxDownload.href = downloadUrl;
  openModal(modalLightbox);
}

/**
 * Close image lightbox preview modal.
 */
function closeLightbox() {
  const modalLightbox = document.getElementById('modal-lightbox');
  closeModal(modalLightbox);
}

/**
 * Open plain text viewer modal and fetch file contents.
 * @param {object} file
 */
async function openTextViewer(file) {
  const modalText = document.getElementById('modal-text');
  const textModalTitle = document.getElementById('text-modal-title');
  const textModalContent = document.getElementById('text-modal-content');
  const textModalDownload = document.getElementById('text-modal-download');
  const downloadUrl = `/api/v1/files/download/${file.id}`;

  if (textModalTitle) textModalTitle.textContent = file.filename;
  if (textModalDownload) textModalDownload.href = downloadUrl;
  if (textModalContent) textModalContent.textContent = 'Loading file content...';
  openModal(modalText);

  try {
    const res = typeof fetchWithAuth === 'function' ? await fetchWithAuth(downloadUrl) : await fetch(downloadUrl);
    if (res.ok) {
      const text = await res.text();
      if (textModalContent) textModalContent.textContent = text;
    } else {
      if (textModalContent) textModalContent.textContent = 'Failed to load text content.';
    }
  } catch (err) {
    if (textModalContent) textModalContent.textContent = 'Error reading file content.';
  }
}

/**
 * Close plain text viewer modal.
 */
function closeTextViewer() {
  const modalText = document.getElementById('modal-text');
  closeModal(modalText);
}

/**
 * Open HTML5 video player modal.
 * @param {object} file
 */
function openVideoPlayer(file) {
  const modalVideo = document.getElementById('modal-video');
  const videoModalTitle = document.getElementById('video-modal-title');
  const videoPlayer = document.getElementById('video-player');
  const videoModalDownload = document.getElementById('video-modal-download');
  const downloadUrl = `/api/v1/files/download/${file.id}`;

  if (videoModalTitle) videoModalTitle.textContent = file.filename;
  if (videoPlayer) videoPlayer.src = downloadUrl;
  if (videoModalDownload) videoModalDownload.href = downloadUrl;
  openModal(modalVideo);
}

/**
 * Close HTML5 video player modal and pause playback.
 */
function closeVideoPlayer() {
  const modalVideo = document.getElementById('modal-video');
  const videoPlayer = document.getElementById('video-player');
  if (videoPlayer) videoPlayer.pause();
  closeModal(modalVideo);
}

/**
 * Open dialog to create a new folder.
 */
function openNewFolderModal() {
  const modalNewFolder = document.getElementById('modal-new-folder');
  const folderNameInput = document.getElementById('folder-name-input');
  openModal(modalNewFolder);
  if (folderNameInput) folderNameInput.focus();
}

/**
 * Close new folder dialog and reset input.
 */
function closeNewFolderModal() {
  const modalNewFolder = document.getElementById('modal-new-folder');
  const folderNameInput = document.getElementById('folder-name-input');
  closeModal(modalNewFolder);
  if (folderNameInput) folderNameInput.value = '';
}

/**
 * Submit folder creation request.
 * @param {string} folderName
 */
async function handleCreateFolder(folderName) {
  if (!folderName || !folderName.trim()) return;
  const modalNewFolder = document.getElementById('modal-new-folder');
  const folderNameInput = document.getElementById('folder-name-input');

  try {
    const res = window.api && window.api.createFolder
      ? await window.api.createFolder(folderName, state.currentFolderId)
      : await fetchWithAuth('/api/v1/folders', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: folderName.trim(),
            parent_id: state.currentFolderId
          })
        });

    if (res.ok) {
      const newFolder = await res.json();
      if (Array.isArray(state.folders)) {
        state.folders.push(newFolder);
      }
      if (Array.isArray(state.allFolders)) {
        state.allFolders.push(newFolder);
      }
      closeModal(modalNewFolder);
      if (folderNameInput) folderNameInput.value = '';
      if (typeof showToast === 'function') showToast(`Folder "${folderName}" created`, 'success');
      if (typeof loadWorkspaceData === 'function') await loadWorkspaceData();
    } else {
      const errData = await res.json().catch(() => ({}));
      if (typeof showToast === 'function') showToast(errData.error || 'Failed to create folder', 'danger');
    }
  } catch (err) {
    console.error('Create folder error:', err);
    if (typeof showToast === 'function') showToast('Failed to create folder', 'danger');
  }
}

/**
 * Route file click to the appropriate viewer modal based on file extension.
 * @param {object} file
 */
async function handleFileClick(file) {
  const downloadUrl = `/api/v1/files/download/${file.id}`;

  if (typeof isImage === 'function' && isImage(file.filename)) {
    openLightbox(file);
  } else if (typeof isVideo === 'function' && isVideo(file.filename)) {
    openVideoPlayer(file);
  } else if (typeof isText === 'function' && isText(file.filename)) {
    await openTextViewer(file);
  } else {
    window.open(downloadUrl, '_blank');
  }
}

/**
 * Close top active modal dialog on Escape key press.
 * @returns {boolean} Whether an active modal was closed
 */
function closeTopModal() {
  const modalConfirmDelete = document.getElementById('modal-confirm-delete');
  const modalNewFolder = document.getElementById('modal-new-folder');
  const modalLightbox = document.getElementById('modal-lightbox');
  const modalText = document.getElementById('modal-text');
  const modalVideo = document.getElementById('modal-video');

  if (modalConfirmDelete && modalConfirmDelete.classList.contains('open')) {
    if (isDeleting) return false;
    closeConfirmDeleteModal();
    return true;
  }
  if (modalNewFolder && modalNewFolder.classList.contains('open')) {
    closeNewFolderModal();
    return true;
  }
  if (modalLightbox && modalLightbox.classList.contains('open')) {
    closeLightbox();
    return true;
  }
  if (modalText && modalText.classList.contains('open')) {
    closeTextViewer();
    return true;
  }
  if (modalVideo && modalVideo.classList.contains('open')) {
    closeVideoPlayer();
    return true;
  }
  return false;
}

// Module state for pending item to delete and in-flight request lock
let pendingDeleteItem = null;
let isDeleting = false;

/**
 * Open confirmation modal to delete a file or folder.
 * @param {string} id
 * @param {'file'|'folder'} type
 * @param {string} name
 */
function openConfirmDeleteModal(id, type, name) {
  pendingDeleteItem = { id, type, name };
  const modal = document.getElementById('modal-confirm-delete');
  const title = document.getElementById('confirm-delete-title');
  const targetNameEl = document.getElementById('confirm-delete-target-name');
  const folderWarningEl = document.getElementById('confirm-delete-folder-warning');
  const cancelBtn = document.getElementById('confirm-delete-cancel');

  if (title) {
    title.textContent = type === 'folder' ? 'Delete Folder' : 'Delete File';
  }
  if (targetNameEl) {
    targetNameEl.textContent = name;
  }
  if (folderWarningEl) {
    if (type === 'folder') {
      folderWarningEl.classList.remove('hidden');
    } else {
      folderWarningEl.classList.add('hidden');
    }
  }
  openModal(modal);
  if (cancelBtn) {
    cancelBtn.focus();
  }
}

/**
 * Close confirmation modal for delete and clear pending item.
 */
function closeConfirmDeleteModal() {
  if (isDeleting) return;
  const modal = document.getElementById('modal-confirm-delete');
  closeModal(modal);
  pendingDeleteItem = null;
}

/**
 * Get current pending deletion item.
 * @returns {{ id: string, type: string, name: string }|null}
 */
function getPendingDeleteItem() {
  return pendingDeleteItem;
}

/**
 * Handle confirmation of file or folder deletion.
 */
async function handleConfirmDelete() {
  if (isDeleting) return;
  const item = typeof getPendingDeleteItem === 'function' ? getPendingDeleteItem() : pendingDeleteItem;
  if (!item || !item.id) {
    closeConfirmDeleteModal();
    return;
  }

  const { id, type, name } = item;
  const isFolder = type === 'folder';
  const confirmDeleteBtn = document.getElementById('confirm-delete-btn');
  const confirmDeleteCancel = document.getElementById('confirm-delete-cancel');
  const confirmDeleteClose = document.getElementById('confirm-delete-close');

  isDeleting = true;
  if (confirmDeleteBtn) {
    confirmDeleteBtn.disabled = true;
    confirmDeleteBtn.textContent = 'Deleting...';
  }
  if (confirmDeleteCancel) confirmDeleteCancel.disabled = true;
  if (confirmDeleteClose) confirmDeleteClose.disabled = true;

  try {
    const res = isFolder
      ? (window.api && window.api.deleteFolder ? await window.api.deleteFolder(id) : await fetchWithAuth(`/api/v1/folders/${encodeURIComponent(id)}`, { method: 'DELETE' }))
      : (window.api && window.api.deleteFile ? await window.api.deleteFile(id) : await fetchWithAuth(`/api/v1/files/${encodeURIComponent(id)}`, { method: 'DELETE' }));

    if (res && res.ok) {
      isDeleting = false;
      closeConfirmDeleteModal();
      if (typeof showToast === 'function') showToast(`${isFolder ? 'Folder' : 'File'} "${name}" deleted successfully`, 'success');
      if (typeof loadWorkspaceData === 'function') await loadWorkspaceData();
    } else {
      const errData = res ? await res.json().catch(() => ({})) : {};
      if (typeof showToast === 'function') showToast(errData.error || `Failed to delete ${isFolder ? 'folder' : 'file'}`, 'danger');
    }
  } catch (err) {
    console.error('Delete error:', err);
    if (typeof showToast === 'function') showToast(`Failed to delete ${isFolder ? 'folder' : 'file'}`, 'danger');
  } finally {
    isDeleting = false;
    if (confirmDeleteBtn) {
      confirmDeleteBtn.disabled = false;
      confirmDeleteBtn.textContent = 'Delete';
    }
    if (confirmDeleteCancel) confirmDeleteCancel.disabled = false;
    if (confirmDeleteClose) confirmDeleteClose.disabled = false;
  }
}

if (typeof window !== 'undefined') {
  window.closeTopModal = closeTopModal;
  window.handleConfirmDelete = handleConfirmDelete;
}


