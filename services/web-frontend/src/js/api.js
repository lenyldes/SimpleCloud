// SimpleCloud Web UI - API Client & Network Methods

/**
 * Enhanced fetch wrapper that attaches credentials and intercepts 401 Unauthorized responses.
 * @param {string} url - The URL to request.
 * @param {RequestInit} [options={}] - Request options.
 * @returns {Promise<Response>}
 */
async function fetchWithAuth(url, options = {}) {
  options.credentials = 'include';
  const res = await fetch(url, options);
  if (res.status === 401) {
    if (typeof showAuthModal === 'function') {
      showAuthModal();
    }
  }
  return res;
}

/**
 * Fetch current authenticated user session.
 * @returns {Promise<Response>}
 */
async function apiGetMe() {
  return fetch('/api/v1/auth/me', { credentials: 'include' });
}

/**
 * Authenticate user with email and password.
 * @param {string} email
 * @param {string} password
 * @returns {Promise<Response>}
 */
async function apiLogin(email, password) {
  return fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ email, password })
  });
}

/**
 * Terminate user session.
 * @returns {Promise<Response>}
 */
async function apiLogout() {
  return fetch('/api/v1/auth/logout', {
    method: 'POST',
    credentials: 'include'
  });
}

/**
 * List files for the authenticated user, optionally filtered by folder.
 * @param {string|null} [folderId=null]
 * @returns {Promise<Response>}
 */
async function apiListFiles(folderId = null) {
  const url = folderId ? `/api/v1/files?folder_id=${encodeURIComponent(folderId)}` : '/api/v1/files';
  return fetchWithAuth(url);
}

/**
 * Upload a file via FormData.
 * @param {FormData} formData
 * @returns {Promise<Response>}
 */
async function apiUploadFile(formData) {
  return fetchWithAuth('/api/v1/files/upload', {
    method: 'POST',
    body: formData
  });
}

/**
 * Delete a file by ID.
 * @param {string} fileId
 * @returns {Promise<Response>}
 */
async function apiDeleteFile(fileId) {
  return fetchWithAuth(`/api/v1/files/${encodeURIComponent(fileId)}`, {
    method: 'DELETE'
  });
}

/**
 * List folders for the authenticated user.
 * @param {string|null} [parentId=null]
 * @returns {Promise<Response>}
 */
async function apiListFolders(parentId = null) {
  const url = parentId ? `/api/v1/folders?parent_id=${encodeURIComponent(parentId)}` : '/api/v1/folders';
  return fetchWithAuth(url);
}

/**
 * List all folders for the authenticated user across all levels.
 * @returns {Promise<Response>}
 */
async function apiListAllFolders() {
  return fetchWithAuth('/api/v1/folders?all=true');
}

/**
 * Create a new folder.
 * @param {string} name
 * @param {string|null} [parentId=null]
 * @returns {Promise<Response>}
 */
async function apiCreateFolder(name, parentId = null) {
  return fetchWithAuth('/api/v1/folders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      name: name.trim(),
      parent_id: parentId
    })
  });
}

/**
 * Delete a folder by ID.
 * @param {string} folderId
 * @returns {Promise<Response>}
 */
async function apiDeleteFolder(folderId) {
  return fetchWithAuth(`/api/v1/folders/${encodeURIComponent(folderId)}`, {
    method: 'DELETE'
  });
}

// Expose on window object for modular access
window.api = {
  fetchWithAuth,
  getMe: apiGetMe,
  login: apiLogin,
  logout: apiLogout,
  listFiles: apiListFiles,
  uploadFile: apiUploadFile,
  deleteFile: apiDeleteFile,
  listFolders: apiListFolders,
  listAllFolders: apiListAllFolders,
  createFolder: apiCreateFolder,
  deleteFolder: apiDeleteFolder
};
