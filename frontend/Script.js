// ============================================
//  OXNET — script.js
// ============================================

document.addEventListener('DOMContentLoaded', () => {

  // ---- Populate Device ID & LAN in dropdown ----
  const deviceIdEl = document.getElementById('deviceId');
  if (deviceIdEl) {
    const id = 'OX-' + Math.random().toString(36).substring(2, 6).toUpperCase()
      + '-' + Math.random().toString(36).substring(2, 6).toUpperCase();
    setTimeout(() => { deviceIdEl.textContent = id; }, 300);
  }

  const lanEl = document.getElementById('lanAddress');
  if (lanEl) {
    const randIp = `192.168.${Math.floor(Math.random()*5)}.${Math.floor(Math.random()*200+10)}`;
    setTimeout(() => { lanEl.textContent = randIp; }, 1100);
  }

  // ---- Profile Dropdown ----
  const profileBtn = document.getElementById('profileBtn');
  const profileDropdown = document.getElementById('profileDropdown');
  const profileBtnName = document.getElementById('profileBtnName');
  const profileAvatar = document.getElementById('profileAvatar');
  const profileDropdownAvatar = document.getElementById('profileDropdownAvatar');
  const profileNameInput = document.getElementById('profileNameInput');
  const profileSaveBtn = document.getElementById('profileSaveBtn');

  function getInitial(name) {
    return (name || 'A').trim()[0].toUpperCase();
  }

  function toggleDropdown(forceClose = false) {
    const isOpen = profileDropdown.classList.contains('open');
    if (forceClose || isOpen) {
      profileDropdown.classList.remove('open');
      profileBtn.classList.remove('open');
      profileBtn.setAttribute('aria-expanded', 'false');
    } else {
      profileDropdown.classList.add('open');
      profileBtn.classList.add('open');
      profileBtn.setAttribute('aria-expanded', 'true');
      setTimeout(() => profileNameInput?.focus(), 150);
    }
  }

  profileBtn?.addEventListener('click', (e) => {
    e.stopPropagation();
    toggleDropdown();
  });

  document.addEventListener('click', (e) => {
    if (!document.getElementById('profileMenuWrap')?.contains(e.target)) {
      toggleDropdown(true);
    }
  });

  // Save username
  function saveUsername() {
    const val = profileNameInput.value.trim() || 'Anonymous';
    profileNameInput.value = val;
    profileBtnName.textContent = val;
    profileAvatar.textContent = getInitial(val);
    profileDropdownAvatar.textContent = getInitial(val);
    // Also sync sidebar userName field if it exists
    const sidebarName = document.getElementById('userName');
    if (sidebarName) sidebarName.value = val;
    profileSaveBtn.classList.add('saved');
    setTimeout(() => profileSaveBtn.classList.remove('saved'), 1200);
    showToast(`Username saved — ${val}`);
  }

  profileSaveBtn?.addEventListener('click', saveUsername);
  profileNameInput?.addEventListener('keydown', (e) => {
    if (e.key === 'Enter') saveUsername();
  });

  // ---- Settings Side Panel ----
  const settingsToggle = document.getElementById('settingsToggle');
  const settingsPanel = document.getElementById('settingsPanel');
  const settingsOverlay = document.getElementById('settingsOverlay');
  const settingsClose = document.getElementById('settingsClose');
  const settingsSave = document.getElementById('settingsSave');

  function openSettings() {
    toggleDropdown(true);
    settingsPanel.classList.add('open');
    settingsOverlay.classList.add('open');
    document.body.style.overflow = 'hidden';
  }

  function closeSettings() {
    settingsPanel.classList.remove('open');
    settingsOverlay.classList.remove('open');
    document.body.style.overflow = '';
  }

  settingsToggle?.addEventListener('click', (e) => {
    e.preventDefault();
    openSettings();
  });

  // Close dropdown when clicking About link
  document.getElementById('aboutLink')?.addEventListener('click', () => {
    toggleDropdown(true);
  });

  settingsClose?.addEventListener('click', closeSettings);
  settingsOverlay?.addEventListener('click', closeSettings);

  settingsSave?.addEventListener('click', () => {
    showToast('Settings saved successfully.');
    closeSettings();
  });

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      closeSettings();
      toggleDropdown(true);
    }
  });



  // ---- Create Session Modal ----
  const createBtn = document.getElementById('createSessionBtn');
  const modalOverlay = document.getElementById('modalOverlay');
  const modalClose = document.getElementById('modalClose');

  createBtn?.addEventListener('click', () => {
    modalOverlay.classList.add('open');
  });

  modalClose?.addEventListener('click', () => {
    modalOverlay.classList.remove('open');
  });

  modalOverlay?.addEventListener('click', (e) => {
    if (e.target === modalOverlay) {
      modalOverlay.classList.remove('open');
    }
  });

  document.addEventListener('keydown', (e) => {
    if (e.key === 'Escape') {
      modalOverlay.classList.remove('open');
    }
  });

  // ---- Launch session from modal ----
  const launchBtn = document.querySelector('.btn-create-modal');
  launchBtn?.addEventListener('click', () => {
    const nameInput = document.querySelector('.modal-field input[type="text"]');
    const name = nameInput?.value.trim() || 'New Session';

    // Add a new session card dynamically
    const grid = document.getElementById('sessionsGrid');
    const count = grid.querySelectorAll('.session-row').length + 1;
    const num = String(count).padStart(2, '0');

    const card = document.createElement('div');
    card.classList.add('session-row');
    const userName = document.getElementById('userName')?.value.trim() || 'you';
    card.innerHTML = `
      <div class="session-indicator"></div>
      <div class="session-main-info">
        <span class="session-name">${name}</span>
        <span class="session-host">Host: ${userName}</span>
      </div>
      <div class="session-divider"></div>
      <div class="session-users">
        <span class="session-users-icon">⊹</span>
        <span class="session-users-count">1 user</span>
      </div>
      <button class="btn-join">Join →</button>
    `;

    card.style.opacity = '0';
    card.style.transform = 'scale(0.9)';
    grid.appendChild(card);

    // Animate in
    requestAnimationFrame(() => {
      card.style.transition = 'opacity 0.4s ease, transform 0.4s ease';
      card.style.opacity = '1';
      card.style.transform = 'scale(1)';
    });

    // Update stat counter
    const statDevices = document.getElementById('statDevices');
    if (statDevices) {
      statDevices.textContent = parseInt(statDevices.textContent) + 1;
    }

    modalOverlay.classList.remove('open');

    // Navigate to session page
    setTimeout(() => {
      window.location.href = `session.html?session=${encodeURIComponent(name)}`;
    }, 400);
  });

  // ---- View More Sessions ----
  const viewMoreBtn = document.getElementById('viewMoreBtn');
  const viewMoreLabel = document.getElementById('viewMoreLabel');
  const viewMoreIcon = document.getElementById('viewMoreIcon');
  const scrollWrap = document.getElementById('sessionsScrollWrap');

  let sessionsExpanded = false;

  viewMoreBtn?.addEventListener('click', () => {
    sessionsExpanded = !sessionsExpanded;

    if (sessionsExpanded) {
      scrollWrap.classList.add('expanded');
      viewMoreLabel.textContent = 'Show Less';
      viewMoreBtn.classList.add('active');
      // Smooth scroll to show new sessions
      setTimeout(() => {
        scrollWrap.scrollTo({ top: scrollWrap.scrollHeight, behavior: 'smooth' });
      }, 200);
    } else {
      scrollWrap.classList.remove('expanded');
      viewMoreLabel.textContent = 'View More Sessions';
      viewMoreBtn.classList.remove('active');
      scrollWrap.scrollTo({ top: 0, behavior: 'smooth' });
    }
  });

  // ---- Session card join buttons ----
  document.addEventListener('click', (e) => {
    if (e.target.classList.contains('btn-join')) {
      const card = e.target.closest('.session-card');
      const num = card.querySelector('.session-num')?.textContent;
      showToast(`Joining Session ${num}...`);
    }
  });





  // ---- Toast Notification ----
  function showToast(msg, type = 'info') {
    const existing = document.querySelector('.toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.className = 'toast';
    toast.textContent = msg;
    Object.assign(toast.style, {
      position: 'fixed',
      bottom: '1.5rem',
      left: '50%',
      transform: 'translateX(-50%) translateY(20px)',
      background: type === 'warn' ? '#443A67' : '#211E46',
      color: '#CBC9CC',
      border: '1px solid rgba(110,97,137,0.3)',
      padding: '0.7rem 1.4rem',
      borderRadius: '100px',
      fontFamily: "'Sora', sans-serif",
      fontSize: '0.8rem',
      zIndex: '999',
      opacity: '0',
      transition: 'opacity 0.3s ease, transform 0.3s ease',
      boxShadow: '0 8px 30px rgba(0,0,0,0.5)',
      whiteSpace: 'nowrap',
    });

    document.body.appendChild(toast);

    requestAnimationFrame(() => {
      toast.style.opacity = '1';
      toast.style.transform = 'translateX(-50%) translateY(0)';
    });

    setTimeout(() => {
      toast.style.opacity = '0';
      toast.style.transform = 'translateX(-50%) translateY(10px)';
      setTimeout(() => toast.remove(), 350);
    }, 2500);
  }
});