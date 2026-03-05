/**
 * OXNET — session.js
 * All session page logic:
 *   - Clock + duration timer
 *   - Mic / Camera toggle
 *   - Screen share toggle
 *   - Right panel (Chat / Files / Members) slide-in/out
 *   - Chat send + simulated replies
 *   - File drag-and-drop share
 *   - Pre-recorded video loader
 *   - Hang-up modal + navigation
 */

'use strict';

document.addEventListener('DOMContentLoaded', () => {

  // ════════════════════════════════════
  //  SESSION NAME (from URL param)
  // ════════════════════════════════════
  const params = new URLSearchParams(window.location.search);
  const SESSION_NAME = params.get('session') || 'Team Alpha';
  const SESSION_ROLE = params.get('role')    || 'host'; // 'host' | 'guest'

  document.getElementById('sessionBadgeName').textContent = SESSION_NAME;
  document.getElementById('hangupLabel').textContent      = SESSION_NAME;
  document.getElementById('roleBadge').textContent        = SESSION_ROLE === 'host' ? 'HOST' : 'GUEST';

  // ════════════════════════════════════
  //  CLOCK
  // ════════════════════════════════════
  function formatTime(date) {
    let h    = date.getHours();
    const m  = String(date.getMinutes()).padStart(2, '0');
    const ap = h >= 12 ? 'PM' : 'AM';
    h        = h % 12 || 12;
    return `${String(h).padStart(2, '0')}:${m} ${ap}`;
  }

  function tickClock() {
    const t = formatTime(new Date());
    document.getElementById('topbarClock').textContent = t;
    document.getElementById('cbClock').textContent     = t;
  }

  tickClock();
  setInterval(tickClock, 1000);

  // ════════════════════════════════════
  //  SESSION DURATION
  // ════════════════════════════════════
  let elapsed = 0;
  const durationEl = document.getElementById('sessionDuration');

  setInterval(() => {
    elapsed++;
    const mm = String(Math.floor(elapsed / 60)).padStart(2, '0');
    const ss = String(elapsed % 60).padStart(2, '0');
    durationEl.textContent = `${mm}:${ss}`;
  }, 1000);

  // ════════════════════════════════════
  //  MIC TOGGLE
  // ════════════════════════════════════
  let micOn = true;
  const btnMic       = document.getElementById('btnMic');
  const micIcon      = document.getElementById('micIcon');
  const selfTileMic  = document.getElementById('selfTileMic');

  btnMic.addEventListener('click', () => {
    micOn = !micOn;
    btnMic.classList.toggle('active', micOn);
    btnMic.classList.toggle('off',    !micOn);
    micIcon.textContent     = micOn ? '🎤' : '🔇';
    selfTileMic.textContent = micOn ? '🎤' : '🔇';
    toast(micOn ? 'Microphone on' : 'Microphone muted');
  });

  // ════════════════════════════════════
  //  CAMERA TOGGLE
  // ════════════════════════════════════
  let camOn = true;
  const btnCamera = document.getElementById('btnCamera');
  const camIcon   = document.getElementById('camIcon');

  btnCamera.addEventListener('click', () => {
    camOn = !camOn;
    btnCamera.classList.toggle('active', camOn);
    btnCamera.classList.toggle('off',    !camOn);
    camIcon.textContent = camOn ? '📷' : '📷';
    toast(camOn ? 'Camera on' : 'Camera off');
  });

  // ════════════════════════════════════
  //  SCREEN SHARE TOGGLE
  // ════════════════════════════════════
  let screenOn = false;
  const btnScreen  = document.getElementById('btnScreen');
  const liveSubEl  = document.getElementById('liveSubText');

  btnScreen.addEventListener('click', () => {
    screenOn = !screenOn;
    btnScreen.classList.toggle('active', screenOn);
    btnScreen.classList.toggle('off',    !screenOn);
    liveSubEl.textContent = screenOn
      ? 'Broadcasting screen · LAN only'
      : 'LAN only · Waiting for stream';
    toast(screenOn ? 'Screen sharing started' : 'Screen sharing stopped');
  });

  // ════════════════════════════════════
  //  PRE-RECORDED VIDEO
  // ════════════════════════════════════
  const btnPrerecorded  = document.getElementById('btnPrerecorded');
  const videoModal      = document.getElementById('videoModal');
  const btnVideoCancel  = document.getElementById('btnVideoCancel');
  const btnVideoBrowse  = document.getElementById('btnVideoBrowse');
  const videoFileInput  = document.getElementById('videoFileInput');
  const prerecordedWrap = document.getElementById('prerecordedWrap');
  const prerecordedVid  = document.getElementById('prerecordedVideo');
  const prerecordedEmp  = document.getElementById('prerecordedEmpty');
  const livePlaceholder = document.getElementById('livePlaceholder');
  let videoLoaded = false;

  btnPrerecorded.addEventListener('click', () => openModal(videoModal));
  btnVideoCancel.addEventListener('click', () => closeModal(videoModal));

  btnVideoBrowse.addEventListener('click', () => videoFileInput.click());

  videoFileInput.addEventListener('change', () => {
    const file = videoFileInput.files[0];
    if (!file) return;

    const url = URL.createObjectURL(file);
    prerecordedVid.src = url;
    prerecordedEmp.style.display = 'none';
    videoLoaded = true;

    // Switch main area to video player
    livePlaceholder.style.display  = 'none';
    prerecordedWrap.classList.remove('hidden');
    btnPrerecorded.classList.add('active');

    closeModal(videoModal);
    toast(`Playing: ${file.name}`);

    videoFileInput.value = ''; // reset input
  });

  // ════════════════════════════════════
  //  RIGHT PANEL — open / close / switch
  // ════════════════════════════════════
  const rightPanel    = document.getElementById('rightPanel');
  const rpTitle       = document.getElementById('rpTitle');
  const rpClose       = document.getElementById('rpClose');
  const rpSections    = {
    chat:    document.getElementById('rpChat'),
    files:   document.getElementById('rpFiles'),
    members: document.getElementById('rpMembers'),
  };
  const panelBtns = document.querySelectorAll('.panel-cbtn');

  let activePanel = null; // 'chat' | 'files' | 'members' | null

  const PANEL_LABELS = { chat: 'Chat', files: 'File Sharing', members: 'Members' };

  function openPanel(name) {
    // Hide all sections
    Object.values(rpSections).forEach(s => s.classList.add('hidden'));

    // Show requested
    rpSections[name].classList.remove('hidden');
    rpTitle.textContent = PANEL_LABELS[name];
    rightPanel.classList.add('open');
    activePanel = name;

    // Highlight button
    panelBtns.forEach(btn => {
      btn.classList.toggle('panel-active', btn.dataset.panel === name);
    });

    // Clear chat notif
    if (name === 'chat') {
      document.getElementById('chatNotif').classList.add('hidden');
    }
  }

  function closePanel() {
    rightPanel.classList.remove('open');
    activePanel = null;
    panelBtns.forEach(btn => btn.classList.remove('panel-active'));
  }

  panelBtns.forEach(btn => {
    btn.addEventListener('click', () => {
      const panel = btn.dataset.panel;
      if (activePanel === panel) {
        closePanel(); // toggle off
      } else {
        openPanel(panel);
      }
    });
  });

  rpClose.addEventListener('click', closePanel);

  // ════════════════════════════════════
  //  CHAT
  // ════════════════════════════════════
  const chatMessages  = document.getElementById('chatMessages');
  const chatInput     = document.getElementById('chatInput');
  const chatSendBtn   = document.getElementById('chatSendBtn');
  const chatNotif     = document.getElementById('chatNotif');

  const FAKE_NAMES    = ['alex_dev', 'maya_k'];
  const FAKE_REPLIES  = [
    'Got it! 👍', 'Makes sense.', 'On it!', 'Noted.', 'Sounds good!',
    'Working on that now.', '🤙', 'Copy that.', 'Agreed!', 'Let me check.'
  ];

  function addMessage({ name, text, self = false }) {
    const now  = formatTime(new Date());
    const msg  = document.createElement('div');
    msg.className = `cmsg ${self ? 'out' : 'in'}`;

    const avLetter = name[0].toUpperCase();
    const avClass  = self ? 'cav self' : 'cav';

    if (self) {
      msg.innerHTML = `
        <div class="cbody">
          <div class="cmeta"><span class="cname">${escHtml(name)}</span><span class="ctime">${now}</span></div>
          <div class="cbubble">${escHtml(text)}</div>
        </div>
        <div class="${avClass}">${avLetter}</div>`;
    } else {
      msg.innerHTML = `
        <div class="${avClass}">${avLetter}</div>
        <div class="cbody">
          <div class="cmeta"><span class="cname">${escHtml(name)}</span><span class="ctime">${now}</span></div>
          <div class="cbubble">${escHtml(text)}</div>
        </div>`;
    }

    // Animate in
    msg.style.cssText = 'opacity:0;transform:translateY(6px)';
    chatMessages.appendChild(msg);
    requestAnimationFrame(() => {
      msg.style.transition = 'opacity .24s ease, transform .24s ease';
      msg.style.opacity    = '1';
      msg.style.transform  = 'translateY(0)';
    });
    chatMessages.scrollTop = chatMessages.scrollHeight;

    // Show notif dot if panel not open
    if (activePanel !== 'chat' && !self) {
      chatNotif.classList.remove('hidden');
    }
  }

  function sendMessage() {
    const text = chatInput.value.trim();
    if (!text) return;

    addMessage({ name: 'You', text, self: true });
    chatInput.value = '';

    // Occasional simulated reply
    if (Math.random() > 0.4) {
      const delay = 900 + Math.random() * 1200;
      setTimeout(() => {
        const name  = FAKE_NAMES[Math.floor(Math.random() * FAKE_NAMES.length)];
        const reply = FAKE_REPLIES[Math.floor(Math.random() * FAKE_REPLIES.length)];
        addMessage({ name, text: reply });
      }, delay);
    }
  }

  chatSendBtn.addEventListener('click', sendMessage);
  chatInput.addEventListener('keydown', e => { if (e.key === 'Enter') sendMessage(); });

  // ════════════════════════════════════
  //  FILE SHARING
  // ════════════════════════════════════
  const fileDropZone = document.getElementById('fileDropZone');
  const fileInput    = document.getElementById('fileInput');
  const fileList     = document.getElementById('fileList');

  fileDropZone.addEventListener('click', () => fileInput.click());

  fileDropZone.addEventListener('dragover', e => {
    e.preventDefault();
    fileDropZone.classList.add('drag');
  });
  fileDropZone.addEventListener('dragleave', () => fileDropZone.classList.remove('drag'));
  fileDropZone.addEventListener('drop', e => {
    e.preventDefault();
    fileDropZone.classList.remove('drag');
    handleFiles(e.dataTransfer.files);
  });

  fileInput.addEventListener('change', () => {
    handleFiles(fileInput.files);
    fileInput.value = '';
  });

  function handleFiles(files) {
    Array.from(files).forEach(file => {
      const kb   = (file.size / 1024).toFixed(0);
      const size = kb > 1024 ? `${(kb / 1024).toFixed(1)} MB` : `${kb} KB`;
      const ext  = file.name.split('.').pop().toLowerCase();
      const ico  = ['png','jpg','jpeg','gif','webp','svg'].includes(ext) ? '🖼' :
                   ['mp4','mov','avi','mkv'].includes(ext)                ? '🎬' :
                   ['pdf'].includes(ext)                                  ? '📄' :
                   ['zip','rar','7z'].includes(ext)                       ? '📦' : '📎';

      const row = document.createElement('div');
      row.className = 'file-row';
      row.innerHTML = `
        <span class="ficon">${ico}</span>
        <div class="finfo">
          <span class="fname">${escHtml(file.name)}</span>
          <span class="fmeta">You · ${size}</span>
        </div>
        <button class="fdl">↓</button>`;

      row.style.cssText = 'opacity:0;transform:translateX(-6px)';
      fileList.appendChild(row);
      requestAnimationFrame(() => {
        row.style.transition = 'opacity .3s ease, transform .3s ease';
        row.style.opacity    = '1';
        row.style.transform  = 'translateX(0)';
      });
    });

    if (files.length) toast(`${files.length} file(s) shared`);
  }

  // ════════════════════════════════════
  //  HANG UP
  // ════════════════════════════════════
  const hangupModal = document.getElementById('hangupModal');
  const btnHangup   = document.getElementById('btnHangup');
  const btnStay     = document.getElementById('btnStay');
  const btnLeave    = document.getElementById('btnLeave');

  btnHangup.addEventListener('click', () => openModal(hangupModal));
  btnStay.addEventListener('click',   () => closeModal(hangupModal));

  hangupModal.addEventListener('click', e => {
    if (e.target === hangupModal) closeModal(hangupModal);
  });

  btnLeave.addEventListener('click', () => {
    btnLeave.textContent = 'Leaving…';
    setTimeout(() => {
      // Electron-compatible navigation back to home page
      window.location.href = 'index.html';
    }, 600);
  });

  // ════════════════════════════════════
  //  BACK HOME (logo button)
  // ════════════════════════════════════
  document.getElementById('btnBackHome').addEventListener('click', () => {
    openModal(hangupModal); // ask to confirm
  });

  // ════════════════════════════════════
  //  KEYBOARD SHORTCUTS
  // ════════════════════════════════════
  document.addEventListener('keydown', e => {
    if (e.key === 'Escape') {
      closeModal(hangupModal);
      closeModal(videoModal);
    }
    // M = mic, C = camera
    if (e.target.tagName !== 'INPUT') {
      if (e.key === 'm' || e.key === 'M') btnMic.click();
      if (e.key === 'c' || e.key === 'C') btnCamera.click();
    }
  });

  // ════════════════════════════════════
  //  MODAL HELPERS
  // ════════════════════════════════════
  function openModal(el)  { el.classList.add('open'); }
  function closeModal(el) { el.classList.remove('open'); }

  // ════════════════════════════════════
  //  TOAST
  // ════════════════════════════════════
  function toast(msg) {
    document.querySelectorAll('.ox-toast').forEach(t => t.remove());

    const t = document.createElement('div');
    t.className = 'ox-toast';
    t.textContent = msg;

    Object.assign(t.style, {
      position:      'fixed',
      bottom:        '96px',
      left:          '50%',
      transform:     'translateX(-50%) translateY(8px)',
      background:    'var(--c5)',
      color:         'var(--c1)',
      border:        '1px solid rgba(110,97,137,.3)',
      padding:       '.55rem 1.35rem',
      borderRadius:  '100px',
      fontFamily:    "'Sora', sans-serif",
      fontSize:      '.76rem',
      zIndex:        '999',
      opacity:       '0',
      transition:    'opacity .26s ease, transform .26s ease',
      boxShadow:     '0 8px 28px rgba(0,0,0,.5)',
      whiteSpace:    'nowrap',
      pointerEvents: 'none',
    });

    document.body.appendChild(t);
    requestAnimationFrame(() => {
      t.style.opacity   = '1';
      t.style.transform = 'translateX(-50%) translateY(0)';
    });

    setTimeout(() => {
      t.style.opacity   = '0';
      t.style.transform = 'translateX(-50%) translateY(6px)';
      setTimeout(() => t.remove(), 300);
    }, 2200);
  }

  // ════════════════════════════════════
  //  HTML ESCAPE UTILITY
  // ════════════════════════════════════
  function escHtml(str) {
    return str
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;');
  }

}); // end DOMContentLoaded