// SimpleCloud Web UI - Authentication & Session Management

/**
 * Verify user session and update application state and avatar.
 */
async function checkAuth() {
  try {
    const res = window.api ? await window.api.getMe() : await fetch('/api/v1/auth/me', { credentials: 'include' });
    if (res.ok) {
      const user = await res.json();
      state.user = user;
      if (state.user) {
        if (typeof state.user.used_bytes === 'number') state.quota.used = state.user.used_bytes;
        if (typeof state.user.quota_bytes === 'number') state.quota.total = state.user.quota_bytes;
      }
      updateUserAvatar();
    } else if (res.status === 401) {
      showAuthModal();
    }
  } catch (err) {
    console.warn('Auth check warning:', err);
  }
}

/**
 * Display the authentication modal dialog.
 */
function showAuthModal() {
  const modalAuth = document.getElementById('modal-auth');
  const authEmail = document.getElementById('auth-email');
  if (modalAuth) {
    modalAuth.classList.remove('hidden');
    setTimeout(() => modalAuth.classList.add('open'), 10);
    if (authEmail) authEmail.focus();
  }
}

/**
 * Dismiss the authentication modal dialog.
 */
function hideAuthModal() {
  const modalAuth = document.getElementById('modal-auth');
  const authError = document.getElementById('auth-error');
  if (modalAuth) {
    modalAuth.classList.remove('open');
    setTimeout(() => modalAuth.classList.add('hidden'), 150);
  }
  if (authError) authError.classList.add('hidden');
}

/**
 * Update the user avatar icon and dropdown email display.
 */
function updateUserAvatar() {
  const avatar = document.getElementById('user-avatar');
  const emailEl = document.getElementById('profile-dropdown-email');
  if (state.user && state.user.email) {
    if (avatar) avatar.textContent = state.user.email[0].toUpperCase();
    if (emailEl) emailEl.textContent = state.user.email;
  } else {
    if (avatar) avatar.textContent = 'U';
    if (emailEl) emailEl.textContent = '';
  }
}

/**
 * Toggle visibility of user profile dropdown menu.
 */
function toggleProfileDropdown() {
  const profileDropdown = document.getElementById('profile-dropdown');
  if (!profileDropdown) return;
  if (profileDropdown.classList.contains('open')) {
    closeProfileDropdown();
  } else {
    openProfileDropdown();
  }
}

/**
 * Open user profile dropdown menu.
 */
function openProfileDropdown() {
  const profileDropdown = document.getElementById('profile-dropdown');
  const userProfile = document.getElementById('user-profile');
  if (!profileDropdown) return;
  profileDropdown.style.display = 'flex';
  profileDropdown.classList.add('open');
  if (userProfile) userProfile.classList.add('active');
}

/**
 * Close user profile dropdown menu.
 */
function closeProfileDropdown() {
  const profileDropdown = document.getElementById('profile-dropdown');
  const userProfile = document.getElementById('user-profile');
  if (!profileDropdown) return;
  profileDropdown.style.display = 'none';
  profileDropdown.classList.remove('open');
  if (userProfile) userProfile.classList.remove('active');
}

/**
 * Handle user logout, purge local state, and reset UI.
 */
async function handleLogout() {
  const btnLogout = document.getElementById('btn-logout');
  if (btnLogout) btnLogout.disabled = true;
  try {
    if (window.api && window.api.logout) {
      await window.api.logout();
    } else {
      await fetch('/api/v1/auth/logout', {
        method: 'POST',
        credentials: 'include'
      });
    }
  } catch (err) {
    console.error('Logout error:', err);
  } finally {
    if (btnLogout) btnLogout.disabled = false;
    closeProfileDropdown();

    // Client state purge
    state.user = null;
    state.files = [];
    state.folders = [];
    state.allFolders = [];
    state.currentFolderId = null;
    state.breadcrumbs = [{ id: null, name: 'All Files' }];
    state.quota.used = 0;
    state.searchQuery = '';
    const searchInput = document.getElementById('search-input');
    if (searchInput) searchInput.value = '';
    if (typeof resetRoutingState === 'function') resetRoutingState();
    window.location.hash = '#/';

    updateUserAvatar();
    if (typeof updateQuotaDisplay === 'function') updateQuotaDisplay();
    if (typeof renderBreadcrumbs === 'function') renderBreadcrumbs();
    if (typeof renderWorkspace === 'function') renderWorkspace();

    showAuthModal();
    if (typeof showToast === 'function') showToast('You have been logged out', 'info');
  }
}

/**
 * Handle user authentication form submit.
 * @param {Event} [e]
 */
async function handleAuthSubmit(e) {
  if (e && e.preventDefault) e.preventDefault();
  const authEmail = document.getElementById('auth-email');
  const authPassword = document.getElementById('auth-password');
  const authError = document.getElementById('auth-error');
  const email = authEmail ? authEmail.value : '';
  const password = authPassword ? authPassword.value : '';

  if (!email || !password) return;

  try {
    if (authError) authError.classList.add('hidden');
    const res = window.api && window.api.login
      ? await window.api.login(email, password)
      : await fetch('/api/v1/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          credentials: 'include',
          body: JSON.stringify({ email, password })
        });

    if (res.ok) {
      const data = await res.json().catch(() => ({}));
      state.user = data.user || data;
      if (state.user) {
        if (typeof state.user.used_bytes === 'number') state.quota.used = state.user.used_bytes;
        if (typeof state.user.quota_bytes === 'number') state.quota.total = state.user.quota_bytes;
      }
      updateUserAvatar();
      hideAuthModal();
      if (authPassword) authPassword.value = '';
      if (typeof showToast === 'function') showToast('Successfully logged in', 'success');
      if (typeof loadWorkspaceData === 'function') await loadWorkspaceData();
    } else {
      const errData = await res.json().catch(() => ({}));
      if (authError) {
        authError.textContent = errData.error || 'Invalid email or password';
        authError.classList.remove('hidden');
      }
    }
  } catch (err) {
    console.error('Login error:', err);
    if (authError) {
      authError.textContent = 'Network error during login';
      authError.classList.remove('hidden');
    }
  }
}

/**
 * Backward-compatible wrapper for login handling.
 * @param {string} email
 * @param {string} password
 */
async function handleLogin(email, password) {
  const authEmail = document.getElementById('auth-email');
  const authPassword = document.getElementById('auth-password');
  if (authEmail && email !== undefined) authEmail.value = email;
  if (authPassword && password !== undefined) authPassword.value = password;
  return handleAuthSubmit();
}
